package net

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"time"
)

// shared client (keep-alive, TLS session reuse).
var client = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        32,
		MaxIdleConnsPerHost: 16,
		DisableCompression:  false,
		TLSClientConfig:     &tls.Config{MinVersion: tls.VersionTLS12},
	},
}

const base = "https://nara-speller.co.kr"

// NewPOST builds a pre-populated request.
func NewPOST(ctx context.Context, path string, body any) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+path, body.(io.Reader))
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", ua)
	req.Header.Set("X-Forwarded-For", RandV4())
	return req, nil
}

// Do forwards to the shared *http.Client.
func Do(req *http.Request) (*http.Response, error) { return client.Do(req) }

const ua = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 " +
	"(KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36"
