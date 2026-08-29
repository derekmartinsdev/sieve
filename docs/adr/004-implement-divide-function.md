# ADR-004: Implement Divide Function

## Status
Accepted

## Context
The sieve project needs a division function to complement the existing add, subtract, and multiply operations.

## Decision
Implement a simple `divide(a, b)` function that returns the quotient of two numbers. The function will:
- Accept two integer arguments
- Return their quotient as an integer
- Handle division by zero by returning an error

## Consequences
- Provides a reusable division primitive
- Consistent with the pattern established by other arithmetic operations
- Easy to unit test