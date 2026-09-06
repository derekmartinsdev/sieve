# ADR-001: Indentation-based DSL with zero external dependencies

## Status
Accepted

## Context
We need a DSL to transpile into PySpark code. The DSL must be human-readable, support nested structures, and the transpiler must be distributable as a single static binary.

## Decision
1. **Indentation-based parsing**: sections start at column 1, body content at column 5+. This avoids braces, semicolons, or other delimiters. See [lexer.md](../lexer.md) for the token system (IDENT, Column, etc.).
2. **Go stdlib only**: no external dependencies. The transpiler compiles to a single static binary.
3. **Recursive descent parser**: custom parser producing an AST, not a parser generator.
4. **File extensions**: `.pipeline.sieve`, `.transform.sieve`, `.quality.sieve`, `.schedule.sieve` — each corresponds to a different domain concern.

## Consequences
- Simple, readable DSL syntax
- Zero supply-chain risk
- Single binary distribution across Linux, macOS, Windows
- Parser must be maintained manually (no code generation)
- Indentation errors are harder to debug than bracket-based grammars