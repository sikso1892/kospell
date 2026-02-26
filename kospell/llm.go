package kospell

import (
	"context"
	"unicode/utf8"

	internalllm "github.com/Alfex4936/kospell/internal/llm"
	"github.com/Alfex4936/kospell/internal/model"
	"github.com/Alfex4936/kospell/internal/util"
)

// CheckLLM checks text using the LLM backend.
// protectedWords are passed to the LLM prompt as 고유명사 (not flagged as errors).
func CheckLLM(ctx context.Context, text string, c *internalllm.Checker, protectedWords []string) (*model.Result, error) {
	raw, err := c.Check(ctx, text, protectedWords)
	if err != nil {
		return nil, err
	}
	return llmToResult(raw, text), nil
}

// CheckLLMWithDict is like CheckLLM but passes dict.Words as protected words
// to the LLM prompt, so they are never flagged.
func CheckLLMWithDict(ctx context.Context, text string, c *internalllm.Checker, dict *Dict) (*model.Result, error) {
	var protected []string
	if dict != nil {
		protected = dict.Words
	}
	res, err := CheckLLM(ctx, text, c, protected)
	if err != nil {
		return nil, err
	}
	if dict != nil && len(dict.Words) > 0 {
		res.Corrected = canonicalizeByDictWords(res.Corrected, dict)
		res.EditDistance = util.Levenshtein(res.Original, res.Corrected)
	}
	return res, nil
}

// llmToResult converts the LLM response into model.Result,
// filling in computed fields (distances, editDistance, charCount, …).
func llmToResult(raw *internalllm.Response, originalText string) *model.Result {
	original := raw.Original
	if original == "" {
		original = originalText
	}
	corrected := raw.Corrected
	if corrected == "" {
		corrected = original
	}

	var chunks []model.Chunk
	totalErrors := 0

	for _, c := range raw.Corrections {
		items := make([]model.Correction, 0, len(c.Items))
		for _, item := range c.Items {
			dists := make([]int, len(item.Suggest))
			for i, s := range item.Suggest {
				dists[i] = util.Levenshtein(item.Origin, s)
			}
			items = append(items, model.Correction{
				Start:     item.Start,
				End:       item.End,
				Origin:    item.Origin,
				Suggest:   item.Suggest,
				Distances: dists,
				Help:      item.Help,
			})
		}
		totalErrors += len(items)
		if len(items) > 0 {
			chunks = append(chunks, model.Chunk{
				Idx:   c.Idx,
				Input: c.Input,
				Items: items,
			})
		}
	}

	chunkCount := len(raw.Corrections)
	if chunkCount == 0 {
		chunkCount = 1
	}

	return &model.Result{
		Original:     original,
		Corrected:    corrected,
		EditDistance: util.Levenshtein(original, corrected),
		CharCount:    utf8.RuneCountInString(original),
		ChunkCount:   chunkCount,
		ErrorCount:   totalErrors,
		Corrections:  chunks,
	}
}
