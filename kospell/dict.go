package kospell

import (
	"encoding/json"
	"os"
)

// Dict is a user dictionary for protecting specific terms from spell-check.
type Dict struct {
	Words []string `json:"words"`
}

// NewDict creates a Dict from the given words.
func NewDict(words ...string) *Dict {
	return &Dict{Words: words}
}

// LoadDict reads a JSON file of the form {"words": ["목제솜틀기", ...]}.
func LoadDict(path string) (*Dict, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var d Dict
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, err
	}
	return &d, nil
}
