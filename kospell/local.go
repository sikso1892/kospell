package kospell

import (
	"context"
	"unicode/utf8"

	"github.com/Alfex4936/kospell/internal/local"
	"github.com/Alfex4936/kospell/internal/model"
	"github.com/Alfex4936/kospell/internal/util"
)

// CheckLocal checks text using the local hunspell backend.
func CheckLocal(ctx context.Context, text string, h *local.Hunspell) (*model.Result, error) {
	items, err := h.CheckText(text)
	if err != nil {
		return nil, err
	}

	res := &model.Result{
		Original:   text,
		CharCount:  utf8.RuneCountInString(text),
		ChunkCount: 1,
		ErrorCount: len(items),
	}
	if len(items) > 0 {
		res.Corrections = []model.Chunk{{Idx: 0, Input: text, Items: items}}
	}

	res.Corrected = applyCorrections(text, items)
	res.EditDistance = util.Levenshtein(res.Original, res.Corrected)
	return res, nil
}

// CheckLocalWithDict is like CheckLocal but filters words listed in dict.
func CheckLocalWithDict(ctx context.Context, text string, h *local.Hunspell, dict *Dict) (*model.Result, error) {
	res, err := CheckLocal(ctx, text, h)
	if err != nil || dict == nil || len(dict.Words) == 0 {
		return res, err
	}
	filterByDict(res, dict)

	// Recompute corrected after filtering
	var filtered []model.Correction
	for _, c := range res.Corrections {
		filtered = append(filtered, c.Items...)
	}
	res.Corrected = applyCorrections(text, filtered)
	res.Corrected = canonicalizeByDictWords(res.Corrected, dict)
	res.EditDistance = util.Levenshtein(res.Original, res.Corrected)
	return res, nil
}
