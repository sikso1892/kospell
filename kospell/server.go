package kospell

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	internalllm "github.com/Alfex4936/kospell/internal/llm"
	"github.com/Alfex4936/kospell/internal/local"
	"github.com/Alfex4936/kospell/internal/model"
	"github.com/Alfex4936/kospell/internal/util"
)

// Mode selects the spell-check backend: "nara" | "hunspell" | "openai".
var Mode = "nara"

// LocalHunspell is the shared hunspell process used when Mode == "hunspell".
var LocalHunspell *local.Hunspell

// LLMChecker is the shared LLM client used when Mode == "openai".
var LLMChecker *internalllm.Checker

// CheckSpellRequest is the HTTP request body for /v1/check-spell
type CheckSpellRequest struct {
	Text     string   `json:"text"`                // 검사할 텍스트 (필수)
	Words    []string `json:"words,omitempty"`     // 인라인 허용 단어 목록 (선택)
	Dict     *Dict    `json:"dict,omitempty"`      // 사용자 딕셔너리 {"words":[...]} (선택)
	DictPath string   `json:"dict_path,omitempty"` // (deprecated) 딕셔너리 JSON 파일 경로 (서버 로컬)
	Timeout  int      `json:"timeout,omitempty"`   // 타임아웃 (초, 기본 8)
}

// CheckSpellHandler handles POST /v1/check-spell requests
func CheckSpellHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CheckSpellRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// 타임아웃 설정 (기본: nara/hunspell=8초, openai=60초)
	defaultTimeout := 8 * time.Second
	if Mode == "openai" {
		defaultTimeout = 3 * 60 * time.Second
	}
	timeout := defaultTimeout
	if req.Timeout > 0 {
		timeout = time.Duration(req.Timeout) * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 딕셔너리 구성: words(인라인) + dict(요청 본문) + dict_path(서버 로컬 파일, deprecated) 병합
	var dict *Dict
	if len(req.Words) > 0 || (req.Dict != nil && len(req.Dict.Words) > 0) || req.DictPath != "" {
		dict = NewDict(req.Words...)
		if req.Dict != nil {
			dict.Words = append(dict.Words, req.Dict.Words...)
		}
		if req.DictPath != "" {
			fileDict, err := LoadDict(req.DictPath)
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to load dictionary: %v", err), http.StatusInternalServerError)
				return
			}
			dict.Words = append(dict.Words, fileDict.Words...)
		}
	}

	var res *model.Result
	var err error

	switch Mode {
	case "openai":
		if LLMChecker == nil {
			http.Error(w, "openai mode: LLM checker not initialized", http.StatusInternalServerError)
			return
		}
		if dict != nil {
			res, err = CheckLLMWithDict(ctx, req.Text, LLMChecker, dict)
		} else {
			res, err = CheckLLM(ctx, req.Text, LLMChecker, nil)
		}
	case "hunspell":
		if LocalHunspell == nil {
			http.Error(w, "hunspell mode: checker not initialized", http.StatusInternalServerError)
			return
		}
		if dict != nil {
			res, err = CheckLocalWithDict(ctx, req.Text, LocalHunspell, dict)
		} else {
			res, err = CheckLocal(ctx, req.Text, LocalHunspell)
		}
	default: // nara
		if dict != nil {
			res, err = CheckWithDict(ctx, req.Text, dict)
		} else {
			res, err = Check(ctx, req.Text)
		}
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Check failed: %v", err), http.StatusInternalServerError)
		return
	}

	// JSON 응답 (HTML 이스케이프 비활성화)
	w.Header().Set("Content-Type", "application/json")
	out, _ := util.MarshalNoEscape(res, true)
	fmt.Fprint(w, string(out))
}

// HealthHandler handles GET /health requests
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"service": "kospell",
	})
}

// OpenAPIHandler serves the OpenAPI 3.0 spec at GET /openapi.json
func OpenAPIHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, openAPISpec)
}

// DocsHandler serves the Redoc UI at GET /
func DocsHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, redocHTML)
}

