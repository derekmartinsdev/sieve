# ADR-003: Window Functions and Temporal Computations

## Status
Proposed

## Context
Data engineering pipelines frequently need window functions for:
- Rolling aggregations (moving averages, cumulative sums)
- Deduplication (row_number + filter)
- Temporal comparisons (lag, lead)
- Sessionization and gap detection

The current DSL has no support for these patterns.

## Decision
Add a `window` keyword that defines window specifications with partition, ordering, and range/frame.

## Proposed Syntax

### 1. Rolling window (temporal range)
```
transform transacoes_rolling:
    from transacoes_silver
    | window by cliente_id order by timestamp:
        avg(valor) over 24h as media_valor_24h
        count(*) over 24h as qtd_transacoes_24h
        avg(valor) over 30d as media_valor_30d
        sum(valor) over 30d as total_gasto_30d
```

### 2. Deduplication (row_number)
```
transform contas_atuais:
    from contas_cdc_raw
    | window by conta_id order by updated_at desc:
        row_number() as rnk
    | filter rnk == 1
    | select conta_id, titular, saldo, updated_at
```

### 3. Temporal comparison (lag/lead)
```
transform delta_transacoes:
    from transacoes_silver
    | window by cliente_id order by timestamp asc:
        lag(valor, 1, default=0.0) as valor_anterior
        lag(timestamp, 1) as ts_anterior
    | compute:
        diff_valor = valor - valor_anterior
        segundos_desde_ultima = timestamp - ts_anterior
```

## AST Changes
- New `WindowSpec` node: PartitionBy, OrderBy, Frame, RangeDuration
- New `WindowTransform` node: TargetCol, Function, SourceCol, Window
- New `WindowNode` container for window definitions

## Consequences
- Enables feature engineering, deduplication, and temporal analysis
- Requires refactoring TransformChain to support window+filter+compute
- Window functions generate Window.partitionBy().orderBy() in PySpark
- `rangeBetween` requires timestamp columns cast to long
- `over 24h` syntax maps to `rangeBetween(-86400, 0)` in seconds