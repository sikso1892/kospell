package kospell

import (
	"testing"

	"github.com/Alfex4936/kospell/internal/model"
)

func TestNormalizeErrorTypes(t *testing.T) {
	set, invalid := normalizeErrorTypes([]string{"spacing", "맞춤법", "표준어의심", "bad_type"})
	if len(invalid) != 1 || invalid[0] != "bad_type" {
		t.Fatalf("invalid = %v, want [bad_type]", invalid)
	}
	if _, ok := set[errorTypeSpacing]; !ok {
		t.Fatalf("missing %q", errorTypeSpacing)
	}
	if _, ok := set[errorTypeSpelling]; !ok {
		t.Fatalf("missing %q", errorTypeSpelling)
	}
	if _, ok := set[errorTypeStandard]; !ok {
		t.Fatalf("missing %q", errorTypeStandard)
	}
}

func TestDefaultAllowedErrorTypes(t *testing.T) {
	set := defaultAllowedErrorTypes()
	if len(set) != 2 {
		t.Fatalf("len(default set) = %d, want 2", len(set))
	}
	if _, ok := set[errorTypeSpelling]; !ok {
		t.Fatalf("missing %q", errorTypeSpelling)
	}
	if _, ok := set[errorTypeSpacing]; !ok {
		t.Fatalf("missing %q", errorTypeSpacing)
	}
}

func TestFilterResultByErrorTypes_OnlySpacing(t *testing.T) {
	original := "됬습니다 안녕 하세요"
	res := &model.Result{
		Original:   original,
		Corrected:  "됐습니다 안녕하세요",
		ChunkCount: 1,
		ErrorCount: 2,
		Corrections: []model.Chunk{
			{
				Idx:   0,
				Input: original,
				Items: []model.Correction{
					{
						Start:     0,
						End:       4,
						Origin:    "됬습니다",
						Suggest:   []string{"됐습니다"},
						Distances: []int{1},
						Help:      "맞춤법 오류",
					},
					{
						Start:     5,
						End:       11,
						Origin:    "안녕 하세요",
						Suggest:   []string{"안녕하세요"},
						Distances: []int{1},
						Help:      "띄어쓰기 오류",
					},
				},
			},
		},
	}

	filterResultByErrorTypes(res, map[string]struct{}{errorTypeSpacing: {}}, nil)

	if res.ErrorCount != 1 {
		t.Fatalf("ErrorCount = %d, want 1", res.ErrorCount)
	}
	if len(res.Corrections) != 1 || len(res.Corrections[0].Items) != 1 {
		t.Fatalf("corrections len = %d/%d, want 1/1", len(res.Corrections), len(res.Corrections[0].Items))
	}
	if got, want := res.Corrected, "됬습니다 안녕하세요"; got != want {
		t.Fatalf("Corrected = %q, want %q", got, want)
	}
}

func TestClassifyErrorType_BySpacingShape(t *testing.T) {
	item := &model.Correction{
		Origin:  "한국인 입니다",
		Suggest: []string{"한국인입니다"},
	}
	if got := classifyErrorType(item); got != errorTypeSpacing {
		t.Fatalf("classifyErrorType = %q, want %q", got, errorTypeSpacing)
	}
}

func TestFilterResultByErrorTypes_DropsNoopSuggestionItem(t *testing.T) {
	original := "성남미디어센터의 로비"
	res := &model.Result{
		Original:   original,
		Corrected:  original,
		ChunkCount: 1,
		ErrorCount: 1,
		Corrections: []model.Chunk{
			{
				Idx:   0,
				Input: original,
				Items: []model.Correction{
					{
						Start:     0,
						End:       8,
						Origin:    "성남미디어센터의",
						Suggest:   []string{"성남미디어센터의"},
						Distances: []int{0},
						Help:      "띄어쓰기 오류",
					},
				},
			},
		},
	}

	filterResultByErrorTypes(res, map[string]struct{}{errorTypeSpacing: {}}, nil)

	if res.ErrorCount != 0 {
		t.Fatalf("ErrorCount = %d, want 0", res.ErrorCount)
	}
	if len(res.Corrections) != 0 {
		t.Fatalf("len(Corrections) = %d, want 0", len(res.Corrections))
	}
	if got, want := res.Corrected, original; got != want {
		t.Fatalf("Corrected = %q, want %q", got, want)
	}
}

func TestNormalizeSuggestionSet_RemovesOnlyNoopEntries(t *testing.T) {
	item := &model.Correction{
		Origin:    "안 내",
		Suggest:   []string{"안 내", "안내"},
		Distances: []int{0, 1},
	}

	ok := normalizeSuggestionSet(item)
	if !ok {
		t.Fatal("normalizeSuggestionSet should keep effective suggestions")
	}
	if len(item.Suggest) != 1 || item.Suggest[0] != "안내" {
		t.Fatalf("Suggest = %v, want [안내]", item.Suggest)
	}
	if len(item.Distances) != 1 || item.Distances[0] != 1 {
		t.Fatalf("Distances = %v, want [1]", item.Distances)
	}
}

func TestFilterResultByErrorTypes_SetsErrorTypeOnResponseItem(t *testing.T) {
	original := "맞춤법"
	res := &model.Result{
		Original:   original,
		Corrected:  original,
		ChunkCount: 1,
		ErrorCount: 1,
		Corrections: []model.Chunk{
			{
				Idx:   0,
				Input: original,
				Items: []model.Correction{
					{
						Start:     0,
						End:       3,
						Origin:    "맞춤법",
						Suggest:   []string{"마춤법"},
						Distances: []int{1},
						Help:      "맞춤법 오류",
					},
				},
			},
		},
	}

	filterResultByErrorTypes(res, map[string]struct{}{errorTypeSpelling: {}}, nil)

	if len(res.Corrections) != 1 || len(res.Corrections[0].Items) != 1 {
		t.Fatalf("unexpected corrections size: %d/%d", len(res.Corrections), len(res.Corrections[0].Items))
	}
	if got := res.Corrections[0].Items[0].ErrorType; got != errorTypeSpelling {
		t.Fatalf("ErrorType = %q, want %q", got, errorTypeSpelling)
	}
}
