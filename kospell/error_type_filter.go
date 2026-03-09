package kospell

import (
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/Alfex4936/kospell/internal/model"
	"github.com/Alfex4936/kospell/internal/util"
)

const (
	errorTypeSpelling    = "spelling"
	errorTypeSpacing     = "spacing"
	errorTypeStandard    = "standard"
	errorTypeStatistical = "statistical"
	errorTypeUnknown     = "unknown"
)

func defaultAllowedErrorTypes() map[string]struct{} {
	return map[string]struct{}{
		errorTypeSpelling: {},
		errorTypeSpacing:  {},
	}
}

// normalizeErrorTypes normalizes user-provided error type names.
// It returns the normalized set and invalid values (if any).
func normalizeErrorTypes(types []string) (map[string]struct{}, []string) {
	set := make(map[string]struct{}, len(types))
	var invalid []string

	for _, raw := range types {
		t, ok := normalizeErrorType(raw)
		if !ok {
			invalid = append(invalid, raw)
			continue
		}
		set[t] = struct{}{}
	}

	return set, invalid
}

func normalizeErrorType(raw string) (string, bool) {
	s := strings.ToLower(strings.TrimSpace(raw))
	s = strings.ReplaceAll(s, "_", "")
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, " ", "")

	switch s {
	case "spelling", "wrongspelling", "맞춤법", "철자":
		return errorTypeSpelling, true
	case "spacing", "wrongspacing", "띄어쓰기":
		return errorTypeSpacing, true
	case "standard", "ambiguous", "표준어", "표준어의심":
		return errorTypeStandard, true
	case "statistical", "statisticalcorrection", "통계적교정", "통계교정":
		return errorTypeStatistical, true
	case "unknown", "기타":
		return errorTypeUnknown, true
	default:
		return "", false
	}
}

func filterResultByErrorTypes(res *model.Result, allowed map[string]struct{}, dict *Dict) {
	if res == nil || len(allowed) == 0 {
		return
	}

	newCorrs := res.Corrections[:0]
	totalErrors := 0

	for _, c := range res.Corrections {
		kept := c.Items[:0]
		for i := range c.Items {
			t := classifyErrorType(&c.Items[i])
			if _, ok := allowed[t]; ok {
				kept = append(kept, c.Items[i])
			}
		}
		c.Items = kept
		if len(c.Items) > 0 {
			totalErrors += len(c.Items)
			newCorrs = append(newCorrs, c)
		}
	}

	res.Corrections = newCorrs
	res.ErrorCount = totalErrors
	res.Corrected = applyCorrectionsFromChunks(res.Original, res.Corrections)
	if dict != nil && len(dict.Words) > 0 {
		res.Corrected = canonicalizeByDictWords(res.Corrected, dict)
	}
	res.EditDistance = util.Levenshtein(res.Original, res.Corrected)
}

func classifyErrorType(item *model.Correction) string {
	help := strings.TrimSpace(strings.ToLower(item.Help))

	switch {
	case strings.Contains(help, "띄어쓰기"), strings.Contains(help, "spacing"):
		return errorTypeSpacing
	case strings.Contains(help, "표준어"), strings.Contains(help, "ambiguous"):
		return errorTypeStandard
	case strings.Contains(help, "통계"), strings.Contains(help, "statistical"):
		return errorTypeStatistical
	case strings.Contains(help, "맞춤법"), strings.Contains(help, "철자"), strings.Contains(help, "spelling"):
		return errorTypeSpelling
	}

	if len(item.Suggest) > 0 && looksLikeSpacingChange(item.Origin, item.Suggest[0]) {
		return errorTypeSpacing
	}

	return errorTypeUnknown
}

func looksLikeSpacingChange(origin, suggest string) bool {
	originCanon := strings.Join(strings.Fields(origin), "")
	suggestCanon := strings.Join(strings.Fields(suggest), "")
	return originCanon != "" && originCanon == suggestCanon && origin != suggest
}

type chunkReplacement struct {
	start int
	end   int
	text  string
}

// applyCorrectionsFromChunks rebuilds corrected text from original + chunk-local corrections.
func applyCorrectionsFromChunks(original string, chunks []model.Chunk) string {
	if len(chunks) == 0 {
		return original
	}

	ordered := make([]model.Chunk, 0, len(chunks))
	ordered = append(ordered, chunks...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Idx < ordered[j].Idx })

	var reps []chunkReplacement
	searchFrom := 0 // byte index in original

	for _, ch := range ordered {
		if ch.Input == "" {
			continue
		}

		pos := -1
		if searchFrom >= 0 && searchFrom <= len(original) {
			if rel := strings.Index(original[searchFrom:], ch.Input); rel >= 0 {
				pos = searchFrom + rel
			}
		}
		if pos < 0 {
			pos = strings.Index(original, ch.Input)
		}
		if pos < 0 {
			continue
		}

		baseRune := utf8.RuneCountInString(original[:pos])
		for _, item := range ch.Items {
			if len(item.Suggest) == 0 {
				continue
			}
			reps = append(reps, chunkReplacement{
				start: baseRune + item.Start,
				end:   baseRune + item.End,
				text:  item.Suggest[0],
			})
		}

		searchFrom = pos + len(ch.Input)
	}

	if len(reps) == 0 {
		return original
	}

	sort.Slice(reps, func(i, j int) bool { return reps[i].start > reps[j].start })
	runes := []rune(original)

	for _, r := range reps {
		if r.start < 0 || r.end < r.start || r.end > len(runes) {
			continue
		}
		repl := []rune(r.text)
		runes = append(runes[:r.start], append(repl, runes[r.end:]...)...)
	}

	return string(runes)
}
