package parse

import (
	"encoding/json"
	"html"
	"strings"

	"github.com/Alfex4936/kospell/internal/model"
)

// Decode converts the raw server JSON into []Correction.
func Decode(raw []byte) ([]model.Correction, error) {
	var wrap []model.RawChunk

	if err := json.Unmarshal(raw, &wrap); err != nil {
		return nil, err
	}
	if len(wrap) == 0 {
		return nil, nil
	}

	// Pre-alloc once
	out := make([]model.Correction, 0, len(wrap[0].ErrInfo))

	for _, e := range wrap[0].ErrInfo {
		// 1) HTML entity → literal rune    (&gt;  → >)
		help := html.UnescapeString(e.Help)
		// 2) <br/>  → newline               (<br/> → \n)
		help = strings.ReplaceAll(help, "<br/>", "\n")

		out = append(out, model.Correction{
			Start:   e.Start,
			End:     e.End,
			Origin:  e.OrgStr,
			Suggest: strings.Split(e.CandWord, "|"),
			Help:    help,
		})
	}
	return out, nil
}
