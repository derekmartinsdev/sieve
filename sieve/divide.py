def divide(a: int, b: int) -> int:
    if b == 0:
        raise ValueError("Cannot divide by zero")
    return a // b