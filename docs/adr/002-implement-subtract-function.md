# ADR-002: Implement Subtract Function

## Status
Accepted

## Context
The sieve project needs a subtraction function to complement the existing add operation.

## Decision
Implement a simple `subtract(a, b)` function that returns the difference of two numbers. The function will:
- Accept two integer arguments
- Return their difference as an integer

## Consequences
- Provides a reusable subtraction primitive
- Consistent with the pattern established by other arithmetic operations
- Easy to unit test