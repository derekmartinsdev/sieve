package main

import (
    "fmt"
    "os"

    "github.com/derekmartinsdev/sieve/ast"
    "github.com/derekmartinsdev/sieve/codegen"
    "github.com/derekmartinsdev/sieve/lexer"
    "github.com/derekmartinsdev/sieve/parser"
    "github.com/derekmartinsdev/sieve/semantic"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Fprintf(os.Stderr, "Usage: %s <input.dsl>\n", os.Args[0])
        os.Exit(1)
    }

    input, err := os.ReadFile(os.Args[1])
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
        os.Exit(1)
    }

    l := lexer.New(string(input))
    p := parser.New(l)
    prog := p.ParseProgram()

    if len(p.Errors()) > 0 {
        fmt.Fprintf(os.Stderr, "Parse errors:\n")
        for _, e := range p.Errors() {
            fmt.Fprintf(os.Stderr, "  %s\n", e)
        }
        // still continue to show partial AST
    }

    // Semantic analysis
    analyzer := semantic.New()
    valid := analyzer.Analyze(prog)
    if !valid {
        fmt.Fprintf(os.Stderr, "\nSemantic errors:\n")
        for _, e := range analyzer.Errors() {
            fmt.Fprintf(os.Stderr, "  %s\n", e)
        }
    }

    // Code generation
    gen := codegen.New()
    result := gen.Generate(prog)

    // Write output
    outPath := os.Args[1] + ".py"
    if err := os.WriteFile(outPath, []byte(result), 0644); err != nil {
        fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
        os.Exit(1)
    }

    fmt.Printf("Generated: %s\n", outPath)
    _ = ast.Program{} // ensure import is used
}
