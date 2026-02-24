package net

import (
	"context"
	"io"
	"net/http"

	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
	fhttp "github.com/bogdanfinn/fhttp"
)

var tlsClient tls_client.HttpClient

func init() {
	c, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(),
		tls_client.WithTimeoutSeconds(10),
		tls_client.WithClientProfile(profiles.Chrome_133),
		tls_client.WithRandomTLSExtensionOrder(),
		tls_client.WithCookieJar(tls_client.NewCookieJar()),
	)
	if err != nil {
		panic("kospell/net: " + err.Error())
	}
	tlsClient = c
}

const base = "https://nara-speller.co.kr"

// NewPOST builds a pre-populated request with Chrome headers.
func NewPOST(ctx context.Context, path string, body any) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+path, body.(io.Reader))
	if err != nil {
		return nil, err
	}

	req.Header = http.Header{
		"Accept":                    {"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8"},
		"Accept-Language":           {"ko-KR,ko;q=0.9,en-US;q=0.8,en;q=0.7"},
		"Accept-Encoding":           {"gzip, deflate, br"},
		"Content-Type":              {"application/x-www-form-urlencoded"},
		"Origin":                    {"https://nara-speller.co.kr"},
		"Referer":                   {"https://nara-speller.co.kr/speller/"},
		"Sec-Ch-Ua":                 {`"Not(A:Brand";v="99", "Google Chrome";v="133", "Chromium";v="133"`},
		"Sec-Ch-Ua-Mobile":          {"?0"},
		"Sec-Ch-Ua-Platform":        {`"Windows"`},
		"Sec-Fetch-Dest":            {"empty"},
		"Sec-Fetch-Mode":            {"cors"},
		"Sec-Fetch-Site":            {"same-origin"},
		"User-Agent":                {ua},
	}
	return req, nil
}

// Do forwards to the TLS spoofing client, converting types as needed.
func Do(req *http.Request) (*http.Response, error) {
	// Convert net/http.Request to fhttp.Request
	fhttpReq := req.Clone(context.Background())
	fhttpReq.RequestURI = ""

	fReq := &fhttp.Request{
		Method:        fhttpReq.Method,
		URL:           fhttpReq.URL,
		Proto:         fhttpReq.Proto,
		ProtoMajor:    fhttpReq.ProtoMajor,
		ProtoMinor:    fhttpReq.ProtoMinor,
		Header:        fhttp.Header(fhttpReq.Header),
		Body:          fhttpReq.Body,
		ContentLength: fhttpReq.ContentLength,
		Host:          fhttpReq.Host,
	}

	// Execute with tls-client
	fResp, err := tlsClient.Do(fReq)
	if err != nil {
		return nil, err
	}

	// Convert fhttp.Response back to net/http.Response
	return &http.Response{
		Status:        fResp.Status,
		StatusCode:    fResp.StatusCode,
		Proto:         fResp.Proto,
		ProtoMajor:    fResp.ProtoMajor,
		ProtoMinor:    fResp.ProtoMinor,
		Header:        http.Header(fResp.Header),
		Body:          fResp.Body,
		ContentLength: fResp.ContentLength,
		Request:       req,
	}, nil
}

const ua = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 " +
	"(KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36"
