package kospell

import (
	"context"
	"errors"
	htmlstd "html"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"unicode/utf8"

	internalhanspell "github.com/Alfex4936/kospell/internal/hanspell"
	"github.com/Alfex4936/kospell/internal/model"
	"github.com/Alfex4936/kospell/internal/util"
)

const hanspellMaxRunesPerChunk = 300

var (
	reOriginUnderline = regexp.MustCompile(`(?is)<span[^>]*class=['"]result_underline['"][^>]*>(.*?)</span>`)
	reCorrectedText   = regexp.MustCompile(`(?is)<em[^>]*class=['"]([^'"]+)['"][^>]*>(.*?)</em>`)
	reHTMLTag         = regexp.MustCompile(`(?is)<[^>]+>`)
)

type hanspellChunkResult struct {
	idx       int
	input     string
	corrected string
	items     []model.Correction
}

type correctedSegment struct {
	Class string
	Text  string
}

// CheckHanspell checks text using the Naver(py-hanspell style) backend.
func CheckHanspell(ctx context.Context, text string, c *internalhanspell.Checker) (*model.Result, error) {
	text = strings.TrimSpace(text)
	if ctx == nil {
		return nil, errors.New("ctx is nil")
	}
	if c == nil {
		return nil, errors.New("hanspell checker is nil")
	}

	parts := splitHanspellChunks(text)
	out := make([]hanspellChunkResult, len(parts))

	sem := make(chan struct{}, cap(make([]byte, 0, runtime.GOMAXPROCS(0))))
	var wg sync.WaitGroup
	var firstErr error
	var errOnce sync.Once

	for i, p := range parts {
		i, p := i, p
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			raw, err := c.Check(ctx, p)
			if err != nil {
				errOnce.Do(func() { firstErr = err })
				return
			}

			items := buildHanspellCorrections(p, raw.OriginHTML, raw.HTML)
			corrected := raw.Corrected
			if corrected == "" {
				corrected = applyCorrections(p, items)
			}

			out[i] = hanspellChunkResult{
				idx:       i,
				input:     p,
				corrected: corrected,
				items:     items,
			}
		}()
	}
	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}

	res := &model.Result{
		Original:   text,
		CharCount:  utf8.RuneCountInString(text),
		ChunkCount: len(parts),
	}

	corrParts := make([]string, len(out))
	for i, cr := range out {
		corrParts[i] = cr.corrected
		res.ErrorCount += len(cr.items)
		if len(cr.items) > 0 {
			res.Corrections = append(res.Corrections, model.Chunk{
				Idx:   cr.idx,
				Input: cr.input,
				Items: cr.items,
			})
		}
	}

	res.Corrected = strings.Join(corrParts, "")
	res.EditDistance = util.Levenshtein(res.Original, res.Corrected)
	return res, nil
}

// CheckHanspellWithDict is like CheckHanspell but filters words listed in dict.
func CheckHanspellWithDict(ctx context.Context, text string, c *internalhanspell.Checker, dict *Dict) (*model.Result, error) {
	res, err := CheckHanspell(ctx, text, c)
	if err != nil || dict == nil || len(dict.Words) == 0 {
		return res, err
	}
	filterByDict(res, dict)

	parts := splitHanspellChunks(res.Original)
	itemsByIdx := make(map[int][]model.Correction, len(res.Corrections))
	for _, ch := range res.Corrections {
		itemsByIdx[ch.Idx] = ch.Items
	}

	corrParts := make([]string, len(parts))
	for i, p := range parts {
		corrParts[i] = applyCorrections(p, itemsByIdx[i])
	}
	res.Corrected = strings.Join(corrParts, "")
	res.Corrected = canonicalizeByDictWords(res.Corrected, dict)
	res.EditDistance = util.Levenshtein(res.Original, res.Corrected)
	return res, nil
}

func splitHanspellChunks(text string) []string {
	if text == "" {
		return []string{""}
	}
	runes := []rune(text)
	if len(runes) <= hanspellMaxRunesPerChunk {
		return []string{text}
	}

	parts := make([]string, 0, (len(runes)+hanspellMaxRunesPerChunk-1)/hanspellMaxRunesPerChunk)
	for start := 0; start < len(runes); {
		end := start + hanspellMaxRunesPerChunk
		if end >= len(runes) {
			parts = append(parts, string(runes[start:]))
			break
		}

		cut := end
		for i := end; i > start+hanspellMaxRunesPerChunk/2; i-- {
			if runes[i-1] == ' ' || runes[i-1] == '\n' || runes[i-1] == '\t' {
				cut = i
				break
			}
		}

		parts = append(parts, string(runes[start:cut]))
		start = cut
	}
	return parts
}

func buildHanspellCorrections(originalText, originHTML, correctedHTML string) []model.Correction {
	origins := extractOriginSegments(originHTML)
	if len(origins) == 0 {
		return nil
	}
	corrected := extractCorrectedSegments(correctedHTML)

	out := make([]model.Correction, 0, len(origins))
	searchFrom := 0

	for i, origin := range origins {
		if origin == "" {
			continue
		}

		pos := strings.Index(originalText[searchFrom:], origin)
		if pos >= 0 {
			pos += searchFrom
		} else {
			// Fallback for occasional mismatches between markup and raw text.
			pos = strings.Index(originalText, origin)
			if pos < 0 {
				continue
			}
		}

		start := utf8.RuneCountInString(originalText[:pos])
		end := start + utf8.RuneCountInString(origin)

		suggest := origin
		help := ""
		if i < len(corrected) {
			if corrected[i].Text != "" {
				suggest = corrected[i].Text
			}
			help = helpByClass(corrected[i].Class)
		}

		out = append(out, model.Correction{
			Start:     start,
			End:       end,
			Origin:    origin,
			Suggest:   []string{suggest},
			Distances: []int{util.Levenshtein(origin, suggest)},
			Help:      help,
		})

		searchFrom = pos + len(origin)
	}

	return out
}

func extractOriginSegments(originHTML string) []string {
	matches := reOriginUnderline.FindAllStringSubmatch(originHTML, -1)
	if len(matches) == 0 {
		return nil
	}
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		seg := cleanMarkedText(m[1])
		if seg == "" {
			continue
		}
		out = append(out, seg)
	}
	return out
}

func extractCorrectedSegments(correctedHTML string) []correctedSegment {
	matches := reCorrectedText.FindAllStringSubmatch(correctedHTML, -1)
	if len(matches) == 0 {
		return nil
	}
	out := make([]correctedSegment, 0, len(matches))
	for _, m := range matches {
		if len(m) < 3 {
			continue
		}
		out = append(out, correctedSegment{
			Class: strings.TrimSpace(m[1]),
			Text:  cleanMarkedText(m[2]),
		})
	}
	return out
}

func cleanMarkedText(s string) string {
	s = htmlstd.UnescapeString(s)
	s = strings.ReplaceAll(s, "<br>", "\n")
	s = strings.ReplaceAll(s, "<br/>", "\n")
	s = strings.ReplaceAll(s, "<br />", "\n")
	return reHTMLTag.ReplaceAllString(s, "")
}

func helpByClass(className string) string {
	switch className {
	case "red_text":
		return "맞춤법 오류"
	case "green_text":
		return "띄어쓰기 오류"
	case "violet_text":
		return "표준어 의심"
	case "blue_text":
		return "통계적 교정"
	default:
		return ""
	}
}
