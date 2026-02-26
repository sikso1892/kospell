// Package local provides a spell-checker backend backed by the hunspell binary.
// It communicates via the ispell-compatible pipe protocol (-a flag).
package local

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"unicode"

	"github.com/Alfex4936/kospell/internal/model"
	"github.com/Alfex4936/kospell/internal/util"
)

// Hunspell wraps a running hunspell process in ispell-compatible pipe mode.
type Hunspell struct {
	stdin io.WriteCloser
	out   *bufio.Reader
	mu    sync.Mutex
}

// New starts a hunspell subprocess.
// dictDir: directory containing <lang>.aff / <lang>.dic  (pass "" to use system dictionary).
// lang:    dictionary name, e.g. "ko".
func New(dictDir, lang string) (*Hunspell, error) {
	dictArg := lang
	if dictDir != "" {
		aff := filepath.Join(dictDir, lang+".aff")
		dic := filepath.Join(dictDir, lang+".dic")
		if _, err := os.Stat(aff); err != nil {
			return nil, fmt.Errorf("local: hunspell dict missing: %s", aff)
		}
		if _, err := os.Stat(dic); err != nil {
			return nil, fmt.Errorf("local: hunspell dict missing: %s", dic)
		}
		dictArg = filepath.Join(dictDir, lang)
	}

	cmd := exec.Command("hunspell", "-d", dictArg, "-a", "-i", "UTF-8")
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("local: stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("local: stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("local: hunspell start (is hunspell installed?): %w", err)
	}

	h := &Hunspell{
		stdin: stdin,
		out:   bufio.NewReader(stdout),
	}
	// Discard the initial banner: "Hunspell x.y.z\n"
	if _, err := h.out.ReadString('\n'); err != nil {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
		return nil, fmt.Errorf("local: hunspell init failed: %w", err)
	}

	return h, nil
}

// CheckText tokenizes text, checks each word with hunspell, and returns corrections.
func (h *Hunspell) CheckText(text string) ([]model.Correction, error) {
	tokens := tokenize(text)
	if len(tokens) == 0 {
		return nil, nil
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	var out []model.Correction
	for _, tok := range tokens {
		correct, suggest, err := h.checkWord(tok.word)
		if err != nil {
			return nil, err
		}
		if correct {
			continue
		}

		dists := make([]int, len(suggest))
		for i, s := range suggest {
			dists[i] = util.Levenshtein(tok.word, s)
		}

		out = append(out, model.Correction{
			Start:     tok.start,
			End:       tok.end,
			Origin:    tok.word,
			Suggest:   suggest,
			Distances: dists,
		})
	}
	return out, nil
}

// checkWord sends one word to hunspell and parses the response.
// Ispell pipe protocol:
//
//   - → correct
//   - …     → correct compound
//     & w n o: s1, s2  → misspelled, suggestions
//     # w o   → misspelled, no suggestions
func (h *Hunspell) checkWord(word string) (correct bool, suggest []string, err error) {
	if _, err = fmt.Fprintf(h.stdin, "^%s\n", word); err != nil {
		return false, nil, err
	}

	for {
		line, e := h.out.ReadString('\n')
		if e != nil && e != io.EOF {
			return false, nil, e
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break // blank line = end of result for this word
		}

		switch line[0] {
		case '*', '+': // correct / root found
			correct = true
		case '-': // correct compound
			correct = true
		case '&': // misspelled with suggestions: & word count offset: s1, s2
			correct = false
			if idx := strings.Index(line, ": "); idx != -1 {
				for _, s := range strings.Split(line[idx+2:], ", ") {
					if s = strings.TrimSpace(s); s != "" {
						suggest = append(suggest, s)
					}
				}
			}
		case '#': // misspelled, no suggestions
			correct = false
		}
	}
	return
}

// wordToken is a word with its rune offsets in the original text.
type wordToken struct {
	word  string
	start int // inclusive rune offset
	end   int // exclusive rune offset
}

// tokenize splits text into word tokens (letter/digit runs), tracking rune offsets.
func tokenize(text string) []wordToken {
	runes := []rune(text)
	var tokens []wordToken
	i := 0
	for i < len(runes) {
		if !isWordChar(runes[i]) {
			i++
			continue
		}
		start := i
		for i < len(runes) && isWordChar(runes[i]) {
			i++
		}
		tokens = append(tokens, wordToken{
			word:  string(runes[start:i]),
			start: start,
			end:   i,
		})
	}
	return tokens
}

func isWordChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsNumber(r) || r == '\''
}
