// Package kospell is a thin, allocation-aware wrapper around
// https://nara-speller.co.kr for non-commercial spell-checking.
package kospell

import (
	"context"
	"errors"
	"io"
	"net/url"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/Alfex4936/kospell/internal/chunk"
	"github.com/Alfex4936/kospell/internal/model"
	"github.com/Alfex4936/kospell/internal/net"
	"github.com/Alfex4936/kospell/internal/parse"
	"github.com/Alfex4936/kospell/internal/util"
)

// Check submits text (any length) and returns a normalized Result.
//
// It transparently splits input into ≤300-어절 chunks,
// dispatches them in parallel (bounded by GOMAXPROCS), and merges the outcome.
//
// ctx controls overall timeout / cancellation.
func Check(ctx context.Context, text string) (*model.Result, error) {
	text = strings.TrimSpace(text) // remove all whitespace before and after
	if ctx == nil {
		return nil, errors.New("ctx is nil")
	}

	parts := chunk.Split300(text)
	out := make([]chunkResult, len(parts))

	sem := make(chan struct{}, cap(make([]byte, 0, runtime.GOMAXPROCS(0))))
	var wg sync.WaitGroup
	var firstErr error
	for i, p := range parts {
		i, p := i, p
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			cr, err := doRequest(ctx, p)
			if err != nil && firstErr == nil {
				firstErr = err
				return
			}
			cr.idx = i
			if cr.input == "" {
				cr.input = p
			}
			out[i] = cr
		}()
	}
	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}

	var totalErrors int
	for _, cr := range out {
		totalErrors += len(cr.items)
	}

	// merge → public Result
	res := &model.Result{
		Original:   text,
		CharCount:  utf8.RuneCountInString(text),
		ErrorCount: totalErrors,
		ChunkCount: len(parts),
	}
	res.Corrections = make([]model.Chunk, 0, len(out))
	for _, cr := range out {
		if len(cr.items) > 0 {
			res.Corrections = append(res.Corrections, model.Chunk{
				Idx:   cr.idx,
				Input: cr.input,
				Items: cr.items,
			})
		}
	}

	// build corrected text: apply first suggestion per chunk, then join
	corrParts := make([]string, len(out))
	for i, cr := range out {
		corrParts[i] = applyCorrections(cr.input, cr.items)
	}
	res.Corrected = strings.Join(corrParts, " ")
	res.EditDistance = util.Levenshtein(res.Original, res.Corrected)

	return res, nil
}

// applyCorrections replaces each error span with its first suggestion.
// Applies right-to-left so earlier rune offsets stay valid.
func applyCorrections(input string, items []model.Correction) string {
	if len(items) == 0 {
		return input
	}
	sorted := make([]model.Correction, len(items))
	copy(sorted, items)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Start > sorted[j].Start })

	runes := []rune(input)
	for _, c := range sorted {
		if len(c.Suggest) == 0 {
			continue
		}
		repl := []rune(c.Suggest[0])
		runes = append(runes[:c.Start], append(repl, runes[c.End:]...)...)
	}
	return string(runes)
}

// CheckWithDict is like Check but filters out any Correction whose Origin
// is listed in dict.
func CheckWithDict(ctx context.Context, text string, dict *Dict) (*model.Result, error) {
	res, err := Check(ctx, text)
	if err != nil || dict == nil || len(dict.Words) == 0 {
		return res, err
	}
	filterByDict(res, dict)

	// Rebuild corrected text after filtering/reordering suggestions.
	parts := chunk.Split300(res.Original)
	itemsByIdx := make(map[int][]model.Correction, len(res.Corrections))
	for _, c := range res.Corrections {
		itemsByIdx[c.Idx] = c.Items
	}
	corrParts := make([]string, len(parts))
	for i, p := range parts {
		corrParts[i] = applyCorrections(p, itemsByIdx[i])
	}
	res.Corrected = strings.Join(corrParts, " ")
	res.EditDistance = util.Levenshtein(res.Original, res.Corrected)

	return res, nil
}

