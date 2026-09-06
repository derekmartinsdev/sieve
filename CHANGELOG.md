# Changelog

All notable changes to the DuckDB DeltaLake Spark Transpiler.

## [Unreleased] — feat/spark-transpiler

### Added

- **Lexer** (`token/`, `lexer/`): indentation-aware tokenizer supporting identifiers, keywords, strings, numbers, operators, dots, pipes, arrows, and join symbols (`/\`).
- **Parser** (`parser/`, `ast/`): recursive descent parser producing an AST from `.sieve` DSL files.
- **Codegen** (`codegen/`): PySpark Python code generator from AST.
- **CLI** (`cmd/transpiler/`): entry point — `transpiler <input.sieve>`, auto-detects file type from extension.
- **S3 delta source reads**: `from s3 bucket/region/prefix/format delta`.
- **Derived sources**: `from otherSection` — reuses a previously defined DataFrame.
- **JSON struct extraction**: `json select message` → `F.get_json_object(F.col("message"), "$.field")`.
- **JSON explode**: `json explode message.path array` → `F.explode(F.from_json(F.get_json_object(...), "array<string>"))`.
- **JSON extract**: `json extract col.path array` → explode from existing column.
- **Computed columns**: `quantity * price financeiro decimal(18,2)` → `(F.col("quantidade") * F.col("preco")).cast("decimal(18,2)")` emitted as `.withColumn()` after select.
- **Join syntax**: `A /\ B -> cond, left` and `A (alias) /\ B (alias) -> cond` with inner/left/right join types.
- **Select statements**: `select path alias type` with optional commas and types.
- **Transform chain**: `transform alias = source | cast(...) | default(...) | replace(...) | prefix(...) | hash()`.
- **Sink to S3**: `to s3 bucket/region/prefix/format delta mode overwrite|append|merge partitioned by col1, col2, col3`.
- **Date functions**: `year()`, `month()` → `F.date_format()`, `day()` → `F.dayofmonth()`.
- **File extension support**: `.pipeline.sieve`, `.transform.sieve`, `.quality.sieve`, `.schedule.sieve`.
- **Makefile**: `build`, `test`, `vet` targets.
- **Go unit tests**: 27 tests across 5 packages (token, lexer, ast, parser, codegen).
- **Integration tests**: 20 `.transform.sieve` files with generated PySpark output.
- **Visitor pattern**: `ast.Walk(v Visitor, node Node)` for AST traversal.
- **BinaryExpr AST node**: type-safe computed expressions (Left, Operator, Right).
- **CLI flags**: `-o output.py`, `--version`, stdin support.
- **io.Writer output**: codegen accepts `io.Writer` for flexible output targets.
- **ADR-003**: window functions and temporal computations (Proposed).
- **Canonical examples**: 4 AI-First reference examples (rolling windows, deduplication, lag/lead, pipeline orchestration).

### Fixed

- **Lexer — tab handling** (`lexer/lexer.go`): `\t` now advances column by 4, not 1. Fixes indentation-based section detection with tabs.
- **Lexer — `\r\n` handling**: CR+LF counted as single newline, not double.
- **Lexer — string reader deduplication**: `readString()` and `readSingleQuotedString()` merged into `readDelimitedString(delim)`.
- **Lexer — FLOAT emission**: `readNumber()` now returns `token.FLOAT` for decimal numbers (was falling through to INT).
- **Token — dead constants removed**: `COLON`, `INNER`, `BY` declared but never used by lexer or parser.
- **Token — MESSAGE/ARRAY removed from keywords**: context-sensitive tokens treated as IDENTs, not reserved words.
- **AST — ComputedExpr**: changed from `string` to `*BinaryExpr` with `Left`, `Operator`, `Right` fields. Eliminates panic risk from `strings.Split(..., " * ")` and supports `+`, `-`, `/` operators.
- **AST — Transforms moved**: from `SelectStmt.Transforms` to `Section.Transforms` (correct domain model).
- **Parser — isTypeKeyword signature**: changed from `(token.Token)` to `(token.TokenType)` for consistency with other type-checking functions.
- **Parser — FLOAT/INT in type checks**: `token.FLOAT` and `token.INT` added to `isTypeKeyword` switch.
- **Parser — error messages**: `parseExplode()` and `parseJsonExtract()` now report errors when `array` keyword is missing (was silently returning `nil`).
- **Parser — shared field alias logic**: extracted `parseAlias()` helper, used by both `parseExtractFields` and `parseSelectFields`.
- **Parser — indentation fix**: long alias-detection condition line reformatted (was unindented, breaking `go fmt`).
- **Codegen — io.Writer**: `Generate(io.Writer, *ast.Program) error` — writes to any writer, not just `strings.Builder`.
- **Codegen — withColumn warning**: generates `# ⚠️ N withColumn calls detected` comment when ≥ 5 transform steps.
- **Codegen — ComputedExpr**: uses `*ast.BinaryExpr` fields instead of string split, no panic risk.
- **CLI — flags**: `-o <file>`, `--version`, stdin support (reads from stdin when no file argument).
- **CLI — error handling**: codegen errors now propagated to exit code.

### Created

| File | Lines | Purpose |
|------|-------|---------|
| `go.mod` | 3 | Go 1.27 module |
| `token/token.go` | ~100 | Token types + keyword map |
| `token/token_test.go` | 30 | Keyword lookup tests |
| `lexer/lexer.go` | ~220 | Indentation-aware tokenizer |
| `lexer/lexer_test.go` | 120 | 9 tests: tokens, comments, strings, operators, numbers, dates, EOF, tabs, mixed indentation |
| `ast/ast.go` | ~130 | AST node definitions + visitor + BinaryExpr |
| `ast/ast_test.go` | 80 | 5 tests: interfaces, structure, BinaryExpr, visitor |
| `parser/parser.go` | ~720 | Recursive descent parser |
| `parser/parser_test.go` | 150 | 9 tests: sections, computed columns, joins, transforms, date functions, 4 error scenarios |
| `codegen/codegen.go` | ~340 | PySpark code generator (io.Writer) |
| `codegen/codegen_test.go` | 100 | 6 tests: imports, get_json_object, computed, explode, transform, sink |
| `cmd/transpiler/main.go` | ~80 | CLI with -o, --version, stdin |
| `Makefile` | 11 | Build/test/vet targets |
| `.gitignore` | 4 | Ignore patterns |
| `test.pipeline.sieve` | 117 | End-to-end test DSL |
| `tests/transform/pyspark/` | 20 files | Integration test suite |
| `docs/adr/001-architecture.md` | 20 | Architecture decisions |
| `docs/adr/002-pipeline-orchestration.md` | 68 | Pipeline orchestration ADR |
| `docs/adr/003-window-functions.md` | 80 | Window functions & temporal computations ADR |
| `docs/lexer.md` | 67 | Token system documentation |
| `docs/code-review.md` | 326 | Comprehensive code review |
| `docs/canonical-examples.md` | 120 | AI-First canonical examples |

### Design Decisions

- **Zero external dependencies**: only Go stdlib (`fmt`, `strings`, `os`, `io`).
- **Indentation-based section detection**: sections start at column 1, body at column 5+.
- **JSON extraction**: uses `get_json_object` + `from_json` pattern, not struct column access.
- **Computed columns**: emitted as separate `.withColumn()` after the `.select()` to ensure type casts are applied before arithmetic.
- **Derived sources**: when `from` references a section name, fields use `F.col()` directly (no `get_json_object`).
- **Transform defaults**: use `F.coalesce()` instead of `F.when().isNull()`.
- **BinaryExpr in AST**: type-safe computed expressions, not fragile string splits.
- **io.Writer pattern**: idiomatic Go codegen output, compatible with files, buffers, and test assertions.

### Concepts Documented

- **Dead tokens**: constants declared but never used by lexer/parser (COLON, INNER, BY). Removed to reduce noise.
- **Silent nil returns**: parser functions returning `nil` without error accumulation. Fixed with explicit error messages.
- **Magic strings**: hardcoded values like `"exploded"`, `"array<string>"`, `"s3a://"`. Documented for future constant extraction.
- **Tab-to-space conversion**: `\t` in DSL source advances column by 4 to match visual indentation. Critical for section detection.
- **CR+LF handling**: Windows line endings (`\r\n`) counted as single newline, preventing double line-counting.
- **withColumn proliferation**: chain of ≥ 5 `withColumn` calls can cause Catalyst optimizer StackOverflowError in Spark. Warning comment generated.

### Not Yet Implemented

- Airflow DAG generation (`.schedule.sieve`)
- Data quality rules (`.quality.sieve`)
- Transform-only pipelines (`.transform.sieve`)
- Pipeline orchestration (`.pipeline.sieve` referencing other `.sieve` files) — see [ADR-002](./docs/adr/002-pipeline-orchestration.md)
- Window functions (rolling windows, deduplication, lag/lead) — see [ADR-003](./docs/adr/003-window-functions.md)
- Additional operators beyond `*` (multiplication) — AST now supports `+`, `-`, `/` via BinaryExpr
- `$` path shorthand in `get_json_object` (always generates `$.` prefix)
- Multi-engine codegen (Spark, Airflow, DuckDB) — see [ADR-002](./docs/adr/002-pipeline-orchestration.md)
- Auto-tuning suggestions (shuffle partitions, broadcast threshold, storage memory) — see [ADR-002](./docs/adr/002-pipeline-orchestration.md)
- Canonical AI-First examples — see [docs/canonical-examples.md](./docs/canonical-examples.md)