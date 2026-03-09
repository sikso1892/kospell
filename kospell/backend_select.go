package kospell

import (
	"fmt"
	"strings"
	"sync"
	"time"

	internalhanspell "github.com/Alfex4936/kospell/internal/hanspell"
)

const (
	backendNara     = "nara"
	backendHunspell = "hunspell"
	backendHanspell = "hanspell"
	backendOpenAI   = "openai"
)

var hanspellMu sync.Mutex

func resolveBackend(requestBackend string) (string, error) {
	if strings.TrimSpace(requestBackend) == "" {
		if backend, ok := normalizeBackend(Mode); ok {
			return backend, nil
		}
		return "", fmt.Errorf("server default backend is invalid: %q", Mode)
	}

	if backend, ok := normalizeBackend(requestBackend); ok {
		return backend, nil
	}

	return "", fmt.Errorf("invalid backend: %q (allowed: nara, hunspell, hanspell, openai)", requestBackend)
}

func normalizeBackend(raw string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case backendNara:
		return backendNara, true
	case backendHunspell:
		return backendHunspell, true
	case backendHanspell, "naver":
		return backendHanspell, true
	case backendOpenAI:
		return backendOpenAI, true
	default:
		return "", false
	}
}

func defaultTimeoutForBackend(backend string) time.Duration {
	if backend == backendOpenAI {
		return 3 * time.Minute
	}
	return 8 * time.Second
}

func getHanspellChecker() *internalhanspell.Checker {
	hanspellMu.Lock()
	defer hanspellMu.Unlock()

	if HanspellChecker == nil {
		HanspellChecker = internalhanspell.New()
	}
	return HanspellChecker
}
