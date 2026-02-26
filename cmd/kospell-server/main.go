// Command kospell-server provides an HTTP REST API for spell checking.
//
// Usage:
//
//	kospell-server -p 8080 -mode nara
//	kospell-server -p 8080 -mode hunspell -dict /path/to/ko-dict -lang ko
//	kospell-server -p 8080 -mode openai -llm-key $OPENAI_API_KEY
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	internalllm "github.com/Alfex4936/kospell/internal/llm"
	"github.com/Alfex4936/kospell/internal/local"
	"github.com/Alfex4936/kospell/kospell"
)

func main() {
	port    := flag.String("p", "8080", "port to listen on")
	mode    := flag.String("mode", envOr("MODE", "nara"), "backend: nara | hunspell | openai")

	// hunspell flags
	dictDir := flag.String("dict", envOr("DICT_DIR", ""), "hunspell dictionary directory (hunspell mode)")
	lang    := flag.String("lang", envOr("DICT_LANG", "ko"), "hunspell dictionary name (hunspell mode)")

	// openai flags
	llmKey   := flag.String("llm-key",   envOr("OPENAI_API_KEY", ""), "OpenAI API key (openai mode)")
	llmModel := flag.String("llm-model", envOr("LLM_MODEL", internalllm.DefaultModel), "LLM model name")
	llmURL   := flag.String("llm-url",   envOr("LLM_BASE_URL", internalllm.DefaultBaseURL), "OpenAI-compatible base URL")

	flag.Parse()

	switch *mode {
	case "hunspell":
		h, err := local.New(*dictDir, *lang)
		if err != nil {
			log.Fatalf("hunspell init failed: %v", err)
		}
		kospell.Mode = "hunspell"
		kospell.LocalHunspell = h
		log.Printf("   backend : hunspell (dict=%s/%s)\n", *dictDir, *lang)

	case "openai":
		if *llmKey == "" {
			log.Fatal("openai mode requires -llm-key or OPENAI_API_KEY env var")
		}
		kospell.Mode = "openai"
		kospell.LLMChecker = internalllm.New(*llmKey, *llmModel, *llmURL)
		log.Printf("   backend : openai (model=%s url=%s)\n", *llmModel, *llmURL)

	default:
		kospell.Mode = "nara"
		log.Printf("   backend : nara (nara-speller API)\n")
	}

	http.HandleFunc("/v1/check-spell", kospell.CheckSpellHandler)
	http.HandleFunc("/health", kospell.HealthHandler)
	http.HandleFunc("/openapi.json", kospell.OpenAPIHandler)
	http.HandleFunc("/", kospell.DocsHandler)

	addr := fmt.Sprintf(":%s", *port)
	log.Printf("🚀 kospell server listening on http://localhost:%s\n", *port)
	log.Printf("   POST http://localhost:%s/v1/check-spell\n", *port)
	log.Printf("   GET  http://localhost:%s/health\n", *port)
	log.Printf("   GET  http://localhost:%s/       (Redoc UI)\n", *port)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
