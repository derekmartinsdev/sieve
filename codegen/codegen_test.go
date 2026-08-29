package codegen

import (
	"testing"
)

func TestVarName(t *testing.T) {
	g := New()
	tests := []struct {
		input    string
		expected string
	}{
		{"position", "df_position"},
		{"trade", "df_trade"},
		{"perAcquisition", "df_perAcquisition"},
	}

	for _, tt := range tests {
		result := g.varName(tt.input)
		if result != tt.expected {
			t.Errorf("varName(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestToSparkType(t *testing.T) {
	g := New()
	result := g.toSparkType(struct{ BaseType string; Params []string }{BaseType: "string", Params: nil})
	if result != "StringType()" {
		t.Errorf("expected StringType(), got %s", result)
	}
}