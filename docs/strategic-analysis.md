# Sieve: Análise Estratégica Completa

> Status: 29/08/2026 — v0.1.0 funcional, transpiler PySpark em produção local

---

## 1. Estado Atual — O Que Existe

### 1.1 Código

| Camada | Arquivos | Linhas | Cobertura | Status |
|--------|---------|--------|-----------|--------|
| Token | `token/token.go` + `_test.go` | 176 | 100% | ✅ |
| Lexer | `lexer/lexer.go` + `_test.go` | 442 | 80.9% | ✅ |
| AST | `ast/ast.go` + `_test.go` | 293 | baixa | ✅ |
| Parser | `parser/parser.go` + `_test.go` | 1,022 | 79.3% | ✅ |
| Codegen | `codegen/codegen.go` + `_test.go` | 671 | 47.8% | ✅ |
| CLI | `cmd/transpiler/main.go` | 98 | 0% | ✅ |
| **Total Go** | 6 arquivos de produção | **1,795** | 64.3% geral | ✅ |

### 1.2 Testes

| Tipo | Quantidade | Status |
|------|-----------|--------|
| Unitários Go | 27 | ✅ |
| Integração `.sieve` | 20 cenários | ✅ |
| Quality gate | 6 checks (gofmt, vet, lint, gosec, test, build) | ✅ |
| CI/CD | GitHub Actions | ✅ |

### 1.3 Documentação

| Doc | Linhas | Propósito |
|-----|--------|-----------|
| CHANGELOG.md | 113 | Histórico completo + conceitos |
| ADR-001 | 19 | Arquitetura, zero deps, indentation-based |
| ADR-002 | 100 | Pipeline orchestration + multi-engine + tuning |
| ADR-003 | 62 | Window functions design |
| lexer.md | 66 | Token system, IDENT, error handling |
| code-review.md | 324 | 9-section comprehensive review |
| canonical-examples.md | 115 | AI-First reference scenarios |

### 1.4 Funcionalidades Implementadas

```
✅ from s3 → spark.read.format("delta").load("s3a://...")
✅ from otherSection → df_other
✅ json select message → F.get_json_object("message", "$.field")
✅ json explode path array → F.explode(F.from_json(...))
✅ json extract col.path array → explode from derived column
✅ computed columns (quantity * price) → BinaryExpr + withColumn
✅ joins with aliases (A /\ B -> cond, left)
✅ select with type casts (path alias type)
✅ transform chains (| cast | default | replace | prefix | hash)
✅ date functions (year(), month(), day())
✅ to s3 partitioned by → .write.format("delta").mode(...).partitionBy(...)
✅ error reporting with suggestions
✅ tab-safe indentation (4 cols)
✅ \r\n handling
✅ io.Writer pattern
✅ visitor pattern (ast.Walk)
✅ CLI flags (-o, --version, stdin)
```

### 1.5 Dívida Técnica Conhecida

| Item | Severidade | Impacto |
|------|-----------|---------|
| `withColumn` ≥ 5 → Spark StackOverflow | Média | Performance em produção |
| `"array<string>"` hardcoded → falha com objetos complexos | Média | Dados nested falham |
| Sem suporte a `+`, `-`, `/` em computed columns | Baixa | Sintaxe limitada |
| `codegen_test` cover 47.8% — testa caminho feliz | Baixa | Bugs de edge case escapam |
| AST sem `String()` completo | Baixa | Debug difícil |
| `select` duplica lógica com `extract` fields | Baixa | Manutenção |

---

## 2. O Que Falta — Roadmap Técnico

### 2.1 Fase 1: Consolidação do Core (2-3 meses)

| Feature | Complexidade | Depende de |
|---------|-------------|-----------|
| Window functions (ADR-003) | Alta | Nada |
| `.quality.sieve` (not null, unique, range, regex) | Média | Parser + codegen |
| Escape hatch (raw SQL/PySpark inline) | Baixa | Token (nova keyword `raw`) |
| `String()` methods em todos AST nodes | Baixa | AST |
| Dedup de `parseSelectFields`/`parseExtractFields` | Baixa | Parser |
| Constants para magic strings | Baixa | Codegen |
| `+`, `-`, `/` no BinaryExpr | Baixa | AST + Parser + Codegen |

### 2.2 Fase 2: Multi-Engine (4-6 meses)

