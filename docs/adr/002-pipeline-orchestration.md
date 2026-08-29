# ADR-002: Pipeline orchestration with multi-engine codegen

## Status
Proposed

## Context
Currently the transpiler only generates PySpark code. The future vision includes multiple output targets (Airflow DAGs, DuckDB SQL, etc.) and a pipeline orchestrator that ties together multiple `.sieve` files.

The `.pipeline.sieve` file must be able to reference other `.sieve` files, define data sources and sinks, and select the execution engine.

## Proposed Syntax

```
engine spark | airflow | duckdb

position:transform -> df_position
trade:transform -> df_trade
quality:check -> qc_trade

df_position /\ df_trade -> df_position.trade_id = df_trade.trade_id
    select ...

target s3 bucket=... format=delta mode=overwrite
```

### File references (`prefix:type -> alias`)

```
position:transform -> df_position
```

- `position` — file prefix, resolves to `position.transform.sieve`
- `transform` — file type (maps to extension: `.transform.sieve`, `.quality.sieve`, `.schedule.sieve`)
- `-> df_position` — exposes the result (DataFrame in Spark, task in Airflow, CTE in DuckDB) with the given alias
- If `:type` is omitted, defaults to `:transform`

### Engine selection

```
engine spark
```

Selects which codegen backend to use. Future engines:
- `spark` — PySpark `.py` (current)
- `airflow` — Airflow DAG `.py`
- `duckdb` — DuckDB SQL `.sql`

### File role separation

| Extension | Purpose | Has source? | Has sink? |
|-----------|---------|-------------|-----------|
| `.transform.sieve` | Pure transformations | No | No |
| `.quality.sieve` | Data quality rules | No | No |
| `.schedule.sieve` | Scheduling/DAG definitions | No | No |
| `.pipeline.sieve` | Orchestrator | Yes (via `+` refs) | Yes |

## Decision
- `prefix:type -> alias` syntax for cross-file references
- `engine` keyword for multi-engine selection
- Pipeline orchestrator is the `.pipeline.sieve` file
- Transform/quality/schedule files are pure — no source/sink declarations

## Consequences
- Clear separation of concerns: business logic vs orchestration
- Multi-engine support without changing transform/quality files
- Pipeline file becomes the single source of truth for deployment
- Parser must support `@import`-style cross-file resolution
- Codegen must be pluggable per engine (strategy pattern or interface)

## Future: Automatic Tuning Suggestions

Based on static analysis of the pipeline, the transpiler will generate a tuning report:

### Detection rules

| Pattern detected | Suggestion |
|-----------------|------------|
| ≥ 3 joins in a single section | Increase `spark.sql.shuffle.partitions` |
| Window functions (`row_number`, `rank`, `lag`) | Switch to storage memory: `spark.memory.storageFraction` |
| Large broadcast joins (small table + large table) | Adjust `spark.sql.autoBroadcastJoinThreshold` |
| No `partitionBy` on sink | Warn: "Consider partitioning your output to avoid full scans" |
| Many `withColumn` calls (≥ 5) | Suggest `select()` with all columns at once to avoid DAG bloat |

### Output format

The tuning report is emitted as a comment block at the top of the generated PySpark:

```python
# === Tuning Suggestions ===
# ⚠️ 3 joins detected in 'enriched'. Consider:
#    spark.conf.set("spark.sql.shuffle.partitions", 200)
# ⚠️ 'enriched' has 7 withColumn calls. Consider merging into a single select().
# === End Tuning Suggestions ===
```

### When to trigger

The tuning report is **always generated** but can be silenced with:
```
engine spark notune
```