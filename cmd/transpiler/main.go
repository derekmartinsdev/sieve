package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/derekmartinsdev/sieve/codegen"
	"github.com/derekmartinsdev/sieve/parser"
)

const version = "0.1.0"

func main() {
	var output string
	var showVersion bool

	flag.StringVar(&output, "o", "", "output file path")
	flag.BoolVar(&showVersion, "version", false, "print version and exit")
	flag.Parse()

	if showVersion {
		fmt.Println("transpiler version", version)
		return
	}

	var input io.Reader
	var path string

	args := flag.Args()
	if len(args) >= 1 {
		path = args[0]
		typ, known := detectType(path)
		if !known {
			fmt.Fprintf(os.Stderr, "warning: unrecognized file extension for %q, processing anyway\n", path)
		}
		fmt.Fprintf(os.Stderr, "type: %s\n", typ)
		f, err := os.Open(path) // #nosec G304 — CLI tool reads user-specified input file
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading file %q: %v\n", path, err)
			os.Exit(1)
		}
		defer func() { _ = f.Close() }()
		input = f
	} else {
		input = os.Stdin
	}

	content, err := io.ReadAll(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading input: %v\n", err)
		os.Exit(1)
	}

	p := parser.New(string(content))
	program := p.ParseProgram()

	if errs := p.Errors(); len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintln(os.Stderr, e)
		}
		os.Exit(1)
	}

	var w io.Writer
	if output != "" {
		f, err := os.Create(output) // #nosec G304 — CLI tool writes to user-specified output file
		if err != nil {
			fmt.Fprintf(os.Stderr, "error creating output file %q: %v\n", output, err)
			os.Exit(1)
		}
		defer func() { _ = f.Close() }()
		w = f
	} else {
		w = os.Stdout
	}

	if err := codegen.Generate(w, program); err != nil {
		fmt.Fprintf(os.Stderr, "error generating code: %v\n", err)
		os.Exit(1)
	}
}

func detectType(path string) (string, bool) {
	for _, suffix := range []string{
		".pipeline.sieve",
		".transform.sieve",
		".quality.sieve",
		".schedule.sieve",
	} {
		if strings.HasSuffix(path, suffix) {
			return strings.TrimSuffix(suffix, ".sieve"), true
		}
	}
	return "unknown", false
}
