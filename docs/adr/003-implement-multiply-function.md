# ADR-003: Implement Multiply Function

## Status
Accepted

## Context
The sieve project needs a multiplication function to complement the existing add and subtract operations.

## Decision
Implement a simple `multiply(a, b)` function that returns the product of two numbers. The function will:
- Accept two integer arguments
- Return their product as an integer
- Handle basic edge cases (e.g., zero, negative numbers)

## Consequences
- Provides a reusable multiplication primitive
- Consistent with the pattern established by add and subtract
- Easy to unit test