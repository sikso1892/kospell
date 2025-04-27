// Command kospell-cli pipes stdin (or a file) through kospell.Check
// and prints the pretty-printed JSON result.
//
// Usage:
//
//	echo "너는나와 ..." | kospell-cli
//	kospell-cli -f text.txt
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/Alfex4936/kospell/internal/util"
	"github.com/Alfex4936/kospell/kospell"
)

func main() {
	file := flag.String("f", "", "file to read instead of stdin")
	timeout := flag.Duration("t", 8*time.Second, "overall timeout")
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

	res, err := kospell.Check(ctx, string(data))
	// fmt.Printf("data = %q\n", string(data))
	// fmt.Printf("result = %+v\n", res)
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
