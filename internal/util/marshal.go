package util

import (
	"bytes"
	"encoding/json"
)

// MarshalNoEscape behaves like json.Marshal but keeps <, >, & intact.
func MarshalNoEscape(v any, indent bool) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if indent {
		enc.SetIndent("", "  ")
	}
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil // drop trailing newline
}
