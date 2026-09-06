# Compiler Theory Applied to Sieve

## Why You Got This Done In 4 Hours When It Would Take A Semester

The short answer: you didn't build a full compiler. You built a **transpiler** — source-to-source translation. This removes the entire back-end (register allocation, instruction selection, code generation to assembly). What's left is exactly what you did:

```
Source DSL (.sieve) → Tokens → AST → Target Code (PySpark .py)
```

But even then, there are **five distinct compiler theory concepts** at play. Let me walk through each one and how they map to your code.

---

## 1. Lexical Analysis (Lexer/Tokenizer) — `lexer/lexer.go`

### Concept: Finite Automaton

Every lexer is a **deterministic finite automaton** (DFA). You start at state 0, read one character, transition to a new state, repeat until you emit a token.

Your lexer has these states:

```
State 0 (start):
  if ch is letter → State IDENTIFIER
  if ch is digit  → State NUMBER
  if ch is '"'    → State STRING
  if ch is '/'    → State SLASH
  if ch is '-'    → State DASH
  if ch is '|'    → State PIPE
  if ch is '='    → State ASSIGN (emit, return to 0)
  ...

State IDENTIFIER:
  while letter/digit/_   → stay in IDENTIFIER
  otherwise              → emit IDENT, return to 0

State NUMBER:
  while digit/./-        → stay in NUMBER
  has '.' and '-'        → emit as DATE (special case)
  has '.'                → emit as FLOAT
  otherwise              → emit as INT

State DASH:
  if next is '>'         → emit ARROW, return to 0
  otherwise              → restart as NUMBER (negative number)

State SLASH:
  if next is '\'         → emit JOIN, return to 0
  if next is '/'         → skipComment, return to 0
  otherwise              → emit ILLEGAL
```

**Where this is in your code:** `lexer/lexer.go:38-130` — a giant `switch l.ch` that implements the DFA.

### Concept: Lookahead

You can't decide what a token is from one character alone. `-` could be ILLEGAL, ARROW (`->`), or part of a date/number. `/` could be JOIN (`/\`), comment (`//`), or ILLEGAL. The lexer uses **`peekChar()`** — look one character ahead without consuming — to decide.

**Where:** `lexer/lexer.go:67-82` — `peekChar()` used for `->`, `/\`, `//`.

### Concept: Keywords vs Identifiers

Keywords (`from`, `select`, `json`, `cast`) and identifiers (`position`, `trade`, `user_id`) look identical to the lexer. You distinguish them **after** tokenizing: if the literal is in the keyword map, emit a keyword token; otherwise, emit IDENT.

**Where:** `lexer/lexer.go:103-108` — `token.LookupIdent(tok.Literal)`, which checks `token/keywords` map.

### Concept: Column Tracking for Indentation

Python uses `INDENT`/`DEDENT` tokens. Sieve is simpler: every token carries its `Column` (1-based character position). The **parser** uses column to decide "is this a new section?" (column ≤ 1) vs "is this body content?" (column ≥ 5). The lexer just counts columns; the parser interprets them.

**Where:** `lexer/lexer.go:137-144` — `skipWhitespace()` increments `l.column` and `l.line`.

---

## 2. Syntax Analysis (Parser) — `parser/parser.go`

### Concept: Recursive Descent Parsing

Recursive descent means: every grammar rule becomes one function. The functions call each other recursively, mirroring the grammar structure:

```
parseProgram()
  → parseSection()

parseSection()
  → parseFrom()
  → parseExtract()
    → parseJsonSelect()
      → parseExtractFields()
  → parseSelect()
    → parseSelectFields()
  → parseDerivedSection()
    → parseJoinCondition()
```

Each function **consumes tokens** and **builds AST nodes**. This is the Pratt parsing style — simple, readable, no table generation.

**Where:** Basically all of `parser/parser.go`.

### Concept: Predictive Parsing (LL(1))

The parser uses **1 token of lookahead** to decide which rule to apply. That's what `curToken` and `peekToken` are:

```go
type Parser struct {
    curToken  token.Token   // current token
    peekToken token.Token   // next token (lookahead)
}
```

Example of LL(1) decision in `parseFrom()`:
```go
if p.curTokenIs(token.S3) {
    source.From = "s3"      // "from s3" path
} else if p.curTokenIs(token.IDENT) {
    source.From = p.curToken.Literal  // "from otherSection"
}
```

**Where:** `parser/parser.go:53-55` — `nextToken()` advances the lookahead window.

### Concept: Error Recovery vs Error Reporting

Full compilers do **error recovery** — skip to a synchronization point and continue parsing. Sieve does **error reporting** only: stop and accumulate error messages. This is simpler but means one syntax error can mask others.

```go
func (p *Parser) error(msg string) {
    p.errors = append(p.errors, fmt.Sprintf("line %d col %d: %s", ...))
}
```

**Where:** `parser/parser.go:81-83`.

### The Bug I Just Fixed: Token Boundary Detection

The duplicate-field bug was a **token boundary detection failure**. The parser couldn't tell where one field ended and the next began, because:

1. `parseAlias()` saw `string` (a type keyword) and returned `""` (no alias)
2. But the type-reading code also advanced the token
3. The field-loop saw the next IDENT and thought it was a new field's source

The fix: `parseAlias(sameLine int)` — if the potential alias is on a different line than the source, it's not an alias; it's the next field. This is **indentation-aware parsing**, a concept from Python's grammar.

---

## 3. Abstract Syntax Tree (AST) — `ast/ast.go`

### Concept: Heterogeneous AST

Not all nodes are the same type. The AST uses Go interfaces to handle this:

```go
type Node interface { nodeMarker() }
type Statement interface { Node; statementMarker() }
```

A `Program` contains `[]Statement`. Each `Statement` can be a `*Section`. A `Section` contains `*Source`, `[]*Extract`, `[]*Join`, `*SelectStmt`, `*Sink`, `[]TransformChain`. This is a **heterogeneous tree** — different node types at different levels.

### Concept: Binary Expression Tree

Computed columns like `quantity * price` are not stored as strings. They're stored as:

```go
type BinaryExpr struct {
    Left     string   // "quantity"
    Operator string   // "*"
    Right    string   // "price"
}
```

This is the minimal form of an expression tree. A real compiler would have `Expr` interface with `BinaryExpr`, `UnaryExpr`, `Literal`, `Variable`, etc. Sieve keeps it simple because the DSL only has one operator (`*`) for now.

### Concept: Visitor Pattern

The AST has a `Walk()` function that traverses every node:

```go
func Walk(v Visitor, node Node) {
    if !v.Visit(node) { return }
    switch n := node.(type) {
    case *Program:
        for _, s := range n.Statements { Walk(v, s) }
    case *Section:
        // walk children...
    }
}
```

This is the **visitor pattern** from the Gang of Four. It lets you add new operations (pretty-printing, optimization, analysis) without modifying the AST.

**Where:** `ast/ast.go` — the `Visitor` interface and `Walk()` function.

---

## 4. Code Generation — `codegen/codegen.go`

### Concept: Tree-Walking Code Generator

The codegen is a **tree-walking code generator**: it traverses the AST and emits code string by string. No intermediate representation (IR), no optimization passes, no register allocation. Just `fmt.Fprintf(w, ...)` for each AST node.

```go
func generateSection(w io.Writer, sec *ast.Section) error {
    // emit: df_name = spark.read.format(...).load(...)
    // for each Extract: emit get_json_object / explode / from_json
    // for each Join: emit alias().join()
    // for each Transform: emit withColumn()
    // for each Select: emit select()
    // for Sink: emit .write.format().mode().save()
}
```

### Concept: Multi-Engine via Codegen Interface

The real compiler-design concept here is the **strategy pattern**: same AST, different code generators:

```
         AST
          │
    ┌─────┼─────┬──────────┐
    ▼     ▼     ▼          ▼
  Spark  SQL  Airflow   DuckDB
  (.py)  (.sql) (.py)   (.sql)
```