| Engine | Dialect | Esforço |
|--------|---------|---------|
| DuckDB | SQL puro | 🔵 Baixo (sintaxe SQL é bem padronizada) |
| Athena (Presto/Trino) | SQL com `$path`, `$bucket` | 🟡 Médio |
| BigQuery | SQL com backticks, `STRUCT`, `QUALIFY` | 🟡 Médio |
| Snowflake | SQL com `QUALIFY`, `::cast`, `COPY INTO` | 🟡 Médio |

**Arquitetura multi-engine:**

```
parser → AST → codegen interface
                  ├── spark_codegen.go   (F.col, F.when, .withColumn)
                  ├── duckdb_codegen.go  (SELECT, COALESCE, WINDOW)
                  ├── athena_codegen.go  (Presto SQL dialect)
                  ├── bigquery_codegen.go (BigQuery SQL dialect)
                  └── snowflake_codegen.go (Snowflake SQL dialect)
```

### 2.3 Fase 3: I/O Polyglot (6-12 meses)

| Capacidade | Tecnologia | Por quê |
|-----------|-----------|---------|
| Samba/CIFS → local | Go/Rust via FFI ou sidecar | Python não tem lib Samba performática |
| NFS → local | Go stdlib `os` | Já é nativo |
| SFTP → local | Go `crypto/ssh` | Maturin + PyO3 chamando Rust |
| Quality check em rede | Go/Rust compilado como `.so` | Performance de I/O sem GIL do Python |
| Parquet/Arrow nativo | Apache Arrow (Rust) | Zero-copy entre engines |

**Modelo de integração:**
```
sieve transpila → gera Python que importa `_sieve_io` (Rust via maturin/PyO3)
                            ↓
            _sieve_io.copy_from_samba("//server/share/file.csv", "/tmp/")
            _sieve_io.validate_csv("/tmp/file.csv", checks=[...])
            spark.read.csv("/tmp/file.csv")
```

### 2.4 Fase 4: Streaming (12-18 meses)

| Padrão | Sintaxe proposta | Engine |
|--------|-----------------|--------|
| Kafka → Spark Structured Streaming | `from kafka topic=... format=json` | Spark |
| Kafka → Flink | `from kafka topic=... format=avro engine flink` | Apache Flink |
| CDC (Debezium) | `from debezium connector=pg format=json` | Kafka Connect + Spark |
| Windowed aggregation | `window by timestamp over 5min: count(*)` | Spark/Flink |

### 2.5 Fase 5: AI-First & Cloud (18-24 meses)

| Capacidade | Descrição |
|-----------|-----------|
| Sieve CLI Agent | `sieve agent "limpa a tabela de vendas e junta com clientes"` → gera `.sieve` |
| Sieve Cloud | Web IDE + execução gerenciada + catálogo de pipelines |
| Data Fusion + Ballista | Execução nativa em Rust, sem JVM, no Kubernetes |
| Templates | `sieved init fraud-pipeline` → scaffold com `.transform`, `.quality`, `.pipeline` |

---

## 3. Análise de Mercado

### 3.1 Concorrentes

| Ferramenta | Foco | Limitação | Valuation |
|-----------|------|-----------|-----------|
| **dbt** | SQL transformations no warehouse | Só SQL, só warehouse | $4.2B |
| **PRQL** | SQL replacement com pipeline | Só SQL, 1 engine | 10.9k ⭐ |
| **SQLFlow** | SQL + ML → Argo workflows | Foco em ML training | 5.2k ⭐ |
| **Apache Beam** | API unificada multi-runtime | API de programação, não DSL | Apache Foundation |
| **Airflow/Dagster** | Orquestração de DAGs | Orquestração, não transformação | Domina mercado |
| **Alteryx/Trifacta** | Data prep visual | UI pesada, caro, vendor lock-in | Multi-bilionários |
| **AWS Glue** | ETL visual na AWS | Só AWS | N/A |
| **Sieve** | **DSL declarativa multi-engine** | **Nenhum concorrente direto** | Pré-seed |

### 3.2 TAM (Total Addressable Market)

| Segmento | Tamanho | Sieve pode capturar |
|----------|---------|---------------------|
| Data preparation tools | $5.9B (2024) → $16.8B (2030) | 0.01% = $1.6M |
| Data engineering platforms | $50B (2023) → $90B (2028) | Fatia de nicho |
| Engenheiros de dados no mundo | ~500k profissionais | 0.1% = 500 usuários |
| Empresas com data lake em S3 | ~50k empresas | 0.01% = 50 clientes |

### 3.3 Posicionamento