const openAPISpec = `{
  "openapi": "3.0.3",
  "info": {
    "title": "KoSpell API",
    "description": "한국어 맞춤법 검사 REST API (나라 맞춤법 검사기 기반)",
    "version": "1.0.0"
  },
  "paths": {
    "/v1/check-spell": {
      "post": {
        "summary": "Check Spell",
        "description": "텍스트의 맞춤법·문법 오류를 검사합니다. 사용자 딕셔너리로 특정 단어를 오류에서 제외할 수 있습니다.",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": { "$ref": "#/components/schemas/CheckSpellRequest" },
              "examples": {
                "기본": {
                  "value": { "text": "너는나와 kafka 머고나서" }
                },
                "인라인 딕셔너리": {
                  "value": { "text": "너는나와 kafka 머고나서", "words": ["kafka"] }
                },
                "사용자 딕셔너리(dict)": {
                  "value": { "text": "너는나와 kafka 머고나서", "dict": { "words": ["kafka"] } }
                },
                "타임아웃 지정": {
                  "value": { "text": "긴 텍스트...", "timeout": 15 }
                }
              }
            }
          }
        },
        "responses": {
          "200": {
            "description": "검사 결과",
            "content": {
              "application/json": {
                "schema": { "$ref": "#/components/schemas/Result" },
                "example": {
                  "original": "너는나와 kafka 머고나서",
                  "charCount": 15,
                  "chunkCount": 1,
                  "errorCount": 2,
                  "corrections": [
                    {
                      "idx": 0,
                      "input": "너는나와 kafka 머고나서",
                      "items": [
                        { "start": 0, "end": 4, "origin": "너는나와", "suggest": ["너는 나와"], "help": "띄어쓰기 오류" },
                        { "start": 11, "end": 15, "origin": "머고나서", "suggest": ["머고 나서"], "help": "띄어쓰기 오류" }
                      ]
                    }
                  ]
                }
              }
            }
          },
          "400": { "description": "잘못된 요청 (JSON 파싱 오류 등)" },
          "500": { "description": "서버 오류 (딕셔너리 로드 실패, 외부 API 오류 등)" }
        }
      }
    },
    "/health": {
      "get": {
        "summary": "Health",
        "responses": {
          "200": {
            "description": "서비스 정상",
            "content": {
              "application/json": {
                "example": { "status": "ok", "service": "kospell" }
              }
            }
          }
        }
      }
    }
  },
  "components": {
    "schemas": {
      "CheckSpellRequest": {
        "type": "object",
        "required": ["text"],
        "properties": {
          "text":      { "type": "string", "description": "검사할 텍스트 (필수)", "example": "너는나와 kafka 머고나서" },
          "words":     { "type": "array", "items": { "type": "string" }, "description": "오류에서 제외할 단어 목록 (인라인)", "example": ["kafka", "KoSpell"] },
          "dict":      { "$ref": "#/components/schemas/Dict" },
          "dict_path": { "type": "string", "description": "(deprecated) 딕셔너리 JSON 파일 경로 (서버 로컬)", "example": "/etc/kospell/dict.json", "deprecated": true },
          "timeout":   { "type": "integer", "description": "타임아웃 (초, 기본 8)", "example": 8 }
        }
      },
      "Dict": {
        "type": "object",
        "properties": {
          "words": { "type": "array", "items": { "type": "string" }, "description": "오류에서 제외할 단어 목록", "example": ["kafka", "KoSpell"] }
        }
      },
      "Result": {
        "type": "object",
        "properties": {
          "original":     { "type": "string", "description": "원본 입력 텍스트" },
          "corrected":    { "type": "string", "description": "첫 번째 제안을 적용한 교정 결과 텍스트" },
          "editDistance": { "type": "integer", "description": "Levenshtein(original, corrected) — 교정 전후 전체 편집거리" },
          "charCount":    { "type": "integer" },
          "chunkCount":   { "type": "integer" },
          "errorCount":   { "type": "integer" },
          "corrections":  { "type": "array", "items": { "$ref": "#/components/schemas/Chunk" } }
        }
      },
      "Chunk": {
        "type": "object",
        "properties": {
          "idx":   { "type": "integer" },
          "input": { "type": "string" },
          "items": { "type": "array", "items": { "$ref": "#/components/schemas/Correction" } }
        }
      },
      "Correction": {
        "type": "object",
        "properties": {
          "start":   { "type": "integer", "description": "오류 시작 위치 (바이트)" },
          "end":     { "type": "integer", "description": "오류 끝 위치 (바이트)" },
          "origin":  { "type": "string",  "description": "원본 오류 단어" },
          "suggest":   { "type": "array", "items": { "type": "string" }, "description": "교정 제안 목록" },
          "distances": { "type": "array", "items": { "type": "integer" }, "description": "suggest[i]와 origin 간 Levenshtein 편집거리" },
          "help":      { "type": "string",  "description": "오류 설명" }
        }
      }
    }
  }
}`

const redocHTML = `<!DOCTYPE html>
<html>
<head>
  <title>KoSpell API Docs</title>
  <meta charset="utf-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <link href="https://fonts.googleapis.com/css?family=Montserrat:300,400,700|Roboto:300,400,700" rel="stylesheet">
  <style>body { margin: 0; padding: 0; }</style>
</head>
<body>
  <redoc spec-url="/openapi.json" expand-responses="200" hide-download-button></redoc>
  <script src="https://cdn.jsdelivr.net/npm/redoc@latest/bundles/redoc.standalone.js"></script>
</body>
</html>`