Each engine is a separate codegen that implements the same interface:
```go
type Generator interface {
    Generate(w io.Writer, prog *ast.Program) error
}
```

Currently only Spark is implemented, but the architecture supports plugging in any engine.

---

## 5. The Pipeline Flow (End to End)

```
INPUT: position
           from s3
               bucket prd_tables
               region us-east-1
               prefix position
               format delta
           extract
               json select message
                   client.name name string
                   client.id id bigint

═══════════════════════════════════════════
STEP 1: LEXICAL ANALYSIS (lexer.go)
═══════════════════════════════════════════

"position"     → IDENT     line=1 col=1
"from"         → FROM      line=2 col=5
"s3"           → S3        line=2 col=10
"bucket"       → BUCKET    line=3 col=9
"prd_tables"   → IDENT     line=3 col=16
"region"       → REGION    line=4 col=9
"us-east-1"    → ...       (string token)
"prefix"       → PREFIX    line=5 col=9
"position"     → IDENT     line=5 col=16
"format"       → FORMAT    line=6 col=9
"delta"        → DELTA     line=6 col=16
"extract"      → EXTRACT   line=8 col=5
"json"         → JSON      line=9 col=9
"select"       → SELECT    line=9 col=14
"message"      → MESSAGE   line=9 col=21
"client"       → IDENT     line=10 col=13
"."            → DOT       line=10 col=19
"name"         → IDENT     line=10 col=20
"name"         → IDENT     line=10 col=25
"string"       → TYPE_STR  line=10 col=30
"client"       → IDENT     line=11 col=13
"."            → DOT       line=11 col=19
"id"           → IDENT     line=11 col=20
"id"           → IDENT     line=11 col=23
"bigint"       → BIGINT    line=11 col=26

═══════════════════════════════════════════
STEP 2: SYNTAX ANALYSIS (parser.go)
═══════════════════════════════════════════

createSection("position")
  |
  ├─ parseFrom() → Source{From:"s3", Bucket:"prd_tables",
  │                       Region:"us-east-1", Prefix:"position",
  │                       Format:"delta"}
  │
  └─ parseExtract() → Extract{JsonSelect: {
        Path: "message",
        Fields: [
          FieldDef{Name:"name",  Source:"client.name",  Alias:"name",  DataType:"string"},
          FieldDef{Name:"id",    Source:"client.id",    Alias:"id",    DataType:"bigint"},
        ]
      }}

═══════════════════════════════════════════
STEP 3: AST (ast.go)
═══════════════════════════════════════════

Program {
  Statements: [
    Section {
      Name: "position"
      Source: Source{From:"s3", Bucket:"prd_tables", ...}
      Extracts: [
        Extract {
          JsonSelect: {
            Path: "message"
            Fields: [
              FieldDef{Name:"name", Source:"client.name", Alias:"name", DataType:"string"}
              FieldDef{Name:"id",   Source:"client.id",   Alias:"id",   DataType:"bigint"}
            ]
          }
        }
      ]
    }
  ]
}

═══════════════════════════════════════════
STEP 4: CODE GENERATION (codegen.go)
═══════════════════════════════════════════

df_position = spark.read.format("delta").load("s3a://prd_tables/position")

df_position = df_position.select(
    F.get_json_object(F.col("message"), "$.client.name").alias("name").cast("string"),
    F.get_json_object(F.col("message"), "$.client.id").alias("id").cast("bigint")
)
```

---

## Why AI Accelerated This

The hard part of building a compiler is not understanding the concepts — you learned them in college. The hard part is:

1. **Writing 1,795 lines without typos** — AI generates the boilerplate
2. **Debugging edge cases** — AI can run experiments faster than you can type
3. **Remembering API details** — AI knows `F.get_json_object()` takes `(col, path)` not `(path, col)`
4. **Iterating fast** — AI rewrites entire functions in seconds

The 4 hours you spent would have been 40-50 hours without AI. But the **design** — the DSL syntax, the grammar, the architecture decisions — that's yours. AI executed; you architected.