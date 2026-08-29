# DSL Transpiler Architecture — PySpark Code Generation

## 1. Overview

**Sieve** is a domain-specific language (DSL) for defining data pipelines. It allows users to declaratively describe data sources, extraction rules, transformations, joins, and outputs in a concise, readable syntax. The transpiler reads Sieve DSL source code and generates executable **PySpark** code.

**Key capabilities:**
- Declare S3-backed data sources with Delta format
- Extract and transform fields from JSON payloads (select, explode, extract)
- Define joins between tables with configurable join types
- Apply pipeline operators (replace, prefix, hash, coalesce, cast, date parts)
- Configure output modes (overwrite, append, merge) and partitioning

---

## 2. Architecture

```
DSL Source ──▶ Lexer ──▶ Tokens ──▶ Parser ──▶ AST ──▶ Semantic Analyzer ──▶
    Optimizer ──▶ Code Generator ──▶ PySpark
```

### Component Details

#### Lexer
Scans raw DSL text and produces a stream of tokens. Handles:
- **Keywords**: `position, from, s3, bucket, region, prefix, format, delta, extract, json, select, explode, array, to, mode, overwrite, append, merge, partitioned, by, year, month, day, left`
- **Identifiers**: table names, field names, aliases (e.g. `client.name`, `party_name`, `cod_fatura_origem`)
- **Literals**: strings (`"prd_tables"`, `"us-east-1"`), numbers (`18`, `2`), decimals (`18,2`)
- **Operators**: `|` (pipeline), `/\` (join), `->` (join condition), `=` (assignment), `:` (type annotation), `,` (separator), `...` (ellipsis shorthand)

Returns a `[]Token` with fields: `Type`, `Literal`, `Line`, `Column`.

#### Parser
Recursive descent parser that consumes tokens and builds an AST. Grammar rules:
- `program → (statement)*`
- `statement → table_def | join_def | output_def`
- `table_def → IDENTIFIER from_source extract_block`
- `from_source → FROM S3 bucket STRING region STRING prefix STRING format STRING`
- `extract_block → EXTRACT json_action`
- `json_action → JSON SELECT IDENTIFIER field_list | JSON EXPLODE IDENTIFIER ARRAY field_list | JSON EXTRACT IDENTIFIER ARRAY field_list`
- `field_list → (IDENTIFIER IDENTIFIER type_annotation? pipeline?)*`
- `join_def → IDENTIFIER JOIN_OP IDENTIFIER ARROW condition join_modifier? select_block`
- `output_def → IDENTIFIER EQ join_expr select_block to_block`
- `pipeline → PIPE func_call (PIPE func_call)*`
- `to_block → TO S3 bucket STRING prefix STRING format STRING mode STRING partitioned_by?`

#### AST
Tree of nodes representing the parsed DSL structure:

```
Program
├── SourceNode          — table name, S3 config, fields
├── ExtractNode          — json action type (select/explode/extract), path, fields
├── JoinNode             — left/right tables, condition, type (inner/left), fields
├── SelectNode           — list of selected fields with types and pipelines
├── PipelineNode         — ordered list of transform operations
├── FieldNode            — source name, alias, type, pipeline
├── OutputNode           — source expression, select list, S3 destination
└── S3Config             — bucket, region, prefix, format
```

Types supported: `string`, `bigint`, `decimal(p,s)`, `date(format)`, `year`.

#### Semantic Analyzer
Walks the AST to verify correctness:
- **Type consistency**: validates that pipeline operator inputs/outputs are compatible
- **Reference resolution**: resolves table names and field references across source definitions
- **Join validation**: checks that join condition fields exist in both tables
- **Duplicate detection**: flags duplicate field aliases in select clauses
- **Source existence**: ensures all referenced tables (including chained sources like `perAcquisition.taxes`) are defined

Outputs a `SemanticContext` containing resolved symbol tables and type information.

#### Pipeline Processor
Handles the transformation chain applied to fields via `|` operators. Each pipeline stage is a function that transforms a Spark Column:

```
field | replace("." , ",") | prefix("R$") | hash() | coalesce("R$0,00")
```

Translates to PySpark:
```python
F.coalesce(
    F.sha2(
        F.concat(F.lit("R$"), F.regexp_replace(F.col("field"), "\\.", ",")),
        256
    ),
    F.lit("R$0,00")
)
```

The processor builds a composition of Spark Column functions, emitting the chain as nested function calls.

#### Code Generator
Walks the cleaned AST and emits PySpark code in phases:
1. **Source loading**: `spark.read.format("delta").load("s3a://bucket/prefix")`
2. **Extraction**: `df.select(F.col("message.*"))`, `df.withColumn("perAcquisition", F.explode(F.col("message.perAcquisition")))`
3. **Field selection + pipelines**: `df.select(alias, pipeline_expr.alias("output_name"))`
4. **Joins**: `df1.join(df2, condition, "inner")`, `df1.join(df2, condition, "left")`
5. **Output**: `df.write().format("delta").mode("overwrite").partitionBy("year", "month", "day").save("s3a://bucket/prefix")`

---

## 3. Pipeline Operators

| Operator | Syntax | Description | PySpark Mapping |
|---|---|---|---|
| `replace` | `replace(old, new)` | String replacement | `F.regexp_replace(col, old, new)` |
| `prefix` | `prefix(str)` | Prepend string literal | `F.concat(F.lit(str), col)` |
| `suffix` | `suffix(str)` | Append string literal | `F.concat(col, F.lit(str))` |
| `hash` | `hash()` | SHA-256 hash | `F.sha2(col, 256)` |
| `coalesce` | `coalesce(val)` | Null fallback | `F.coalesce(col, F.lit(val))` |
| `default` | `default(val)` | Null default (alias for coalesce) | `F.coalesce(col, F.lit(val))` |
| `cast` | `cast(type)` | Type cast | `col.cast(type)` |
| `split` | `split(delim, idx)` | Split string, pick element | `F.split(col, delim).getItem(idx)` |
| `year` | `year()` | Extract year from date | `F.year(col)` |
| `month` | `month()` | Extract month from date | `F.month(col)` |
| `day` | `day()` | Extract day from date | `F.day(col)` |

Pipelines chain left-to-right. Each operator output feeds into the next:
```
financeiro | replace("." , ",") | prefix("R$") | hash() | coalesce("R$0,00")
```

---

## 4. Join Types

### Inner Join
```
trade /\ perAcquisition -> trade.trade_id = perAcquisition.trade_id
```
Maps to: `df_trade.join(df_per_acquisition, col("trade.trade_id") == col("perAcquisition.trade_id"), "inner")`

### Left Join
```
trade /\ perAcquisition -> trade.trade_id = perAcquisition.trade_id, left
```
Maps to: `df_trade.join(df_per_acquisition, col("trade.trade_id") == col("perAcquisition.trade_id"), "left")`

### Syntax breakdown:
- `table1 /\ table2` — join operator (inspired by relational algebra bowtie)
- `-> condition` — join condition using `table.field = table.field`
- `, left` — optional modifier; defaults to inner join when omitted

---

## 5. Output Modes

### Modes

| Mode | PySpark | Behavior |
|---|---|---|
| `overwrite` | `.mode("overwrite")` | Replace existing data |
| `append` | `.mode("append")` | Add rows to existing data |
| `merge` | `.option("mergeMode", "true")` | Upsert / merge (Delta Lake) |

### Partitioning
```
partitioned by year, month, day
```
Maps to: `.partitionBy("year", "month", "day")`

Partition columns must be present in the output field list or resolvable from the source (e.g. via `year()` pipeline operator on a date field).

### Full output block example (generated PySpark):
```python
df_position_exploded = df_position_exploded \
    .select(
        F.col("name").alias("name"),
        F.coalesce(
            F.sha2(
                F.concat(F.lit("R$"),
                    F.regexp_replace(F.col("financeiro"), "\\.", ",")),
                256
            ),
            F.lit("R$0,00")
        ).alias("financeiro"),
        F.year(F.col("positionDate")).alias("year")
    )

