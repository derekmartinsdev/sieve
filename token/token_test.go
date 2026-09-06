package token

import "testing"

func TestLookupIdentKeywords(t *testing.T) {
	tests := []struct {
		input    string
		expected TokenType
	}{
		{"from", FROM},
		{"s3", S3},
		{"bucket", BUCKET},
		{"region", REGION},
		{"prefix", PREFIX},
		{"format", FORMAT},
		{"delta", DELTA},
		{"extract", EXTRACT},
		{"json", JSON},
		{"select", SELECT},
		{"explode", EXPLODE},
		{"message", IDENT},
		{"array", ARRAY},
		{"string", TYPE_STRING},
		{"bigint", BIGINT},
		{"date", TYPE_DATE},
		{"decimal", DECIMAL},
		{"to", TO},
		{"mode", MODE},
		{"overwrite", OVERWRITE},
		{"append", APPEND},
		{"merge", MERGE},
		{"partitioned", PARTITIONED},
		{"default", DEFAULT_KW},
		{"cast", CAST},
		{"replace", REPLACE},
		{"hash", HASH},
		{"transform", TRANSFORM},
		{"left", LEFT},
		{"right", RIGHT},
		{"year", YEAR},
		{"month", MONTH},
		{"day", DAY},
		{"or", OR},
	}

	for _, tt := range tests {
		tok := LookupIdent(tt.input)
		if tok != tt.expected {
			t.Errorf("LookupIdent(%q) = %q, want %q", tt.input, tok, tt.expected)
		}
	}
}

func TestLookupIdentUnknown(t *testing.T) {
	unknowns := []string{"foo", "bar", "baz", "my_column", "some_field", "x"}

	for _, ident := range unknowns {
		tok := LookupIdent(ident)
		if tok != IDENT {
			t.Errorf("LookupIdent(%q) = %q, want %q", ident, tok, IDENT)
		}
	}
}
