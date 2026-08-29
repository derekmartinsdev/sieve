# ADR-002: Implement Subtract Function

## Status
Proposed

## Context
The sieve project needs a basic arithmetic subtraction function to complement the existing addition operation and support core mathematical operations.

## Decision
Implement a simple `subtract(a, b)` function that returns the difference of two numbers (a - b). The function will:
- Accept two integer arguments
- Return their difference as an integer
- Handle basic edge cases (e.g., negative results, zero)

## Consequences
- Provides a reusable subtraction primitive for other features
- Keeps the implementation simple with no external dependencies
- Easy to unit test
- Consistent with the pattern established by the add function