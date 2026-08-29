# Concurrent Development Workflow — Guia Completo

## O Problema

Você quer que múltiplos agentes de IA (opencode, Claude Code, Cursor, etc.) trabalhem em paralelo no mesmo repositório. O problema é: se todos compartilham o mesmo diretório de trabalho, um `git checkout` ou `git add` de um agente corrompe o estado do outro. Isso gera conflitos de merge, arquivos sumindo, commits misturados.

## A Solução: `git worktree`

`git worktree` permite que múltiplos branches tenham **diretórios independentes** compartilhando o **mesmo `.git/objects`** (zero duplicação de dados).

```
repo/
├── .git/                     ← compartilhado (objetos, refs)
├── .workspace/
│   ├── feat/lexer/           ← working tree 1 (branch feat/lexer)
│   ├── feat/parser/          ← working tree 2 (branch feat/parser)
│   ├── feat/codegen/         ← working tree 3 (branch feat/codegen)
│   └── feat/integration/     ← working tree 4 (branch feat/integration)
└── src/                      ← working tree 5 (branch main)
```

Cada worktree:
- Tem seu **próprio diretório** — arquivos completamente isolados
- Está num **branch diferente** — sem checkout collisions
- Compartilha `.git/objects` — commits de um aparecem em todos
- Pode fazer `git push`, `git pull`, `git commit` simultaneamente

---

## Passo a Passo Completo

### Fase 0: Setup do Projeto

```bash
git clone git@github.com:seu-org/seu-repo.git
cd seu-repo
git worktree prune
mkdir -p .workspace docs/adr
```

### Fase 1: Planejamento — ADRs + Contratos + AGENTS.md

**1.1 Crie os ADRs no branch main primeiro**

ADRs (Architecture Decision Records) documentam as decisões de design. Cada task independente ganha um ADR. Crie todos os ADRs sequencialmente em main e faça commit:

```bash
git checkout main
```

Exemplo de `docs/adr/001-lexer-implementation.md`:
```markdown
# ADR-001: Implementação do Lexer

## Status
Proposed

## Context
Precisamos tokenizar a DSL para processamento pelo parser.

## Decision
Implementar um lexer manual (recursive descent) que produz tokens
com Tipo, Literal, Linha e Coluna. Suporta keywords, identificadores,
strings, números, e operadores (|, /\, ->, :, =).

## Consequences
- Lexer simples sem dependências externas
- Fácil de debugar e estender
- Consistente com a abordagem do "Writing a Compiler in Go"
```

Crie um ADR para cada task independente: `001-lexer.md`, `002-ast.md`, `003-parser.md`, etc. Commit tudo em main de uma vez.

**1.2 Crie os contratos/interfaces em main**

Se as tasks compartilham tipos (ex: token types usados pelo parser), crie as interfaces vazias em main:

```bash
mkdir -p token
cat > token/token.go << 'EOF'
package token

type TokenType string

type Token struct {
    Type    TokenType
    Literal string
    Line    int
    Column  int
}

const (
    ILLEGAL = "ILLEGAL"
    EOF     = "EOF"
    IDENT   = "IDENT"
    // ... outros tipos
)

var Keywords = map[string]TokenType{}
func LookupIdent(ident string) TokenType { return IDENT }
EOF

git add docs/adr/ token/
git commit -m "docs: add ADRs for all components + token interfaces"
git push origin main
```

**Isso elimina conflitos no merge porque o contrato já está em main.**

**1.3 Crie/atualize o AGENTS.md**

O `AGENTS.md` (ou `.opencode/rules.md`) é lido automaticamente pelo opencode em toda sessão. Commit ele em main. Exemplo completo:

