# Code Review — Spark DSL Transpiler

## Summary

| Metric | Value |
|--------|-------|
| Total lines | 1,263 Go + 117 test DSL |
| Packages | 6 (token, lexer, ast, parser, codegen, cmd) |
| Dependencies | 0 (stdlib only) |
| Test coverage | 0% (no Go tests, only 20 integration `.sieve` files) |
| `go vet` | ✅ clean |
| `go build` | ✅ clean |
| Cyclomatic complexity | 🟡 moderate (parseDerivedSection: ~15, parseTransformBlock: ~14) |

## 1. Lexer (`lexer/lexer.go`) — 229 lines

### ✅ Good

- **Rune-based scanning**: using `[]rune` for UTF-8 source is correct for Go.
- **Column tracking**: `column` field correctly tracks 1-based character position per line.
- **Comment handling**: `//` line comments are skipped with `skipComment()`.
- **Two-character operators**: `->`, `/\`, `..` handled correctly with `peekChar()`.
- **String escaping**: `readString()` handles `\"` escapes via `\` skip.

### 🟡 Warnings

**Number parsing conflates dates with identifiers** (line 109-122):
```go
} else if isDigit(l.ch) || l.ch == '-' {
```
The `-` in the default branch catches `-` as a number start, but `-` alone should be ILLEGAL. A `-` only becomes a number when followed by a digit. Currently `-east` in `us-east-1` is tokenized as `-` (ILLEGAL) + `east` (IDENT) + `-` (ILLEGAL) + `1` (INT). This works but is fragile.

**Recommendation**: `-` followed by a digit → number; `-` alone → ILLEGAL.

**`readNumber` date detection is heuristic** (line 164-188):
```go
if isFloat && hasDash {
    isDate = true
    isFloat = false
}
```
This returns `(string, true, false)` as a date if it has both `.` and `-`. But the token is set to `token.STRING` for dates. The heuristic conflates `2024-01-15` (date) with `18.2-5` (invalid). Not a bug for the current DSL, but fragile for future extensions.

**Empty `readString`/`readSingleQuotedString` deduplication** (lines 191-221): These two functions differ only in the quote character. They could be unified:
```go
func (l *Lexer) readDelimitedString(delim rune) string {
    l.readChar()
    start := l.position
    for l.ch != delim && l.ch != 0 {
        if l.ch == '\\' { l.readChar() }
        l.readChar()
    }
    end := l.position
    if l.ch != 0 { l.readChar() }
    return string(l.input[start:end])
}
```

### ❌ Issues

**`isLetter` accepts ASCII only** (line 223-225):
```go
func isLetter(ch rune) bool {
    return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z'
}
```
This rejects Unicode identifiers (e.g., `café`, `maçã`). Go's `unicode.IsLetter` exists but adds a dependency on `unicode` package. Since the DSL uses ASCII identifiers (section names, field names), this is acceptable for now. Document this limitation.

---

## 2. Token (`token/token.go`) — 115 lines

### ✅ Good

- **`TokenType` as `string`**: simple, debuggable. You can `fmt.Printf("%s", tok)`.
- **`LookupIdent`** with map: O(1) lookup, idiomatic.
- **`keywords`** map: lowercase keys → uppercase token types. Clean separation.

### 🟡 Warnings

**Duplicate `PREFIX` entry** (line 37 and 76):
```go
PREFIX = "PREFIX"   // line 37: token constant
"prefix": PREFIX,   // line 76: keyword map
```
No bug — both are the same constant. Just duplicated in the keyword map.

**`COLON` token is unused** (line 25): Defined but never consumed by parser or lexer. Dead code.

**`MESSAGE` and `ARRAY` are keywords but not really used as tokens** — they appear in the DSL but are consumed by the parser as IDENT-literals via `p.nextToken()` without type checking. The lexer tokenizes them as `MESSAGE`/`ARRAY` but the parser's `parseDottedPath()` reads them by `Literal` not by `Type`.

**Recommendation**: Either use the typed tokens (check `p.curTokenIs(token.MESSAGE)`) or remove them from the keyword map and treat them as context-sensitive IDENTs.

### ❌ Issues

