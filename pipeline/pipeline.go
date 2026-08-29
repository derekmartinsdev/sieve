package pipeline

import (
    "fmt"
    "strings"
)

// Operator defines a transformation step
type Operator struct {
    Name string
    Args []string
}

// Pipeline represents a chain of transformations
type Pipeline struct {
    Operators []Operator
}

// ParsePipeline parses a pipeline expression like: | replace(".", ",") | prefix("R$") | hash() | coalesce("R$0,00")
func ParsePipeline(input string) (*Pipeline, error) {
    p := &Pipeline{}
    input = strings.TrimSpace(input)

    if !strings.HasPrefix(input, "|") {
        return p, nil
    }

    // Split by "|" but respect parentheses
    var current strings.Builder
    depth := 0

    for i := 0; i < len(input); i++ {
        ch := input[i]
        switch ch {
        case '(':
            depth++
            current.WriteByte(ch)
        case ')':
            depth--
            current.WriteByte(ch)
        case '|':
            if depth == 0 {
                op := strings.TrimSpace(current.String())
                if op != "" {
                    parsed := parseOperator(op)
                    if parsed != nil {
                        p.Operators = append(p.Operators, *parsed)
                    }
                }
                current.Reset()
            } else {
                current.WriteByte(ch)
            }
        default:
            current.WriteByte(ch)
        }
    }

    last := strings.TrimSpace(current.String())
    if last != "" {
        parsed := parseOperator(last)
        if parsed != nil {
            p.Operators = append(p.Operators, *parsed)
        }
    }

    return p, nil
}

func parseOperator(input string) *Operator {
    input = strings.TrimSpace(input)
    if input == "" {
        return nil
    }

    idx := strings.Index(input, "(")
    if idx == -1 {
        return &Operator{Name: input}
    }

    name := strings.TrimSpace(input[:idx])
    argsStr := input[idx+1 : len(input)-1] // remove parens
    args := parseArgs(argsStr)

    return &Operator{Name: name, Args: args}
}

func parseArgs(input string) []string {
    if strings.TrimSpace(input) == "" {
        return nil
    }

    var args []string
    var current strings.Builder
    depth := 0
    inQuote := false

    for i := 0; i < len(input); i++ {
        ch := input[i]
        switch {
        case ch == '(':
            depth++
            current.WriteByte(ch)
        case ch == ')':
            depth--
            current.WriteByte(ch)
        case ch == '"':
            inQuote = !inQuote
            current.WriteByte(ch)
        case ch == ',' && depth == 0 && !inQuote:
            arg := strings.TrimSpace(current.String())
            arg = strings.Trim(arg, "\"")
            args = append(args, arg)
            current.Reset()
        default:
            current.WriteByte(ch)
        }
    }

    if current.Len() > 0 {
        arg := strings.TrimSpace(current.String())
        arg = strings.Trim(arg, "\"")
        args = append(args, arg)
    }

    return args
}

// GeneratePySpark generates PySpark column expressions for the pipeline
func (p *Pipeline) GeneratePySpark(columnExpr string) string {
    if len(p.Operators) == 0 {
        return columnExpr
    }

    current := columnExpr
    for _, op := range p.Operators {
        current = generateOp(current, op)
    }
    return current
}

func generateOp(input string, op Operator) string {
    switch op.Name {
    case "replace":
        if len(op.Args) >= 2 {
            return fmt.Sprintf("F.regexp_replace(%s, '%s', '%s')", input, escapeString(op.Args[0]), escapeString(op.Args[1]))
        }
        return input
    case "prefix":
        if len(op.Args) >= 1 {
            return fmt.Sprintf("F.concat(F.lit('%s'), %s)", escapeString(op.Args[0]), input)
        }
        return input
    case "hash":
        return fmt.Sprintf("F.sha2(F.col(%s).cast('string'), 256)", input)
    case "coalesce":
        if len(op.Args) >= 1 {
            return fmt.Sprintf("F.coalesce(%s, F.lit('%s'))", input, escapeString(op.Args[0]))
        }
        return input
    case "default":
        if len(op.Args) >= 1 {
            return fmt.Sprintf("F.coalesce(%s, F.lit(%s))", input, op.Args[0])
        }
        return input
    case "cast":
        if len(op.Args) >= 1 {
            return fmt.Sprintf("F.col(%s).cast('%s')", input, op.Args[0])
        }
        return input
    default:
        return input
    }
}

func escapeString(s string) string {
    s = strings.ReplaceAll(s, "'", "\\'")
    s = strings.ReplaceAll(s, "\"", "\\\"")
    return s
}

// OperatorSet contains all registered operators
type OperatorSet struct {
    ops map[string]func(args []string, columnExpr string) string
}

func NewOperatorSet() *OperatorSet {
    return &OperatorSet{
        ops: map[string]func(args []string, columnExpr string) string{
            "replace": func(args []string, col string) string {
                if len(args) >= 2 {
                    return fmt.Sprintf("F.regexp_replace(%s, '%s', '%s')", col, args[0], args[1])
                }
                return col
            },
            "prefix": func(args []string, col string) string {
                if len(args) >= 1 {
                    return fmt.Sprintf("F.concat(F.lit('%s'), %s)", args[0], col)
                }
                return col
            },
            "hash": func(args []string, col string) string {
                return fmt.Sprintf("F.sha2(%s.cast('string'), 256)", col)
            },
            "coalesce": func(args []string, col string) string {
                if len(args) >= 1 {
                    return fmt.Sprintf("F.coalesce(%s, F.lit('%s'))", col, args[0])
                }
                return col
            },
            "default": func(args []string, col string) string {
                if len(args) >= 1 {
                    return fmt.Sprintf("F.coalesce(%s, F.lit(%s))", col, args[0])
                }
                return col
            },
            "cast": func(args []string, col string) string {
                if len(args) >= 1 {
                    return fmt.Sprintf("F.col(%s).cast('%s')", col, args[0])
                }
                return col
            },
            "split": func(args []string, col string) string {
                if len(args) >= 1 {
                    return fmt.Sprintf("F.split(%s, '%s')", col, args[0])
                }
                return col
            },
        },
    }
}

func (os *OperatorSet) Apply(name string, args []string, columnExpr string) (string, error) {
    fn, ok := os.ops[name]
    if !ok {
        return "", fmt.Errorf("unknown pipeline operator: %s", name)
    }
    return fn(args, columnExpr), nil
}