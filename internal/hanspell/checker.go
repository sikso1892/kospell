// Package hanspell provides a Naver spell-check backend compatible with
// py-hanspell's request flow.
package hanspell

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	defaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 " +
		"(KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36"

	checkerURL   = "https://ts-proxy.naver.com/ocontent/util/SpellerProxy"
	searchQuery  = "맞춤법 검사기"
	searchURLPC  = "https://search.naver.com/search.naver"
	searchURLMob = "https://m.search.naver.com/search.naver"

	// Passport keys rotate. Keep the key briefly to avoid fetching it per request.
	passportKeyTTL = 15 * time.Minute
)

var (
	passportKeyRe = regexp.MustCompile(`passportKey=([a-zA-Z0-9]+)`)
	errInvalidKey = errors.New("hanspell: invalid passport key")
)

// Response is the raw normalized output from Naver's checker API.
type Response struct {
	ErrataCount int
	OriginHTML  string
	HTML        string
	Corrected   string
}

// Checker fetches passportKey and calls Naver's spell-check endpoint.
type Checker struct {
	client *http.Client

	mu          sync.RWMutex
	passportKey string
	keyFetched  time.Time
}

// New creates a checker with a default HTTP client.
func New() *Checker {
	return &Checker{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// Check runs spell-check against Naver API.
func (c *Checker) Check(ctx context.Context, text string) (*Response, error) {
	if ctx == nil {
		return nil, errors.New("hanspell: ctx is nil")
	}

	key, err := c.getPassportKey(ctx, false)
	if err != nil {
		return nil, err
	}

	res, err := c.checkWithKey(ctx, key, text)
	if errors.Is(err, errInvalidKey) {
		key, kerr := c.getPassportKey(ctx, true)
		if kerr != nil {
			return nil, kerr
		}
		return c.checkWithKey(ctx, key, text)
	}
	return res, err
}

func (c *Checker) checkWithKey(ctx context.Context, passportKey, text string) (*Response, error) {
	q := url.Values{
		"passportKey":     {passportKey},
		"where":           {"nexearch"},
		"color_blindness": {"0"},
		"q":               {text},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checkerURL+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", defaultUserAgent)
	req.Header.Set("Referer", "https://search.naver.com/search.naver?query="+url.QueryEscape(searchQuery))
	req.Header.Set("Accept-Language", "ko-KR,ko;q=0.9,en-US;q=0.8,en;q=0.7")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hanspell: status %d: %s", resp.StatusCode, string(body))
	}

	var payload struct {
		Message struct {
			Error  string `json:"error"`
			Result *struct {
				ErrataCount int    `json:"errata_count"`
				OriginHTML  string `json:"origin_html"`
				HTML        string `json:"html"`
				NoTagHTML   string `json:"notag_html"`
			} `json:"result"`
		} `json:"message"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("hanspell: decode failed: %w", err)
	}

	if payload.Message.Error != "" {
		if strings.Contains(payload.Message.Error, "유효한 키가 아닙니다") {
			return nil, errInvalidKey
		}
		return nil, fmt.Errorf("hanspell: api error: %s", payload.Message.Error)
	}
	if payload.Message.Result == nil {
		return nil, errors.New("hanspell: empty result")
	}

	return &Response{
		ErrataCount: payload.Message.Result.ErrataCount,
		OriginHTML:  payload.Message.Result.OriginHTML,
		HTML:        payload.Message.Result.HTML,
		Corrected:   payload.Message.Result.NoTagHTML,
	}, nil
}

func (c *Checker) getPassportKey(ctx context.Context, forceRefresh bool) (string, error) {
	if !forceRefresh {
		c.mu.RLock()
		key, fetched := c.passportKey, c.keyFetched
		c.mu.RUnlock()
		if key != "" && time.Since(fetched) < passportKeyTTL {
			return key, nil
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if !forceRefresh && c.passportKey != "" && time.Since(c.keyFetched) < passportKeyTTL {
		return c.passportKey, nil
	}

	key, err := c.fetchPassportKey(ctx)
	if err != nil {
		// Keep stale key as a fallback if we already have one.
		if c.passportKey != "" {
			return c.passportKey, nil
		}
		return "", err
	}

	c.passportKey = key
	c.keyFetched = time.Now()
	return key, nil
}

func (c *Checker) fetchPassportKey(ctx context.Context) (string, error) {
	targets := []string{searchURLPC, searchURLMob}
	for _, base := range targets {
		u := base + "?query=" + url.QueryEscape(searchQuery)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return "", err
		}
		req.Header.Set("User-Agent", defaultUserAgent)
		req.Header.Set("Accept-Language", "ko-KR,ko;q=0.9,en-US;q=0.8,en;q=0.7")

		resp, err := c.client.Do(req)
		if err != nil {
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			continue
		}

		if m := passportKeyRe.FindSubmatch(body); len(m) >= 2 {
			return string(m[1]), nil
		}
	}
	return "", errors.New("hanspell: failed to extract passportKey")
}