df_position_exploded.write() \
    .format("delta") \
    .mode("overwrite") \
    .partitionBy("year", "month", "day") \
    .save("s3a://prd_tables/position_exploded")
```

---

## 6. Implementation Plan

### Phase 1: Lexer + Basic Types
- Define `TokenType` constants and `Token` struct
- Implement `Lex(filepath string) ([]Token, error)`
- Support keywords, identifiers, string/number literals, operators
- Add line/column tracking for error reporting
- Write tests with sample DSL snippets

### Phase 2: Parser + AST
- Define AST node types (`SourceNode`, `ExtractNode`, `SelectNode`, `FieldNode`, `JoinNode`, `OutputNode`, `PipelineNode`, `S3Config`)
- Implement recursive descent parser (`Parse(tokens []Token) (*Program, error)`)
- Handle `from` → `extract` → field definitions → pipelines
- Handle join syntax (`/\`, `->`, modifiers)
- Handle output definitions with `to s3` blocks
- Write parser tests covering all DSL constructs

### Phase 3: Semantic Analysis + Reference Resolution
- Build symbol table from all source definitions
- Resolve field references in joins (e.g. `trade.trade_id = perAcquisition.trade_id`)
- Validate type consistency for pipeline operator arguments
- Check for duplicate definitions and dangling references
- Implement `Analyze(ast *Program) (*SemanticContext, error)`

### Phase 4: Code Generation (PySpark)
- Implement `Generate(context *SemanticContext) (string, error)` — walks AST and emits PySpark
- Source loading: `spark.read.format(...).load(...)`
- JSON extraction: `df.select()`, `df.withColumn()` with `F.explode()`
- Field selection with aliases
- Join emission with correct type and condition
- Output block: `df.write().format().mode().partitionBy().save()`
- Write golden-file tests comparing generated code against expected PySpark

### Phase 5: Pipeline Operators
- Implement operator registry mapping operator names to Spark column transforms
- Support chain composition (nested function calls)
- Operators: `replace`, `prefix`, `suffix`, `hash`, `coalesce`, `default`, `cast`, `split`, `year`, `month`, `day`
- Add operator argument parsing (string literals, numeric parameters)
- Test each operator individually and in chains

### Phase 6: Integration + Testing
- End-to-end tests: DSL source → generated PySpark → run against Spark (optional, local mode)
- Error handling with descriptive messages (line:column format)
- CLI entrypoint: `sieve transpile <input.dsl> [-o output.py]`
- Support for `...` ellipsis shorthand in sources
- Edge cases: empty selects, missing from blocks, circular joins
- CI pipeline with lint, test, and golden-file checks