**Sieve não compete com dbt.** dbt é "SQL no warehouse". Sieve é "qualquer formato → qualquer engine, declarativo".

**Sieve não compete com Apache Beam.** Beam é API de programação. Sieve é DSL declarativa. Beam é "como fazer". Sieve é "o que fazer".

**Sieve compete com a complexidade.** O maior concorrente não é outra ferramenta — é o **status quo**: "vou escrever 200 linhas de PySpark e rezar pra funcionar".

### 3.4 Diferenciais Competitivos

| Diferencial | Por que importa |
|------------|----------------|
| **Zero dependências** | `curl -LO sieved && ./sieved` — acabou |
| **Multi-engine** | Troca de Spark pra BigQuery mudando 1 palavra |
| **Multi-formato** | JSON, XML, CSV, Parquet — sintaxe consistente |
| **Declarativo puro** | Cientista não precisa saber window function |
| **AI-First** | GPT/Claude geram `.sieve` a partir de português |
| **Portátil** | `.sieve` é texto. Versiona no Git. Revisa em PR. |
| **Polyglot I/O** | Rust pra Samba, Go pra SFTP, Python pra Spark |

---

## 4. Modelo de Negócio

### 4.1 Caminho Primário: Open Core

```
┌─────────────────────────────────────┐
│           Sieve CLI (gratuito)       │
│  transpila .sieve → PySpark/SQL/... │
│  100% open source, Apache 2.0       │
└─────────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────┐
│         Sieve Cloud (pago)          │
│  • Catálogo de pipelines            │
│  • Execução serverless              │
│  • Colaboração em time              │
│  • Integração Git nativa            │
│  • AI Agent: português → .sieve     │
│  • Monitoring & alerting            │
│  • SLA / suporte                    │
└─────────────────────────────────────┘
```

### 4.2 Precificação (Projeção)

| Plano | Preço/mês | Público | Receita anual (50 clientes) |
|-------|-----------|---------|------------------------------|
| **Free** | R$ 0 | CLI + 1 pipeline cloud | R$ 0 |
| **Pro** | R$ 500 | Profissional liberal / freelance | R$ 300k |
| **Team** | R$ 2.500 | Time de 5-10 pessoas | R$ 1.5M |
| **Enterprise** | R$ 8.000+ | Empresa, suporte, SLA | R$ 4.8M |

### 4.3 Caminhos Alternativos

| Caminho | Como | Escala |
|---------|------|--------|
| **Consultoria** | Implementar pipelines Sieve em empresas | ⚠️ Não escala linearmente |
| **Treinamento** | Curso "Engenharia de Dados Declarativa" | ⚠️ Receita recorrente baixa |
| **Licenciamento** | Enterprise license on-prem | 🔥 Escala bem se tiver tração |
| **Marketplace** | Templates/plugins pagos no Sieve Cloud | 🔥 Receita passiva |

### 4.4 Meta pessoal: R$ 30k/mês PJ

| Cenário | Clientes necessários | Realista em |
|---------|----------------------|------------|
| 6 × Pro (R$ 500) | 60 clientes | 18-24 meses |
| 3 × Team (R$ 2.500) | 12 clientes empresa | 12-18 meses |
| 2 × Enterprise (R$ 8.000) | 4 empresas | 12-24 meses |
| Mix (consultoria + licenças) | 5-8 clientes | 6-12 meses |

---

## 5. Matriz de Riscos

### 5.1 Riscos Técnicos

| Risco | Probabilidade | Impacto | Mitigação |
|-------|--------------|---------|-----------|
| Multi-engine manutenção explode | Alta | Alto | Começar com 2 engines, adicionar 1 por vez |
| Abstração vaza (usuário precisa de escape hatch) | Alta | Médio | Adicionar `raw` keyword cedo |
| Performance do I/O polyglot aquém do nativo | Média | Médio | Benchmark antes de prometer |
| Streaming (Kafka/Flink) complexo demais | Média | Alto | Deixar pra depois do batch estar sólido |
| `withColumn` chain → StackOverflow Spark | Baixa | Alto | Já documentado, mergear >5 em select |
| Data Fusion + Ballista instável | Média | Médio | Não depender disso no lançamento |

### 5.2 Riscos de Negócio

