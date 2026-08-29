# Lexer and Token System

## What is IDENT?

`IDENT` is short for **identifier** — a standard term in compiler/language theory. It represents a user-defined name: section names, field names, aliases, column names.

It is **not** short for "indentation". The token for indentation would be `INDENT` (with an `N`), used by languages like Python.

## How indentation works in this DSL

Indentation is **not a token type**. It is a property of every token via the `Column` field:

```go
type Token struct {
    Type    TokenType  // IDENT, FROM, EXTRACT, JSON, etc.
    Literal string     // "position", "from", "json", etc.
    Line    int        // 1-based line number
    Column  int        // 1-based column number (first char = 1)
}
```

Parser rules:
- **Column ≤ 1**: token is at the start of a line → section name
- **Column ≥ 5**: token is indented → body content (field, keyword, value)

### Example

```
position          ← column 1: IDENT "position" → section name
    from s3       ← column 5: FROM, column 10: S3 → body
        bucket x  ← column 9: BUCKET, column 16: IDENT "x" → body
```

## Token types

| Token type | Purpose | Example |
|-----------|---------|---------|
| `IDENT` | Identifier (section name, field, alias) | `position`, `name`, `trade_id` |
| `INT` | Integer literal | `1`, `18`, `256` |
| `FLOAT` | Float literal | `18.2` |
| `STRING` | Quoted string literal | `"R$"`, `"."` |
| `DOT` | Dotted path separator | `client.name` |
| `JOIN` | Join operator | `/\` |
| `ARROW` | Transform/join arrow | `->` |
| `PIPE` | Transform chain separator | `\|` |
| Keywords | Reserved words | `from`, `extract`, `json`, `select`, `cast`, etc. |

## Error handling

When the parser encounters unexpected syntax, it reports:
- **Line and column** of the error
- **What was expected** vs **what was found**
- **Specific suggestion** on how to fix

Example:
```
line 5 col 9: expected 'json' after 'extract', got 'select'.
Did you mean 'extract json select'?
```

## Future: Tuning suggestions

Based on the pipeline structure, the transpiler will:
- Detect excessive joins → suggest `spark.sql.shuffle.partitions` tuning
- Detect window functions → suggest switching to storage memory
- Detect large broadcasts → suggest `spark.sql.autoBroadcastJoinThreshold`
- Generate a tuning report alongside the PySpark code