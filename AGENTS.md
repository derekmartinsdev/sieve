# Agent Rules — Sieve DSL Transpiler

## Context Budget Rules (MANDATORY)

### Rule 1: No Full-File Reads on Large Files
Files larger than 100 lines MUST NOT be read in full. Use:
- `grep -n "pattern" file` to find line numbers
- `Read` with `offset` and `limit` parameters
- `bash sed -n 'start,endp' file` for exact line ranges

### Rule 2: Pause After Summaries
After generating a summary or analysis, STOP and wait for confirmation before writing code.

### Rule 3: Production vs Test Isolation
- Production code changes → one session/task
- Test code changes → separate session/task
- Never modify production and test code in the same prompt

### Rule 4: Documentation On-Demand
Do NOT load all documentation files at once. Load only the specific ADR or doc relevant to the current task.

## Project Structure

```
sieve/
├── token/          # Token types + keyword map
│   ├── token.go    # 113 lines
│   └── token_test.go
├── lexer/          # Indentation-aware tokenizer
│   ├── lexer.go    # 222 lines
│   └── lexer_test.go
├── ast/            # AST node definitions + visitor
│   ├── ast.go      # 173 lines
│   └── ast_test.go
├── parser/         # Recursive descent parser (split by domain)
│   ├── parser.go           # Core: struct, helpers, parseSection, parseFrom (~320 lines)
│   ├── parser_json.go      # JSON: parseExtract, parseJsonSelect, parseExplode (~150 lines)
│   ├── parser_expr.go      # Expr: parseSelect, parseTransformBlock (~230 lines)
│   ├── parser_relational.go # Joins: parseDerivedSection (~120 lines)
│   ├── parser_io.go        # I/O: parseSink (~55 lines)
│   └── parser_test.go
├── codegen/        # PySpark code generator (split by domain)
│   ├── codegen.go           # Core: Generate, generateSection, generateTransform (~250 lines)
│   ├── codegen_functions.go # Expr builders: buildJsonFieldExpr, emitComputedColumn (~150 lines)
│   ├── codegen_dataframe.go # PySpark ops: generateSelect, generateJoin, generateSink (~100 lines)
│   └── codegen_test.go
├── cmd/transpiler/ # CLI entry point
│   └── main.go     # 98 lines
├── docs/           # ADRs, code reviews, strategic analysis
├── tests/          # 20 integration .sieve files
└── test.pipeline.sieve  # Main end-to-end test
```

## Security Rules

- All `fmt.Fprintf` format strings use `%q` for user-provided values (Go auto-escapes)
- `escapeRegexMeta()` handles ALL regex meta-characters (not just `.`)
- `schemaFromAs()` always returns a safe default (never raw user input)
- Identifiers validated with `isValidIdent()`: `^[a-zA-Z_][a-zA-Z0-9_]*$`

## Build Commands

```
make build          # go build -o sieve ./cmd/transpiler/
make test           # go test ./... -count=1
make test-cover     # go test ./... -cover -coverprofile=coverage.out
make quality-gate   # gofmt + vet + lint + gosec + test + build
make lint           # golangci-lint run
make sec            # gosec -quiet ./...
```

## Code Conventions

- Zero external dependencies (Go stdlib only)
- MixedCaps naming (not snake_case)
- Tab indentation (gofmt standard)
- Production code never touches test files
- Each domain file has its own import block (no shared imports)