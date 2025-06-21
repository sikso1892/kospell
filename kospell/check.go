// Package kospell is a thin, allocation-aware wrapper around
// https://nara-speller.co.kr for non-commercial spell-checking.
package kospell

import (
	"context"
	"errors"
	"io"
	"net/url"
	"runtime"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/Alfex4936/kospell/internal/chunk"
	"github.com/Alfex4936/kospell/internal/model"
	"github.com/Alfex4936/kospell/internal/net"
	"github.com/Alfex4936/kospell/internal/parse"
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
	return res, nil
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