```markdown
# Sieve Transpiler — Regras para Agentes

## Workflow

### ADR-Driven Development
- Toda feature começa com um ADR em docs/adr/NNN-description.md
- Formato: Status, Context, Decision, Consequences
- O ADR deve ser criado em main ANTES de iniciar a implementação

### Branches
- Cada feature tem seu próprio branch: feat/<nome>
- Branches são isolados via git worktree em .workspace/feat/<nome>
- NUNCA compartilhe o mesmo checkout entre agentes

### Commits
- Commits devem ser atômicos por arquivo
- Mensagem no formato: tipo: mensagem (ex: "feat: add lexer", "fix: handle hyphen in identifiers")
- Tipos: feat, fix, docs, refactor, test, chore

### Código
- Linguagem: Go
- Package por diretório (token/, lexer/, ast/, parser/, etc)
- Testes obrigatórios para pipeline operators e codegen
- go vet deve passar antes do push
- Erros devem ser retornados, não panic

## Comandos úteis
- go mod init github.com/derekmartinsdev/sieve
- go build ./...
- go test ./... -v
- go vet ./...

## Estrutura esperada
- token/token.go — tipos de token
- lexer/lexer.go — lexer
- ast/ast.go — AST nodes
- parser/parser.go — parser
- semantic/semantic.go — análise semântica
- pipeline/pipeline.go — operadores de pipeline
- codegen/codegen.go — geração de código PySpark
- cmd/sieve/main.go — entry point
```

Commit o AGENTS.md em main. Agora toda sessão do opencode começa sabendo as regras.

### Fase 2: Criar Worktrees (3 segundos)

```bash
git worktree add -b feat/lexer    .workspace/feat/lexer    main
git worktree add -b feat/ast      .workspace/feat/ast      main
git worktree add -b feat/parser   .workspace/feat/parser   main
git worktree add -b feat/semantic .workspace/feat/semantic main
git worktree add -b feat/pipeline .workspace/feat/pipeline main
git worktree add -b feat/codegen  .workspace/feat/codegen  main
```

### Fase 3: Lançar Agentes Concorrentes

Cada agente recebe instruções precisas. O prompt deve incluir:

1. **Diretório** do worktree (via `--dir` ou `--projectPath`)
2. **Branch** em que está
3. **ADR** que implementa (referência ao arquivo em main)
4. **Arquivos** a criar
5. **Comandos finais** (add, commit, push)

Exemplo de prompt para o agente do lexer:
```
Você está em /repo/.workspace/feat/lexer no branch feat/lexer.

Implemente o lexer conforme ADR-001 (docs/adr/001-lexer-implementation.md).

Crie:
- token/token.go (tipos de token + keywords + LookupIdent)
- lexer/lexer.go (Lexer struct + New + NextToken + métodos auxiliares)

Ao finalizar:
git add token/ lexer/
git commit -m "feat: implement lexer with token types"
git push origin feat/lexer
```

Importante: **não compartilhe contexto entre agentes**. Cada um sabe só o que precisa. O merge resolve as conexões.

### Fase 4: Tempo de Execução

Os agentes rodam em paralelo. O tempo total é o **maior tempo individual**, não a soma.

| Agente | Arquivos | Tempo |
|--------|----------|-------|
| Lexer | token/token.go, lexer/lexer.go | ~30s |
| AST | ast/ast.go | ~20s |
| Parser | parser/parser.go (+ copia token+ast) | ~45s |
| Semantic | semantic/semantic.go (+ copia token+ast+parser+lexer) | ~25s |
| Pipeline | pipeline/pipeline.go, pipeline_test.go | ~20s |
| Codegen | codegen/codegen.go, codegen_test.go (+ copia token+ast+lexer+parser) | ~30s |

**Total paralelo: ~45s** (o mais lento)
**Total sequencial: ~170s** (um atrás do outro)

**Ganho real:** ~75% de redução no tempo.

### Fase 5: Merge da Integração (o ponto crítico)

```bash
# Criar worktree de integração
git worktree add -b feat/integration .workspace/feat/integration main

# Merge um por um (NÃO use octopus merge — falha com conflitos)
cd .workspace/feat/integration
git merge feat/lexer -m "feat: merge lexer"
git merge feat/ast -m "feat: merge ast"
git merge feat/parser -m "feat: merge parser"
git merge feat/semantic -m "feat: merge semantic"
git merge feat/pipeline -m "feat: merge pipeline"
git merge feat/codegen -m "feat: merge codegen"
```

