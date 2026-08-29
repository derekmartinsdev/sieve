# ADR-001: Implement Add Function

## Status
Accepted

## Context
The sieve project needs a basic arithmetic addition function to support core mathematical operations.

## Decision
Implement a simple `add(a, b)` function that returns the sum of two numbers. The function will:
- Accept two integer arguments
- Return their sum as an integer
- Handle basic edge cases (e.g., negative numbers, zero)

## Consequences
- Provides a reusable addition primitive for other features
- Keeps the implementation simple with no external dependencies
- Easy to unit test