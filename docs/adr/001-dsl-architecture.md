# ADR-001: Sieve DSL Architecture

| Status       | Proposed |
| ------------ | -------- |
| Date         | 2026-09-06 |
| Deciders     | @derekmartinsdev |
| Supersedes   | — |

---

## Context

The Sieve DSL defines declarative ETL pipelines: read from S3 (JSON, CSV, Delta), extract fields, transform values, join sources, and write output to Delta/Iceberg tables. The initial grammar was JSON-first — extraction assumed the entire source is JSON. Real-world pipelines mix plain columns and JSON-string columns in the same source (e.g., a CSV where one field contains serialized JSON).

We need to decide:

1. How extraction from plain and nested fields coexists
2. How column ordering is controlled during extraction
3. How the `transform.sieve` and `pipeline.sieve` file convention maps to data architecture patterns (Medallion, Data Vault)
4. How materialized vs. ephemeral (view) outputs are expressed
5. How non-Delta output formats (Iceberg, Parquet) fit in
6. Whether `to s3` is allowed in `transform.sieve` files

---

## Decision Outcomes

### 1. Extraction format is declared per block — execution order defines column order

Each `extract` block declares its format (`json`, `plain`) and operation (`select`, `explode`). Blocks are processed sequentially; columns accumulate in declaration order. This solves the core problem: a Delta/CSV source with both plain columns and JSON-string columns.

```transform.sieve
position
    from s3
        bucket prd_tables
        region us-east-1
        prefix position
        format delta

    extract
        plain
            id             id         bigint
            trade_id       trade_id   bigint

    extract
        json select message
            client.name         name           string
            client.id           client_id      bigint
            client.document     document       string
            positionId          position_id    bigint
            positionDate        position_date  date

    extract
        json explode message.perAcquisition array
            reference         cod_fatura_origem  string
            buyDate           data_compra        date
            quantity * price  financeiro         decimal(18,2)
```

**Rationale:** The data engineer thinks in steps ("extract plain fields, then parse the JSON column, then explode the array"). Multi-block extracts mirror this mental model without forcing CTE boilerplate.

### 2. Parsing (transform) and joining (pipeline) are separate files

| File                | Responsibility                                      |
| ------------------- | --------------------------------------------------- |
| `*.transform.sieve` | Extract, type, parse JSON from a single raw source  |
| `*.pipeline.sieve`  | Join sources, transform values, reorder columns, write output |

**Rationale:** Reading raw data and composing business logic are separate concerns. This separation maps cleanly to both Medallion and Data Vault patterns without imposing either.

#### Medallion alignment

| Layer    | File                  | Behavior                        |
| -------- | --------------------- | ------------------------------- |
| Bronze   | (raw S3/Delta, no .sieve) | Immutable, as-is            |
| Silver   | `*.transform.sieve`   | Extract, type, parse JSON       |
| Gold     | `*.pipeline.sieve`    | Join, aggregate, business logic |

#### Data Vault alignment

| Piece      | File                  | Behavior                            |
| ---------- | --------------------- | ----------------------------------- |
| Hub        | `*.transform.sieve`   | Extract business keys only          |
| Satellite  | `*.transform.sieve`   | Extract descriptive attributes      |
| Link       | `*.pipeline.sieve`    | Join hubs to create relationships   |

The user organizes files into directories that reflect their chosen pattern (`pipelines/silver/`, `pipelines/hubs/`, etc.). Sieve does not enforce a specific model.

### 3. `to s3` is optional on any source — presence determines materialization

A source with `to s3` is materialized to storage. Without it, the source is an ephemeral view (CTE) consumed downstream in the same pipeline.

```transform.sieve
-- ephemeral view: no 'to s3', exists only in the pipeline execution
client_clean
    from s3
        bucket prd_tables
        prefix client
        format delta
    extract
        json select message
            client.id   id     bigint
            client.name name   string

-- materialized table: has 'to s3', persisted to storage
client_clean
    from s3 ...
    extract ...
    to s3
        bucket silver_tables
        prefix client_clean
        format delta
        mode overwrite
```

**Rationale:** This avoids forcing the user to create a separate `.pipeline.sieve` just to materialize a silver table. The `to s3` block is the single, explicit signal.

### 4. Output format is a parameter on `to s3`

```
to s3
    bucket prd_tables
    region us-east-1
    prefix final_table
    format delta       -- or iceberg, parquet, json, csv
    mode overwrite
    partitioned by year, month, day
```

**Rationale:** Adding a new output format requires only a new keyword token, no grammar changes. The `to s3` block remains the single write interface.

### 5. Column reordering happens in `pipeline.sieve` via `select`

The `transform.sieve` produces columns in extraction order. The `pipeline.sieve` `select` block is the single point for reordering columns before the final write.

```pipeline.sieve
position_exploded = position (p) /\ enriched (e) -> p.trade_id = e.trade_id, left

    select
        p.name
        p.asset_name
        p.position_id
        e.trade_id
        e.reference
        e.financeiro

    transform
        financeiro = e.financeiro
        | cast decimal(18,8)
        | replace(".", ",")
        | prefix("R$")

    to s3
        bucket prd_tables
        prefix position_exploded
        format delta
        mode overwrite
```

**Rationale:** A single reorder point is easier to reason about than mixing column ordering with extraction logic.

---

## Alternatives Considered

| Alternative | Rejected because |
|---|---|
| Single `EXTRACT JSON SELECT` block mixing plain and JSON fields | No way to declare field origin format; ambiguous for CSV sources |
| `from source_name` (CTE) required for every explode | Too verbose; forces naming simple intermediate steps |
| `format` and `select` on the same line (`json select message`) | Harder to extend — new formats would break the one-liner contract |
| Forcing `.pipeline.sieve` for materialization of silver tables | Produces boilerplate wrappers with a single line |
| Folding `pipeline.sieve` into `transform.sieve` | Mixes extraction concern (single source) with composition concern (multi-source) |

---

## Consequences

- **Parser must be extended:** `plain` keyword, multi-block extract with format dispatch, `to s3` allowed in `transform.sieve`
- **AST gains structure:** `ExtractBlock.Format` field, output format extensibility
- **User migration path:** Existing `json select` blocks are backward compatible — only the new `plain` block adds capability
- **Documentation:** README and examples must surface the Medallion/DataVault mapping as a recommendation, not a requirement

---

## Future Considerations

- **Version pinning in `to s3`:** `mode versioned` for Delta time travel
- **Quality gates:** `expect` block for data contracts before write
- **Incremental partitions:** `mode merge` with partition pruning hints
- **Spark codegen:** Compile `.sieve` files to PySpark/Scala jobs