// Command kospell-cli pipes stdin (or a file) through kospell.Check
// and prints the pretty-printed JSON result.
//
// Usage:
//
//	echo "너는나와 ..." | kospell-cli
//	kospell-cli -f text.txt
//	kospell-cli -mode local -dict-dir /path/to/hunspell-dict-ko -lang ko
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/Alfex4936/kospell/internal/local"
	"github.com/Alfex4936/kospell/internal/model"
	"github.com/Alfex4936/kospell/internal/util"
	"github.com/Alfex4936/kospell/kospell"
)

func main() {
	file    := flag.String("f", "", "file to read instead of stdin")
	dict    := flag.String("d", "", "user dictionary JSON file (optional)")
	timeout := flag.Duration("t", 8*time.Second, "overall timeout")
	mode    := flag.String("mode", "nara", "backend: nara | local (hunspell)")
	dictDir := flag.String("dict-dir", "", "hunspell dictionary directory (local mode)")
	lang    := flag.String("lang", "ko", "hunspell dictionary name (local mode)")
	flag.Parse()

	var r io.Reader = os.Stdin
	if *file != "" {
		f, err := os.Open(*file)
		must(err)
		defer f.Close()
		r = f
	}

	data, err := io.ReadAll(r)
	must(err)

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	text := string(data)

	// 사용자 딕셔너리 로드 (선택)
	var d *kospell.Dict
	if *dict != "" {
		d, err = kospell.LoadDict(*dict)
		must(err)
	}

	var res *model.Result

	switch *mode {
	case "local":
		h, herr := local.New(*dictDir, *lang)
		must(herr)
		if d != nil {
			res, err = kospell.CheckLocalWithDict(ctx, text, h, d)
		} else {
			res, err = kospell.CheckLocal(ctx, text, h)
		}
	default: // nara
		if d != nil {
			res, err = kospell.CheckWithDict(ctx, text, d)
		} else {
			res, err = kospell.Check(ctx, text)
		}
	}
	must(err)

	out, _ := util.MarshalNoEscape(res, true)
	fmt.Println(string(out))
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "kospell-cli:", err)
		os.Exit(1)
	}
}
