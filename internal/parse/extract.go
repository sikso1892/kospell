package parse

import "bytes"

var (
	keyData = []byte("data = ")
	endSeq  = []byte("];")
	noErr   = []byte("맞춤법과 문법 오류를 찾지")
)

// ExtractDataBlock pulls `[{"str": …}]` from the raw HTML/JS.
// It stops at the *first* occurrence of `"];` after the assignment.
func ExtractDataBlock(b []byte) []byte {
	i := bytes.Index(b, keyData)
	if i < 0 {
		return nil
	}
	// skip the “data = ”
	i += len(keyData)

	j := bytes.Index(b[i:], endSeq)
	if j < 0 {
		return nil
	}
	// include the closing `]`
	return b[i : i+j+1]
}

// NoError reports “맞춤법과 문법 오류를 찾지 못했습니다.” banner.
func NoError(b []byte) bool { return bytes.Contains(b, noErr) }
