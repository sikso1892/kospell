// Package llm provides a spell-check backend backed by an OpenAI-compatible LLM.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	DefaultModel   = "gpt-5-mini"
	DefaultBaseURL = "https://api.openai.com/v1"
)

// Checker sends spell-check requests to an OpenAI-compatible chat completions API.
type Checker struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

// New creates a new LLM Checker.
// Unset fields fall back to their defaults.
func New(apiKey, model, baseURL string) *Checker {
	if model == "" {
		model = DefaultModel
	}
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Checker{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		model:   model,
		client:  &http.Client{Timeout: 60 * time.Second},
	}
}

// --- response structs (LLM doesn't include distances; caller adds them) ---

type Correction struct {
	Start   int      `json:"start"`
	End     int      `json:"end"`
	Origin  string   `json:"origin"`
	Suggest []string `json:"suggest"`
	Help    string   `json:"help"`
}

type Chunk struct {
	Idx   int          `json:"idx"`
	Input string       `json:"input"`
	Items []Correction `json:"items"`
}

type Response struct {
	Original    string  `json:"original"`
	Corrected   string  `json:"corrected"`
	Corrections []Chunk `json:"corrections"`
}

// --- OpenAI wire types ---

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model          string            `json:"model"`
	Messages       []chatMessage     `json:"messages"`
	ResponseFormat map[string]string `json:"response_format"`
}

type chatChoice struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
}

type chatResponse struct {
	Choices []chatChoice `json:"choices"`
	Error   *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Check calls the LLM and returns parsed spell-check results.
// protectedWords are passed as 고유명사 so the LLM won't flag them.
func (c *Checker) Check(ctx context.Context, text string, protectedWords []string) (*Response, error) {
	reqBody := chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: buildUserMessage(text, protectedWords)},
		},
		ResponseFormat: map[string]string{"type": "json_object"},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("llm: request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("llm: read body: %w", err)
	}

	var chatResp chatResponse
	if err := json.Unmarshal(raw, &chatResp); err != nil {
		return nil, fmt.Errorf("llm: decode response: %w", err)
	}
	if chatResp.Error != nil {
		return nil, fmt.Errorf("llm: API error: %s", chatResp.Error.Message)
	}
	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("llm: empty choices (status %d)", resp.StatusCode)
	}

	content := stripMarkdownFence(chatResp.Choices[0].Message.Content)

	var llmResp Response
	if err := json.Unmarshal([]byte(content), &llmResp); err != nil {
		return nil, fmt.Errorf("llm: parse JSON output: %w\ncontent: %s", err, content)
	}
	return &llmResp, nil
}

func buildUserMessage(text string, protected []string) string {
	if len(protected) == 0 {
		return "입력:\n" + text
	}
	wordList, _ := json.Marshal(protected)
	return "<고유명사>\n" + string(wordList) + "\n\n입력:\n" + text
}

// stripMarkdownFence removes optional ```json ... ``` wrapping from LLM output.
func stripMarkdownFence(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		if idx := strings.Index(s, "\n"); idx != -1 {
			s = s[idx+1:]
		}
		if idx := strings.LastIndex(s, "```"); idx != -1 {
			s = s[:idx]
		}
		s = strings.TrimSpace(s)
	}
	return s
}

const systemPrompt = `당신은 한국어 맞춤법·띄어쓰기 교정 전문가입니다. 반드시 JSON만 출력하세요.

규칙:
- 고유명사(특수 단어)로 지정된 단어는 절대 오류로 처리하지 않습니다.
- start/end는 입력 텍스트의 문자(rune) 오프셋입니다 (0부터 시작).
- 교정이 없으면 corrections를 빈 배열로, corrected를 original과 동일하게 반환합니다.
- suggest는 교정 제안 배열 (최소 1개, 최대 4개).
- help는 오류 원인 설명 (띄어쓰기, 맞춤법 등).

출력 형식 (JSON만, 설명·Markdown 불필요):
{
  "original": "<원본 텍스트>",
  "corrected": "<교정된 전체 텍스트>",
  "corrections": [
    {
      "idx": 0,
      "input": "<원본 텍스트>",
      "items": [
        {
          "start": <int>,
          "end": <int>,
          "origin": "<오류 원본>",
          "suggest": ["<제안1>"],
          "help": "<오류 설명>"
        }
      ]
    }
  ]
}`