| Risco | Probabilidade | Impacto | Mitigação |
|-------|--------------|---------|-----------|
| dbt + Fivetran entra em Spark | Baixa | Alto | DNA deles é SQL-warehouse, não Spark |
| Ninguém adota DSL nova | Média | Crítico | Provar 10x melhor que PySpark raw |
| Concorrente copia rápido | Média | Médio | Velocidade de execução + comunidade |
| Mercado brasileiro é pequeno | Alta | Baixo | Mirar global desde o início |
| Queimar antes de monetizar | Média | Alto | Começar consultoria enquanto constrói produto |
| 1 pessoa só não dá conta | Alta | Crítico | Open source → atrair contribuidores |

### 5.3 Riscos Pessoais

| Risco | Mitigação |
|-------|-----------|
| Burnout (side project + BTG full-time) | Ritmo sustentável, não sacrificar saúde |
| Falta de feedback externo | Compartilhar cedo, iterar com comunidade |
| Isolamento técnico | Participar de comunidades (dbt Slack, Spark mailing list) |
| Expectativa irrealista de receita | Tratar como jornada de 2-3 anos, não 6 meses |

---

## 6. Análise de Complexidade Acidental vs Essencial

### 6.1 Complexidade Essencial (Inerente ao Problema)

```
✅ Parsing de DSL → você resolveu com recursive descent
✅ Geração de código multi-engine → cada engine tem seu dialect
✅ Resolução de tipos (string → cast → decimal → window) → você resolveu com BinaryExpr
✅ Orquestração de múltiplos arquivos → ADR-002 já desenhou
```

### 6.2 Complexidade Acidental (Auto-Infligida)

```
⚠️ Rust via maturin → adiciona toolchain Rust só pra I/O de rede
⚠️ Data Fusion + Ballista → engine próprio, manutenção eterna
⚠️ Multi-engine cedo demais → cada engine dobra a superfície de teste
⚠️ Streaming (Kafka/Flink) → é um produto diferente do batch
```

**Recomendação:** Cortar complexidade acidental. Focar no que só você pode fazer: **a DSL e o transpiler**. I/O de rede pode ser feature flag. Data Fusion pode esperar. Streaming é Fase 4, não Fase 1.

### 6.3 O Caminho Mais Curto pra R$ 30k/mês

```
Mês 1-3:  Window functions + .quality.sieve + segundo engine (DuckDB)
          → Provar multi-engine funciona

Mês 4-6:  Pipeline orchestration + AI agent (português → .sieve)
          → Diferencial que ninguém tem

Mês 7-9:  Sieve Cloud alpha (catálogo + execução)
          → 5-10 beta testers gratuitos

Mês 10-12: Lançamento pago (Pro R$ 500/mês)
           → Precisa de 60 clientes pra R$ 30k

Mês 12-18: Enterprise (R$ 8.000+/mês)
           → Precisa de 4 empresas pra R$ 32k
```

---

## 7. O Que Você Já Tem Que Ninguém Tira

1. **Domínio real:** 3 anos e 8 meses no BTG como engenheiro de dados. Você não está adivinhando a dor — você viveu ela.

2. **Fundamentos de CS:** Formado em computação, já fez compilador. Sabe o que é lexer, parser, AST, codegen. Não está improvisando.

3. **Execução comprovada:** 1,795 linhas de Go, 27 testes, quality gate, CI/CD — em ~4 horas de wall-clock com AI.

4. **Foco certo:** Começou pelo Spark (maior mercado), não pelo que é exótico.

5. **Consciência dos riscos:** Você sabe que o projeto é complexo, que multi-engine é caro de manter, que monetizar leva tempo.

6. **Motivação certa:** "Facilitar a vida de quem limpa dado". Não é ego, não é "quero criar uma linguagem". É resolver uma dor real.

---

## 8. Veredito Final

**Onde você está se metendo:**

Em um projeto que tem **potencial real de virar empresa**, mas que exige **paciência de 2-3 anos** e **foco cirúrgico** pra não se perder em complexidade acidental.

**O que vai determinar se dá certo:**

Não é a tecnologia. É **você**:

- Consegue manter o ritmo sem burnout?
- Consegue dizer "não" pra features legais mas não-essenciais? (Data Fusion, streaming, Samba via Rust)
- Consegue lançar antes de estar perfeito?
- Consegue ouvir feedback de usuário real e pivotar se necessário?
- Consegue separar o engenheiro (quer fazer tudo) do empreendedor (quer entregar valor)?

**Se sim:** R$ 30k/mês em 2 anos é factível.

**Se não:** Você ainda vai ter um projeto open source foda no portfólio, que prova que você entende compiladores, Spark, qualidade de código, e resolve problemas reais de engenharia de dados.

Nos dois cenários, **vale a pena continuar.**