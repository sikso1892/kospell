package kospell

import "errors"

var (
	// ErrParse signals unexpected HTML/JS structure from upstream.
	ErrParse = errors.New("kospell: could not parse server response")
)