**`INNER` and `BY` don't appear in the keyword map** (lines 62, 54):
```go
INNER = "INNER"  // but no "inner" in keywords map
BY     = "BY"     // but no "by" in keywords map
```
`INNER` is never returned by the lexer — it would be tokenized as IDENT. `BY` is used in `partitioned by` but the parser reads it as IDENT. These are dead constants.

**`FLOAT` token is never generated** (line 18): The lexer's `readNumber()` returns `isFloat=true` but the token type is never set to `FLOAT`. The path `isFloat && hasDash` catches dates first, then `isFloat` alone is unreachable because `readNumber` returns `isFloat=true` only when there's a `.` but no `-`. For `18.2`, the lexer returns `FLOAT`. But the token `FLOAT` is never consumed by the parser — it's only used in `isTypeKeyword` switch which doesn't include `FLOAT`.

**Fix**: Add `token.FLOAT` to `isTypeKeyword` in the parser, or remove it.

---

## 3. AST (`ast/ast.go`) — 117 lines

### ✅ Good

- **`Node` and `Statement` interfaces**: clean marker interface pattern from Thorsten Ball's "Writing an Interpreter in Go".
- **`Section`** as the central construct: everything is a section.
- **`FieldDef`** with `JsonColumn`: tracks which JSON column the field was extracted from.

### 🟡 Warnings

**No `String()` or `Visit()` methods**: Without a visitor or printer, debugging AST is painful. You can't `fmt.Printf("%+v", sec)` and get a useful tree.

**Recommendation**: Add `func (s *Section) String() string` or a `Walk(fn func(Node) bool)` visitor.

**`SelectStmt.Source` is unused** (line 88): Dead field.

**`TransformChain` is stored on `SelectStmt`** (line 87): but transforms are logically part of the section, not the select. The current code works because `parseSection` stashes transforms on a temporary `SelectStmt` even when no select is present. This is a workaround — the model should be `Section.Transforms []TransformChain`, not `SelectStmt.Transforms`.

**`JoinCondition` uses `[]string` for LeftRefs/RightRefs** (line 80-81): but in practice there's always exactly one element. A `[2]string` or dedicated `LeftRef string` / `RightRef string` fields would be cleaner.

---

## 4. Parser (`parser/parser.go`) — 746 lines

### ✅ Good

- **Pratt/recursive descent** hybrid: simple and effective for this DSL.
- **`Column <= 1` for section detection**: clean indentation-based parsing.
- **Error reporting**: `error()`, `errorAt()`, `peekError()` with specific messages and suggestions.
- **`parseDottedPath()`**: reusable, handles multi-segment paths.

### 🟡 Warnings

**Indentation inconsistency on line 302** (go fmt issue):
```go
if p.curTokenIs(token.IDENT) && !p.isTypeKeyword(p.curToken) && !p.peekTokenIs(token.DOT) && p.curToken.Column > 1 && !isDateFunc(p.peekToken) && !p.isTypeKeyword(p.peekToken) {
```
This line is 180+ characters. Go doesn't have a line limit, but readability suffers. Break it:
```go
if p.curTokenIs(token.IDENT) &&
    !p.isTypeKeyword(p.curToken) &&
    !p.peekTokenIs(token.DOT) &&
    p.curToken.Column > 1 &&
    !isDateFunc(p.peekToken) &&
    !p.isTypeKeyword(p.peekToken) {
```

