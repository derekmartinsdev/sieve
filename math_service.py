from .add import add
from .subtract import subtract
from .multiply import multiply
from .divide import divide


def calculate(a: int, b: int, operator: str) -> int:
    operations = {
        "add": add,
        "subtract": subtract,
        "multiply": multiply,
        "divide": divide,
    }
    if operator not in operations:
        raise ValueError(f"Unknown operator: {operator}")
    return operations[operator](a, b)