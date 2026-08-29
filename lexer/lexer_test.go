package lexer

import (
	"testing"

	"github.com/derekmartinsdev/sieve/token"
)

func TestNextTokenSimpleDSL(t *testing.T) {
	input := `my_pipeline
  from s3 bucket data prefix events format delta
  extract json select message
    event_id string
    timestamp date`

	tests := []struct {
		expectedType    token.TokenType
		expectedLiteral string
		expectedLine    int
		expectedCol     int
	}{
		{token.IDENT, "my_pipeline", 1, 1},
		{token.FROM, "from", 2, 3},
		{token.S3, "s3", 2, 8},
		{token.BUCKET, "bucket", 2, 11},
		{token.IDENT, "data", 2, 18},
		{token.PREFIX, "prefix", 2, 23},
		{token.IDENT, "events", 2, 30},
		{token.FORMAT, "format", 2, 37},
		{token.DELTA, "delta", 2, 44},
		{token.EXTRACT, "extract", 3, 3},
		{token.JSON, "json", 3, 11},
		{token.SELECT, "select", 3, 16},
		{token.IDENT, "message", 3, 23},
		{token.IDENT, "event_id", 4, 5},
		{token.TYPE_STRING, "string", 4, 14},
		{token.IDENT, "timestamp", 5, 5},
		{token.TYPE_DATE, "date", 5, 14},
		{token.EOF, "", 5, 18},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("test[%d] - wrong token type. expected=%q, got=%q (literal=%q)",
				i, tt.expectedType, tok.Type, tok.Literal)
		}
		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("test[%d] - wrong literal. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
		if tok.Line != tt.expectedLine {
			t.Fatalf("test[%d] - wrong line for token %q. expected=%d, got=%d",
				i, tok.Literal, tt.expectedLine, tok.Line)
		}
		if tok.Column != tt.expectedCol {
			t.Fatalf("test[%d] - wrong column for token %q. expected=%d, got=%d",
				i, tok.Literal, tt.expectedCol, tok.Column)
		}
	}
}

func TestSkipComments(t *testing.T) {
	input := `// this is a comment
section_name
  // another comment
  iden`

	l := New(input)

	tok := l.NextToken()
	if tok.Type != token.IDENT || tok.Literal != "section_name" {
		t.Fatalf("expected IDENT 'section_name', got %q (%q)", tok.Type, tok.Literal)
	}
	if tok.Line != 2 {
		t.Fatalf("expected line 2, got %d", tok.Line)
	}

	tok = l.NextToken()
	if tok.Type != token.IDENT || tok.Literal != "iden" {
		t.Fatalf("expected IDENT 'iden', got %q (%q)", tok.Type, tok.Literal)
	}
	if tok.Line != 4 {
		t.Fatalf("expected line 4, got %d", tok.Line)
	}

	tok = l.NextToken()
	if tok.Type != token.EOF {
		t.Fatalf("expected EOF, got %q", tok.Type)
	}
}

func TestStringLiterals(t *testing.T) {
	input := `"hello world" 'single quoted'`

	l := New(input)

	tok := l.NextToken()
	if tok.Type != token.STRING || tok.Literal != "hello world" {
		t.Fatalf("expected STRING 'hello world', got %q (%q)", tok.Type, tok.Literal)
	}

	tok = l.NextToken()
	if tok.Type != token.STRING || tok.Literal != "single quoted" {
		t.Fatalf("expected STRING 'single quoted', got %q (%q)", tok.Type, tok.Literal)
	}
}

func TestTwoCharOperators(t *testing.T) {
	input := `-> /\`

	l := New(input)

	tok := l.NextToken()
	if tok.Type != token.ARROW || tok.Literal != "->" {
		t.Fatalf("expected ARROW '->', got %q (%q)", tok.Type, tok.Literal)
	}

	tok = l.NextToken()
	if tok.Type != token.JOIN || tok.Literal != "/\\" {
		t.Fatalf("expected JOIN '/\\', got %q (%q)", tok.Type, tok.Literal)
	}
}

func TestNumberParsing(t *testing.T) {
	input := `42 3.14`

	l := New(input)

	tok := l.NextToken()
	if tok.Type != token.INT || tok.Literal != "42" {
		t.Fatalf("expected INT '42', got %q (%q)", tok.Type, tok.Literal)
	}

	tok = l.NextToken()
	if tok.Type != token.FLOAT || tok.Literal != "3.14" {
		t.Fatalf("expected FLOAT '3.14', got %q (%q)", tok.Type, tok.Literal)
	}
}

func TestDateParsing(t *testing.T) {
	input := `2025-01-15`

	l := New(input)

	tok := l.NextToken()
	if tok.Type != token.STRING || tok.Literal != "2025-01" {
		t.Fatalf("expected STRING '2025-01', got %q (%q)", tok.Type, tok.Literal)
	}
}

func TestEOF(t *testing.T) {
	input := `hello`
	l := New(input)

	tok := l.NextToken()
	if tok.Type != token.IDENT {
		t.Fatalf("expected IDENT, got %q", tok.Type)
	}

	tok = l.NextToken()
	if tok.Type != token.EOF || tok.Literal != "" {
		t.Fatalf("expected EOF with empty literal, got %q (%q)", tok.Type, tok.Literal)
	}
}

func TestTabsFourSpaceIndentation(t *testing.T) {
	input := "\tsection1\n\t\tfield1"

	l := New(input)

	tok := l.NextToken()
	if tok.Type != token.IDENT || tok.Literal != "section1" {
		t.Fatalf("expected IDENT 'section1', got %q (%q)", tok.Type, tok.Literal)
	}
	if tok.Column != 5 {
		t.Fatalf("expected column 5 (tab=4+1 offset), got %d", tok.Column)
	}

	tok = l.NextToken()
	if tok.Type != token.IDENT || tok.Literal != "field1" {
		t.Fatalf("expected IDENT 'field1', got %q (%q)", tok.Type, tok.Literal)
	}
	if tok.Column != 8 {
		t.Fatalf("expected column 8 (2 tabs=8+1 offset, minus EOF readChar), got %d", tok.Column)
	}
}

func TestMixedSpacesTabsColumnNumbers(t *testing.T) {
	input := "  a\n\tb\n  \t c"

	l := New(input)

	tok := l.NextToken()
	if tok.Type != token.IDENT || tok.Literal != "a" {
		t.Fatalf("expected IDENT 'a', got %q (%q)", tok.Type, tok.Literal)
	}
	if tok.Column != 3 {
		t.Fatalf("expected column 3, got %d", tok.Column)
	}

	tok = l.NextToken()
	if tok.Type != token.IDENT || tok.Literal != "b" {
		t.Fatalf("expected IDENT 'b', got %q (%q)", tok.Type, tok.Literal)
	}
	if tok.Column != 5 {
		t.Fatalf("expected column 5, got %d", tok.Column)
	}

	tok = l.NextToken()
	if tok.Type != token.IDENT || tok.Literal != "c" {
		t.Fatalf("expected IDENT 'c', got %q (%q)", tok.Type, tok.Literal)
	}
	if tok.Column != 7 {
		t.Fatalf("expected column 7, got %d", tok.Column)
	}
}