**`isBodyKeyword()` doesn't include `JOIN`** (line 97-103): The `JOIN` token is `/\` which is a two-char operator. It's correct to not include it since joins are parsed inside `parseDerivedSection`, not in the section body loop. But if someone puts `/\` at the body level, it will be silently consumed by `p.nextToken()` in the default case.

**`parseSelectFields()` and `parseExtractFields()` share 90% logic** (lines 284-332 and 343-419): The alias detection, type parsing, and dotted path logic is duplicated. Extract a shared helper.

**Dead code: `expectPeek()`** (line 66-73): Defined but never called. The parser uses `curTokenIs`/`peekTokenIs` directly.

**`parseTransformBlock()` has a hanging `else` clause** (lines 721-723):
```go
} else {
    break
}
```
This is unreachable because the `for` loop condition already checks for `!p.curTokenIs(token.IDENT)` equivalent. The `if p.curTokenIs(token.IDENT)` guard at line 671 makes the else redundant.

### ❌ Issues

**`isTypeKeyword` uses `token.Token` (struct) not `token.TokenType`** (line 106):
```go
func (p *Parser) isTypeKeyword(tok token.Token) bool {
```
But the function `isTypeKeyword` in the same file (line 650) uses `token.TokenType`:
```go
func isTransformKeyword(t token.TokenType) bool {
```
This is inconsistent. `isTypeKeyword` should take `token.TokenType` not `token.Token`. The struct version is only called in one place (line 106-113) and works because `tok.Type` is accessed, but the naming is confusing.

**`parseExplode()` silently returns `nil` on missing `array`** (line 251-253):
```go
if !p.curTokenIs(token.ARRAY) {
    return nil
}
```
No error message. The user writes `json explode message.path` without `array` and gets nothing — no error, no output, silently skipped.

**`parseJsonExtract()` same issue** (line 270-272).

**`parseSink()` doesn't handle `OVERWRITE`/`APPEND`/`MERGE` as keywords after `mode`** (line 499-502): The `MODE` case reads the next token as the mode value (`sink.Mode = p.curToken.Literal`), but the lexer tokenizes `overwrite` as `token.OVERWRITE`, `append` as `token.APPEND`, `merge` as `token.MERGE`. These are keywords, not IDENTs. The `Literal` is correct (`"overwrite"`), so it works, but the intent is unclear — you're reading the literal of a keyword token instead of its type.

**`parseDerivedSection` alias swapping logic is fragile** (lines 586-607): The code replaces the first segment of a dotted path if it matches the section name. But `trade.trade_id` has `parts[0] == "trade"` and `join.Left == "trade"`, so it replaces `trade` with `trade` (no-op). For `tpa.reference`, `parts[0] == "tpa"` and `join.Left == "trade_perAcquisition"`, so no replacement. The condition is correct but the naming is confusing — `parts[0]` is the alias, not the section name.

---

## 5. Codegen (`codegen/codegen.go`) — 362 lines

### ✅ Good

- **`strings.Builder`**: efficient, no allocations.
- **`splitFields()`**: separates computed columns from regular ones.
- **`buildAliasMap()`**: maps source names → aliases for computed column resolution.
- **`formatLit()`**: distinguishes numeric from string literals.

### 🟡 Warnings

**Hardcoded `"array<string>"`** (lines 54, 57, 76, 80): The `from_json` schema is always `"array<string>"`. This works for the current DSL but is wrong for structured arrays. The `ExtractExplode.As` field is `"array"` but never used in codegen.

**Magic strings everywhere**: `"exploded"`, `"message"`, `"array<string>"`, `"inner"`, `"overwrite"` are hardcoded. These should be constants.

**`generateJoin` uses `alias().join(alias(), ...)`** (line 285): The `alias()` is applied to the left DataFrame, but the variable name is used for the alias. This means `df_foo.alias("foo").join(df_bar.alias("bar"), ...)` — the DataFrame name and alias are the same by default. Works but semantically redundant.

**`generateTransform` generates one `withColumn` per step** (lines 312-341): 7 transform steps = 7 `withColumn` calls. This creates a long DAG in Spark. The `default` step uses `F.coalesce()` which is correct, but the chain could be expressed as a single nested expression (as in the reference code).

**No `explodeCol` variable usage** (line 43): `explodeCol` is set but never referenced after the loop. The `jsonCol` variable is used instead.

### ❌ Issues

**`generateSelect` doesn't handle `Function` for `month`/`day` properly** (lines 234-237): `F.date_format` and `F.dayofmonth` are hardcoded with if-else chains. Add a new date function → add another if branch. This should be a map or switch table.

**`emitComputedColumn` only handles `*`** (line 159): `f.ComputedExpr` is split on ` * `. No support for `+`, `-`, `/`, or other operators. The `ComputedExpr` field stores the raw expression string but it's never parsed as an expression — it's just `split`.

**`buildJsonFieldExpr` always prepends `$.`** (line 216): `"$." + source`. For `F.get_json_object(F.col("message"), "$.client.name")` this is correct. But for `F.get_json_object(F.col("exploded"), "$.reference")` the `$.` is also prepended, which is correct. However, if the source is already a full path (e.g., `$.client.name`), it would double the `$.`.

---

## 6. CLI (`cmd/transpiler/main.go`) — 56 lines

### ✅ Good

- **Simple, single-purpose**: no flags, no config, one file in → one file out.
- **`detectType()`**: extensible via suffix list.

### 🟡 Warnings

**No `-o` flag**: output always goes to stdout. Can't write to a file directly.

**No `--version` flag**: `transpiler --version` returns `usage: transpiler <input.sieve>`.

**No stdin support**: must provide a file path. Can't pipe: `cat test.sieve | transpiler -`.

**`detectType` loop is O(n)** (line 44-53): linear scan of 4 suffixes. Trivial for 4, but the pattern is repetitive.

---

## 7. Cross-cutting concerns

### ❌ No tests

**Zero Go unit tests**: all 6 packages have `[no test files]`. The 20 integration `.sieve` files are good smoke tests but don't cover:
- Lexer edge cases (empty input, Unicode, malformed strings)
- Parser error recovery (how does it handle 50% of a file?)
- Codegen corner cases (all transform chain combinations)
- CLI error handling (missing file, permissions)

### 🟡 No `io.Writer` abstraction

The codegen writes to `*strings.Builder` directly. For multi-engine support, the codegen should accept `io.Writer`:
```go
func Generate(w io.Writer, prog *ast.Program) error
```
This allows writing to files, buffers, network, or test assertions.

### 🟡 No visitor pattern

The AST has no walker. Adding a new analysis pass (e.g., "count joins for tuning") requires modifying the codegen or writing a separate traversal. A visitor would allow:
```go
ast.Walk(prog, func(n ast.Node) bool {
    if join, ok := n.(*ast.Join); ok { joinCount++ }
    return true
})
```

### 🟡 Error handling: silent returns

Many parser functions return `nil` or partial structs on error without accumulating errors. Example: `parseExplode()` returns `nil` when `array` is missing — the caller appends this `nil` to `Extracts` and continues. The generated code silently omits the explode.

### 🟡 No incremental parsing

The entire file is read into memory (`os.ReadFile`). For 117-line test files this is fine. For production pipelines with hundreds of sections, consider a streaming parser or at least document the memory limit.

---

## 8. Comparison with other transpilers

| Feature | This project | ANTLR (Go) | sqlc (Go) | HCL (HashiCorp) |
|---------|-------------|-----------|-----------|-----------------|
| Parser style | Hand-written recursive descent | Generated from grammar | Hand-written recursive descent | Hand-written recursive descent |
| AST visitor | ❌ | ✅ (generated) | ✅ | ✅ |
| Error recovery | ⚠️ partial | ✅ (sync) | ✅ | ✅ |
| Multi-engine | ❌ | N/A | ✅ (Go+SQL) | N/A |
| Tests | 0 Go tests | Generated tests | ✅ | ✅ |
| Dependencies | 0 | 1 (antlr runtime) | 0 | ✅ (stdlib) |
| Output flexibility | `string` | Visitor callback | `io.Writer` | `hclwrite` |

**Takeaways**:
- Zero deps is the right call — matches HCL/sqlc philosophy.
- Hand-written parser is fine for a DSL this size — ANTLR would be overkill.
- Missing visitor/error recovery is the biggest gap vs mature tools.
- `io.Writer` output is the idiomatic Go pattern.

---

## 9. Prioritized action items

### 🔴 P0 (bugs)
1. Fix `isTypeKeyword` to accept `token.TokenType`, not `token.Token` struct
2. Add `token.FLOAT` to `isTypeKeyword` or remove it
3. Remove dead `COLON`, `INNER`, `BY` token constants
4. Add error messages to `parseExplode()`/`parseJsonExtract()` when `array` is missing

### 🟡 P1 (design)
5. Add `io.Writer` to codegen signature
6. Move `Transforms` from `SelectStmt` to `Section`
7. Add visitor pattern to AST
8. Add Go unit tests for lexer + parser + codegen

### 🟢 P2 (nice to have)
9. Unify `readString` / `readSingleQuotedString` in lexer
10. Extract shared field parsing logic from `parseSelectFields`/`parseExtractFields`
11. Add `-o`, `--version`, stdin support to CLI
12. Add `String()` methods to AST nodes for debugging
13. Extract magic strings as constants in codegen