func filterByDict(res *model.Result, dict *Dict) {
	newCorrs := res.Corrections[:0]
	for _, c := range res.Corrections {
		kept := c.Items[:0]
		for i := range c.Items {
			if keepCorrectionForDict(&c.Items[i], dict) {
				kept = append(kept, c.Items[i])
			}
		}
		removed := len(c.Items) - len(kept)
		res.ErrorCount -= removed
		c.Items = kept
		if len(c.Items) > 0 {
			newCorrs = append(newCorrs, c)
		}
	}
	res.Corrections = newCorrs
}

func keepCorrectionForDict(item *model.Correction, dict *Dict) bool {
	if dict == nil || len(dict.Words) == 0 {
		return true
	}

	originNoSpace := strings.ReplaceAll(item.Origin, " ", "")
	relevant := make([]string, 0, len(dict.Words))
	for _, raw := range dict.Words {
		w := strings.TrimSpace(raw)
		if w == "" {
			continue
		}
		wNoSpace := strings.ReplaceAll(w, " ", "")
		if wNoSpace == "" {
			continue
		}
		if strings.Contains(item.Origin, w) || strings.Contains(originNoSpace, wNoSpace) {
			relevant = append(relevant, w)
		}
	}
	if len(relevant) == 0 {
		return true
	}

	best := -1
	for i, s := range item.Suggest {
		fixed := s
		changed := false
		for _, w := range relevant {
			var c bool
			fixed, c = collapseSpacesWithinWord(fixed, w)
			changed = changed || c
		}

		ok := true
		for _, w := range relevant {
			if !strings.Contains(fixed, w) {
				ok = false
				break
			}
		}
		if ok {
			best = i
			if changed {
				item.Suggest[i] = fixed
				if i < len(item.Distances) {
					item.Distances[i] = util.Levenshtein(item.Origin, fixed)
				}
			}
			break
		}
	}

	if best == -1 {
		return false
	}
	if best != 0 {
		item.Suggest[0], item.Suggest[best] = item.Suggest[best], item.Suggest[0]
		if best < len(item.Distances) {
			item.Distances[0], item.Distances[best] = item.Distances[best], item.Distances[0]
		}
	}
	return true
}

func collapseSpacesWithinWord(s, word string) (string, bool) {
	if word == "" || strings.Contains(s, word) {
		return s, false
	}

	runes := []rune(word)
	if len(runes) < 2 {
		return s, false
	}

	var b strings.Builder
	for i, r := range runes {
		b.WriteString(regexp.QuoteMeta(string(r)))
		if i != len(runes)-1 {
			b.WriteString(`\s*`)
		}
	}

	re := regexp.MustCompile(b.String())
	out := re.ReplaceAllString(s, word)
	return out, out != s
}

/***----- private -----***/

type chunkResult struct {
	idx   int
	input string
	items []model.Correction
}

// var bufPool = sync.Pool{New: func() any { return &strings.Builder{} }}

func doRequest(ctx context.Context, text string) (chunkResult, error) {
	// , "bWeakOpt": {"true"}, "pageIdx": {"1"}
	form := url.Values{"text1": {text}}

	req, err := net.NewPOST(ctx, "/old_speller/results", strings.NewReader(form.Encode()))
	if err != nil {
		return chunkResult{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := net.Do(req)
	if err != nil {
		return chunkResult{}, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	raw := parse.ExtractDataBlock(body) // []byte of `[{"str": ...`
	if raw == nil {
		// fmt.Printf("raw err = %v\n", raw)
		if parse.NoError(body) { // “맞춤법과 문법 오류를 찾지 못했습니다.”
			return chunkResult{}, nil
		}
		return chunkResult{}, ErrParse
	}

	items, err := parse.Decode(raw)
	if err != nil {
		// fmt.Printf("items err = %v\n", err)
		return chunkResult{}, err
	}

	return chunkResult{input: text, idx: 0, items: items}, nil
}
