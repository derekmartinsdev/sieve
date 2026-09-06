# Canonical DSL Examples (AI-First Reference)

These examples serve as the ground truth for:
1. Snapshot tests (input → expected output)
2. AI agent few-shot context
3. Grammar validation

## Example 1: Rolling Window (Feature Engineering)

### Input (.transform.sieve)
```
transform transacoes_rolling:
    from transacoes_silver
    | window by cliente_id order by timestamp:
        avg(valor) over 24h as media_valor_24h
        count(*) over 24h as qtd_transacoes_24h
        avg(valor) over 30d as media_valor_30d
        sum(valor) over 30d as total_gasto_30d
```

### Expected Output (PySpark)
```python
w_24h = Window.partitionBy("cliente_id").orderBy(F.col("timestamp").cast("long")).rangeBetween(-86400, 0)
w_30d = Window.partitionBy("cliente_id").orderBy(F.col("timestamp").cast("long")).rangeBetween(-2592000, 0)

df_transacoes_rolling = df_transacoes_silver \
    .withColumn("media_valor_24h", F.avg("valor").over(w_24h)) \
    .withColumn("qtd_transacoes_24h", F.count("*").over(w_24h)) \
    .withColumn("media_valor_30d", F.avg("valor").over(w_30d)) \
    .withColumn("total_gasto_30d", F.sum("valor").over(w_30d))
```

## Example 2: Deduplication (CDC)

### Input (.transform.sieve)
```
transform contas_atuais:
    from contas_cdc_raw
    | window by conta_id order by updated_at desc:
        row_number() as rnk
    | filter rnk == 1
    | select conta_id, titular, saldo, updated_at
```

### Expected Output (PySpark)
```python
w_dedup = Window.partitionBy("conta_id").orderBy(F.col("updated_at").desc())

df_contas_atuais = df_contas_cdc_raw \
    .withColumn("rnk", F.row_number().over(w_dedup)) \
    .filter(F.col("rnk") == 1) \
    .select("conta_id", "titular", "saldo", "updated_at")
```

## Example 3: Temporal Comparison (Lag/Lead)

### Input (.transform.sieve)
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

### Expected Output (PySpark)
```python
w_seq = Window.partitionBy("cliente_id").orderBy(F.col("timestamp").asc())

df_delta_transacoes = df_transacoes_silver \
    .withColumn("valor_anterior", F.lag("valor", 1, 0.0).over(w_seq)) \
    .withColumn("ts_anterior", F.lag("timestamp", 1).over(w_seq)) \
    .withColumn("diff_valor", F.col("valor") - F.col("valor_anterior")) \
    .withColumn("segundos_desde_ultima", F.col("timestamp").cast("long") - F.col("ts_anterior").cast("long"))
```

## Example 4: Pipeline Orchestration

### Input (.pipeline.sieve)
```
pipeline feature_store_fraude:
    engine spark

    source transacoes_raw:
        from delta "s3://datalake-silver/transacoes"

    import:
        transacoes_rolling:transform -> features_transacoes
        contas_atuais:transform      -> snapshot_contas
        qualidade_features:quality   -> checks_qualidade

    features_completas = features_transacoes (t) /\ snapshot_contas (c) -> t.cliente_id == c.conta_id
        | select:
            t.cliente_id,
            t.media_valor_24h,
            t.media_valor_30d,
            c.saldo,
            c.updated_at

    sink features_completas:
        to delta "s3://datalake-gold/feature_store_fraude"
        mode overwrite
        partitioned by date(updated_at, "yyyy-MM")
```

## Usage for AI Agents

When instructing an AI to implement new features:
1. Supply the input DSL as the system prompt
2. Supply the expected output as the ground truth
3. The AI must match the exact PySpark code generation pattern

These examples form the canonical test suite for `parser_test.go` and `codegen_test.go`.