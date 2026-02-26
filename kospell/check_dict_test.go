package kospell

import (
	"testing"

	"github.com/Alfex4936/kospell/internal/model"
)

func TestCanonicalizeByDictWords_MergesSpacedVariant(t *testing.T) {
	dict := NewDict("목제솜틀기")
	in := "나는 목제 솜 틀기 를 어제 주문했는데 아직도 안 와서 답답해."

	got := canonicalizeByDictWords(in, dict)

	want := "나는 목제솜틀기 를 어제 주문했는데 아직도 안 와서 답답해."
	if got != want {
		t.Fatalf("canonicalizeByDictWords() = %q, want %q", got, want)
	}
}

func TestCanonicalizeByDictWords_UsesCanonicalSpacing(t *testing.T) {
	dict := NewDict("우아한 형제들")
	in := "회사명은 우 아한형제들 이야."

	got := canonicalizeByDictWords(in, dict)

	want := "회사명은 우아한 형제들 이야."
	if got != want {
		t.Fatalf("canonicalizeByDictWords() = %q, want %q", got, want)
	}
}

func TestKeepCorrectionForDict_RewritesSuggestionToCanonical(t *testing.T) {
	words := normalizedDictWords(NewDict("목제솜틀기"))
	item := &model.Correction{
		Origin:    "목제 솜 틀기",
		Suggest:   []string{"목제 솜 틀기", "목재 솜틀기"},
		Distances: []int{0, 0},
	}

	ok := keepCorrectionForDict(item, words)
	if !ok {
		t.Fatal("keepCorrectionForDict() returned false, want true")
	}
	if len(item.Suggest) == 0 || item.Suggest[0] != "목제솜틀기" {
		t.Fatalf("first suggestion = %q, want %q", item.Suggest[0], "목제솜틀기")
	}
}