**Problema comum:** Conflitos em `ast/ast.go` ou `token/token.go` porque dois branches criaram o mesmo arquivo. Isso acontece se você NÃO criou os contratos em main (Fase 1.2). Se acontecer:

```bash
git diff --name-only --diff-filter=U   # ver arquivos conflitados
# Se o conteúdo é o mesmo (só diff de import), aceite qualquer versão:
git checkout --ours ast/ast.go
git add ast/ast.go
git commit -m "feat: merge X (resolve conflito em ast/ast.go)"
```

Se os contratos foram criados em main (recomendado), esse passo não terá conflitos — cada branch só criou arquivos novos no seu diretório.

### Fase 6: Build + Testes + Fixes

```bash
go build ./...        # compila tudo
go test ./... -v      # roda testes
go vet ./...          # análise estática

# Se algo quebrar, corrija no branch de integração
# Depois: git add + git commit + git push
```

**Bugs esperados pós-merge (normais):**
1. Interface incompatível — um componente espera `X` mas o outro implementa `Y`
2. Import paths errados — cada branch usou seu próprio `go.mod` path
3. Tipos faltando — um branch assumiu que o outro criaria algo que não criou

**Vantagem:** Você descobre TUDO de uma vez, não um bug por PR sequencial.

### Fase 7: Atualizar ADRs + PR Final

```bash
# Atualizar status dos ADRs de Proposed → Accepted
# (opcional: fazer no branch de integração antes do PR)
for adr in docs/adr/*.md; do
  sed -i '' 's/^## Status$/\n## Status\nAccepted/' "$adr" 2>/dev/null || true
done

git add docs/adr/
git commit -m "docs: update ADR status to Accepted"
git push origin feat/integration

# PR final
gh pr create \
  --base main \
  --head feat/integration \
  --title "Feature: descrição do que foi feito" \
  --body "## ADRs

- ADR-001: $(cat docs/adr/001*.md | grep '^# ' | head -1)
- ADR-002: $(cat docs/adr/002*.md | grep '^# ' | head -1)
...

## O que foi implementado

| Componente | Branch | Arquivos |
|------------|--------|----------|
| Lexer | feat/lexer | token/token.go, lexer/lexer.go |
| AST | feat/ast | ast/ast.go |
| ... | ... | ... |

## Build

- \`go build\`: ✅
- \`go test\`: ✅
- \`go vet\`: ✅"
```

---

## AGENTS.md — Para Copiar e Adaptar

Crie este arquivo em `.opencode/rules.md` ou `AGENTS.md` na raiz do projeto:

```markdown
# Regras para Agentes

## Workflow
1. ADRs em docs/adr/ devem ser criados em main antes da implementação
2. Cada feature usa git worktree: `git worktree add -b feat/<name> .workspace/feat/<name> main`
3. Agentes trabalham em paralelo, cada um no seu worktree (`--dir`)
4. Integração via merge sequencial em um worktree de integração
5. ADRs atualizados para Accepted no PR final

## Commits
- feat: para novas funcionalidades
- fix: para correções
- docs: para documentação (ADRs, comentários)
- refactor: para refatoração
- test: para testes

## Código
- Go module: github.com/org/repo
- Packages por diretório
- Testes unitários com `testing` package
- `go vet` obrigatório antes do push
- Erros como retorno, não panic

## Prompt Padrão para Novos Agentes

Quando for criar um novo agente para implementar uma feature:

1. Leia o ADR correspondente em docs/adr/
2. Crie os arquivos conforme especificado
3. go mod tidy se necessário
4. git add + git commit + git push
5. Não modifique arquivos fora do seu escopo
```

---

## O Que Aprendemos

### ✅ Funciona bem para
- Tarefas que criam **arquivos diferentes** (ex: lexer vs parser vs codegen)
- Tarefas que seguem um **contrato definido** (interfaces conhecidas antes)
- Projetos com **módulos bem isolados** (Go packages, Python modules, Rust crates)
- Quando você quer **PR único** no final, não PRs intermediários

