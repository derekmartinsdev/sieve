package pipeline

import (
    "testing"
)

func TestParsePipeline(t *testing.T) {
    tests := []struct {
        input    string
        expected int // number of operators
    }{
        {"| replace(\".\", \",\") | prefix(\"R$\") | hash() | coalesce(\"R$0,00\")", 4},
        {"| default(0) | cast(string) | replace(\".\", \",\") | prefix(\"R$\") | hash()", 5},
        {"", 0},
        {"| hash()", 1},
    }

    for _, tt := range tests {
        p, err := ParsePipeline(tt.input)
        if err != nil {
            t.Errorf("ParsePipeline(%q) error: %v", tt.input, err)
        }
        if len(p.Operators) != tt.expected {
            t.Errorf("ParsePipeline(%q) got %d operators, want %d", tt.input, len(p.Operators), tt.expected)
        }
    }
}

func TestGeneratePySpark(t *testing.T) {
    p := &Pipeline{
        Operators: []Operator{
            {Name: "replace", Args: []string{".", ","}},
            {Name: "prefix", Args: []string{"R$"}},
        },
    }

    result := p.GeneratePySpark("F.col(\"financeiro\")")
    if result == "" {
        t.Error("GeneratePySpark returned empty")
    }
}