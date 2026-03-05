package kospell

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestBuildHanspellCorrections_Basic(t *testing.T) {
	original := "안녕 하세요. 저는 한국인 입니다."
	originHTML := "<span class='result_underline'>안녕 하세요.</span> 저는 <span class='result_underline'>한국인 입니다.</span>"
	correctedHTML := "<em class='green_text'>안녕하세요.</em> 저는 <em class='green_text'>한국인입니다.</em>"

	items := buildHanspellCorrections(original, originHTML, correctedHTML)
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}

	if items[0].Origin != "안녕 하세요." {
		t.Fatalf("items[0].Origin = %q, want %q", items[0].Origin, "안녕 하세요.")
	}
	if len(items[0].Suggest) == 0 || items[0].Suggest[0] != "안녕하세요." {
		t.Fatalf("items[0].Suggest[0] = %q, want %q", items[0].Suggest[0], "안녕하세요.")
	}
	if items[0].Help != "띄어쓰기 오류" {
		t.Fatalf("items[0].Help = %q, want %q", items[0].Help, "띄어쓰기 오류")
	}

	r := []rune(original)
	got0 := string(r[items[0].Start:items[0].End])
	got1 := string(r[items[1].Start:items[1].End])
	if got0 != items[0].Origin {
		t.Fatalf("span0 = %q, want %q", got0, items[0].Origin)
	}
	if got1 != items[1].Origin {
		t.Fatalf("span1 = %q, want %q", got1, items[1].Origin)
	}
}

func TestBuildHanspellCorrections_HelpByClass(t *testing.T) {
	original := "됬습니다 표준어아닌단어"
	originHTML := "<span class='result_underline'>됬습니다</span> <span class='result_underline'>표준어아닌단어</span>"
	correctedHTML := "<em class='red_text'>됐습니다</em> <em class='violet_text'>표준어 아닌 단어</em>"

	items := buildHanspellCorrections(original, originHTML, correctedHTML)
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].Help != "맞춤법 오류" {
		t.Fatalf("items[0].Help = %q, want %q", items[0].Help, "맞춤법 오류")
	}
	if items[1].Help != "표준어 의심" {
		t.Fatalf("items[1].Help = %q, want %q", items[1].Help, "표준어 의심")
	}
}

func TestSplitHanspellChunks_Rejoin(t *testing.T) {
	// 350+ runes
	text := strings.Repeat("가나다라마바사 ", 40)
	parts := splitHanspellChunks(text)
	if len(parts) < 2 {
		t.Fatalf("len(parts) = %d, want >= 2", len(parts))
	}
	for i, p := range parts {
		if utf8.RuneCountInString(p) > hanspellMaxRunesPerChunk {
			t.Fatalf("parts[%d] rune len = %d, want <= %d", i, utf8.RuneCountInString(p), hanspellMaxRunesPerChunk)
		}
	}

	joined := strings.Join(parts, "")
	if joined != text {
		t.Fatalf("joined text mismatch")
	}
}