### ❌ Não funciona bem para
- Editar o **mesmo arquivo** concorrentemente (gera conflito no merge)
- Tarefas que dependem **sequencialmente** uma da outra (ex: testar o parser antes de implementar o semantic)
- Quando você não sabe os **contratos/interface** de antemão (vai ter que resolver conflito)

### ⚡ Dicas de Performance
1. **Criar worktrees é instantâneo** (~0.5s cada) — faça todos de uma vez
2. **`git merge` sequencial** (um por um) é mais confiável que octopus merge
3. **Conflitos em arquivos idênticos** (mesmo conteúdo) — `git checkout --ours` resolve
4. **Sempre use `--dir`** nos agentes, nunca `cd` — evita confusão de diretório
5. **Limpe worktrees velhos**: `git worktree prune` + `rm -rf .workspace/*`
6. **Crie contratos em main primeiro** — elimina 90% dos conflitos de merge

### 🔧 Adaptação para outras ferramentas

| Ferramenta | Equivalente ao --dir |
|-----------|---------------------|
| Claude Code | `claude --projectPath .workspace/feat/x` |
| Cursor | `cursor --folder .workspace/feat/x` |
| Aider | `aider .workspace/feat/x` |
| Continue.dev | Configurar workspace no VS Code |
| OpenCode (CLI) | `opencode --dir .workspace/feat/x "<prompt>"` |

### 💾 Caching

O `git worktree` também melhora caching porque:
- Cada worktree tem seu **próprio cache de LSP** (arquivos não mudam externamente)
- O `go build` cacheia objetos compilados por worktree
- O `go.sum` é compartilhado via `.git/objects`

Use `OPENCODE_EXPERIMENTAL_WORKSPACES=true` se estiver usando opencode — ele mantém file watcher + contexto quente por worktree.

---

## Exemplo Comandos Completos (para copiar e colar)

```bash
# === SETUP ===
git clone git@github.com:seu-org/seu-repo.git
cd seu-repo
mkdir -p .workspace docs/adr

# === ADRs (sequencial, em main) ===
git checkout main
for i in 1 2 3; do
  cat > "docs/adr/00$i-description.md" << ADR
# ADR-00$i: Description

## Status
Proposed

## Context
...

## Decision
...

## Consequences
...
ADR
done
git add docs/adr/
git commit -m "docs: add ADRs 001-003"
git push origin main

# === CONTRATOS (sequencial, em main) ===
# Crie as interfaces/tipos compartilhados
git add token/  # se aplicável
git commit -m "feat: add shared token interfaces"
git push origin main

# === AGENTS.md (sequencial, em main) ===
cat > AGENTS.md << 'EOF'
# Regras para Agentes
... (conteúdo do guia)
EOF
git add AGENTS.md
git commit -m "docs: add AGENTS.md with workflow rules"
git push origin main

# === CRIAR WORKTREES ===
for branch in feature-a feature-b feature-c; do
  git worktree add -b "feat/$branch" ".workspace/feat/$branch" main
done

# === LANÇAR AGENTES ===
# Terminal 1
opencode --dir .workspace/feat/feature-a "Implemente feature A conforme ADR-001..."
# Terminal 2
opencode --dir .workspace/feat/feature-b "Implemente feature B conforme ADR-002..."
# Terminal 3
opencode --dir .workspace/feat/feature-c "Implemente feature C conforme ADR-003..."

# === MERGE + TEST ===
git worktree add -b feat/integration .workspace/feat/integration main
cd .workspace/feat/integration
for branch in feature-a feature-b feature-c; do
  git merge "feat/$branch" -m "feat: merge $branch" || {
    echo "Conflito em $branch"
    git diff --name-only --diff-filter=U | xargs git checkout --ours
    git add -A
    git commit -m "feat: merge $branch (resolvido)"
  }
done
go build ./... && go test ./... -v && go vet ./... || echo "Corrija os erros acima"

# === ATUALIZAR ADRs + PR ===
for adr in docs/adr/*.md; do
  sed -i '' 's/Proposed/Accepted/' "$adr"
done
git add -A
git commit -m "feat: integrate all components"
git push origin feat/integration
gh pr create --base main --head feat/integration --title "..." --body "..."
```