package bench

import (
	"strings"
	"testing"

	"github.com/Alfex4936/kospell/internal/chunk"
)

// build a 5 000-어절 sample once – reuse in all benches.
var (
	short = strings.Repeat("foo ", 299) + "bar"
	long  = strings.Repeat("x ", 5000) // 5 000 tokens
)

func BenchmarkSplitShort(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = chunk.Split300(short) // single chunk
	}
}

func BenchmarkSplitLong(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = chunk.Split300(long) // ~17 chunks
	}
}
