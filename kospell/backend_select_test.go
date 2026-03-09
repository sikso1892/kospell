package kospell

import (
	"testing"
	"time"
)

func TestNormalizeBackend(t *testing.T) {
	tests := []struct {
		in   string
		want string
		ok   bool
	}{
		{in: "nara", want: backendNara, ok: true},
		{in: "hunspell", want: backendHunspell, ok: true},
		{in: "hanspell", want: backendHanspell, ok: true},
		{in: "naver", want: backendHanspell, ok: true},
		{in: "openai", want: backendOpenAI, ok: true},
		{in: "bad", want: "", ok: false},
	}

	for _, tc := range tests {
		got, ok := normalizeBackend(tc.in)
		if ok != tc.ok || got != tc.want {
			t.Fatalf("normalizeBackend(%q) = (%q, %v), want (%q, %v)", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

func TestResolveBackend_UsesRequestOverride(t *testing.T) {
	prevMode := Mode
	Mode = backendNara
	t.Cleanup(func() { Mode = prevMode })

	got, err := resolveBackend("hanspell")
	if err != nil {
		t.Fatalf("resolveBackend returned error: %v", err)
	}
	if got != backendHanspell {
		t.Fatalf("resolveBackend = %q, want %q", got, backendHanspell)
	}
}

func TestResolveBackend_UsesServerDefaultWhenOmitted(t *testing.T) {
	prevMode := Mode
	Mode = "naver"
	t.Cleanup(func() { Mode = prevMode })

	got, err := resolveBackend("")
	if err != nil {
		t.Fatalf("resolveBackend returned error: %v", err)
	}
	if got != backendHanspell {
		t.Fatalf("resolveBackend = %q, want %q", got, backendHanspell)
	}
}

func TestResolveBackend_InvalidRequest(t *testing.T) {
	prevMode := Mode
	Mode = backendNara
	t.Cleanup(func() { Mode = prevMode })

	if _, err := resolveBackend("invalid"); err == nil {
		t.Fatal("resolveBackend should fail for invalid backend")
	}
}

func TestDefaultTimeoutForBackend(t *testing.T) {
	if got := defaultTimeoutForBackend(backendOpenAI); got != 3*time.Minute {
		t.Fatalf("openai timeout = %v, want %v", got, 3*time.Minute)
	}
	if got := defaultTimeoutForBackend(backendNara); got != 8*time.Second {
		t.Fatalf("nara timeout = %v, want %v", got, 8*time.Second)
	}
}
