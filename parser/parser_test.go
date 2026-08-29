package parser

import (
	"strings"
	"testing"

	"github.com/derekmartinsdev/sieve/ast"
)

func parse(input string) (*ast.Program, []string) {
	p := New(input)
	prog := p.ParseProgram()
	return prog, p.Errors()
}

func TestParseSimpleSectionFromExtract(t *testing.T) {
	input := `events
  from s3 bucket mybucket prefix logs format delta
  extract json select message
    client.age bigint
    client.name string`

	prog, errs := parse(input)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(prog.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(prog.Statements))
	}

	sec, ok := prog.Statements[0].(*ast.Section)
	if !ok {
		t.Fatal("expected *ast.Section")
	}
	if sec.Name != "events" {
		t.Errorf("expected section name 'events', got %q", sec.Name)
	}
	if sec.Source == nil {
		t.Fatal("expected source")
	}
	if sec.Source.From != "s3" {
		t.Errorf("expected from 's3', got %q", sec.Source.From)
	}
	if sec.Source.Bucket != "mybucket" {
		t.Errorf("expected bucket 'mybucket', got %q", sec.Source.Bucket)
	}
	if sec.Source.Prefix != "logs" {
		t.Errorf("expected prefix 'logs', got %q", sec.Source.Prefix)
	}
	if sec.Source.Format != "delta" {
		t.Errorf("expected format 'delta', got %q", sec.Source.Format)
	}

	if len(sec.Extracts) != 1 {
		t.Fatalf("expected 1 extract, got %d", len(sec.Extracts))
	}
	ext := sec.Extracts[0]
	if ext.JsonSelect == nil {
		t.Fatal("expected json select")
	}
	if ext.JsonSelect.Path != "message" {
		t.Errorf("expected json path 'message', got %q", ext.JsonSelect.Path)
	}
	if len(ext.JsonSelect.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(ext.JsonSelect.Fields))
	}
	if ext.JsonSelect.Fields[0].Name != "age" {
		t.Errorf("expected field name 'age', got %q", ext.JsonSelect.Fields[0].Name)
	}
	if ext.JsonSelect.Fields[1].Name != "name" {
		t.Errorf("expected field name 'name', got %q", ext.JsonSelect.Fields[1].Name)
	}
}

func TestParseComputedColumns(t *testing.T) {
	input := `sales
  from s3 bucket data prefix sales format delta
  extract json select message
    quantity * price total`

	prog, errs := parse(input)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	sec := prog.Statements[0].(*ast.Section)
	fields := sec.Extracts[0].JsonSelect.Fields
	if len(fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(fields))
	}

	computed := fields[0]
	if computed.ComputedExpr == nil {
		t.Fatal("expected computed expression")
	}
	if computed.ComputedExpr.Left != "quantity" {
		t.Errorf("expected left 'quantity', got %q", computed.ComputedExpr.Left)
	}
	if computed.ComputedExpr.Right != "price" {
		t.Errorf("expected right 'price', got %q", computed.ComputedExpr.Right)
	}
	if computed.ComputedExpr.Operator != "*" {
		t.Errorf("expected operator '*', got %q", computed.ComputedExpr.Operator)
	}
}

func TestParseJoinWithAliases(t *testing.T) {
	input := `combined = left_section(l) /\ right_section(r) -> l.id = r.id, inner
  select id name`

	prog, errs := parse(input)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	if len(prog.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(prog.Statements))
	}
	sec := prog.Statements[0].(*ast.Section)
	if len(sec.Joins) != 1 {
		t.Fatalf("expected 1 join, got %d", len(sec.Joins))
	}

	join := sec.Joins[0]
	if join.Left != "left_section" {
		t.Errorf("expected left 'left_section', got %q", join.Left)
	}
	if join.LeftAlias != "l" {
		t.Errorf("expected left alias 'l', got %q", join.LeftAlias)
	}
	if join.Right != "right_section" {
		t.Errorf("expected right 'right_section', got %q", join.Right)
	}
	if join.RightAlias != "r" {
		t.Errorf("expected right alias 'r', got %q", join.RightAlias)
	}
	if join.JoinType != "inner" {
		t.Errorf("expected join type 'inner', got %q", join.JoinType)
	}
	if join.Condition == nil {
		t.Fatal("expected join condition")
	}
	if len(join.Condition.LeftRefs) != 1 || join.Condition.LeftRefs[0] != "l.id" {
		t.Errorf("expected left refs ['l.id'], got %v", join.Condition.LeftRefs)
	}
}

func TestParseTransformChains(t *testing.T) {
	input := `clean
  from s3 bucket data prefix raw format delta
  transform
    user_id = raw_id string | hash | prefix("usr_")
    amount = value decimal(18,2) | default(0) | cast(bigint)`

	prog, errs := parse(input)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	sec := prog.Statements[0].(*ast.Section)
	if len(sec.Transforms) != 2 {
		t.Fatalf("expected 2 transforms, got %d", len(sec.Transforms))
	}

	tc1 := sec.Transforms[0]
	if tc1.Alias != "user_id" {
		t.Errorf("expected alias 'user_id', got %q", tc1.Alias)
	}
	if len(tc1.Steps) < 2 {
		t.Fatalf("expected at least 2 steps, got %d", len(tc1.Steps))
	}
	if tc1.Steps[0].Type != "col" {
		t.Errorf("expected step type 'col', got %q", tc1.Steps[0].Type)
	}
	if tc1.Steps[0].Args[0] != "raw_id" {
		t.Errorf("expected col source 'raw_id', got %q", tc1.Steps[0].Args[0])
	}

	tc2 := sec.Transforms[1]
	if tc2.Alias != "amount" {
		t.Errorf("expected alias 'amount', got %q", tc2.Alias)
	}
}

func TestParseDateFunctions(t *testing.T) {
	input := `reports
  from s3 bucket data prefix reports format delta
  select
    created_at month() m, order_date day()`

	prog, errs := parse(input)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	sec := prog.Statements[0].(*ast.Section)
	if sec.Select == nil {
		t.Fatal("expected select")
	}
	if len(sec.Select.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(sec.Select.Fields))
	}

	f1 := sec.Select.Fields[0]
	if f1.Function != "month" {
		t.Errorf("expected function 'month', got %q", f1.Function)
	}
	if f1.Name != "created_at" {
		t.Errorf("expected name 'created_at', got %q", f1.Name)
	}
	if f1.Alias != "m" {
		t.Errorf("expected alias 'm', got %q", f1.Alias)
	}

	f2 := sec.Select.Fields[1]
	if f2.Function != "day" {
		t.Errorf("expected function 'day', got %q", f2.Function)
	}
}

func TestParseErrorsReported(t *testing.T) {
	input := `bad_section
  to s4`

	_, errs := parse(input)
	if len(errs) == 0 {
		t.Fatal("expected errors, got none")
	}
}

func TestMissingArrayAfterExplode(t *testing.T) {
	input := `events
  from s3 bucket data prefix raw format delta
  extract json explode message.inner
    field1 string`

	_, errs := parse(input)
	if len(errs) == 0 {
		t.Fatal("expected errors for missing 'array' after explode")
	}
	found := false
	for _, err := range errs {
		if strings.Contains(err, "expected 'array' after json explode") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error about missing 'array', got: %v", errs)
	}

	prog, _ := parse(input)
	sec := prog.Statements[0].(*ast.Section)
	if len(sec.Extracts) == 0 {
		t.Error("expected extract to exist even with errors")
	}
}

func TestMissingJsonAfterExtract(t *testing.T) {
	input := `events
  from s3 bucket data prefix raw format delta
  extract delta select message
    field1 string`

	_, errs := parse(input)
	if len(errs) == 0 {
		t.Fatal("expected errors for missing 'json'")
	}

	found := false
	for _, err := range errs {
		if strings.Contains(err, "'extract' must be followed by 'json'") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error about missing 'json', got: %v", errs)
	}
}

func TestMissingJoinOperator(t *testing.T) {
	input := `combined = left_section -> l.id = r.id`

	_, errs := parse(input)
	if len(errs) == 0 {
		t.Fatal("expected errors for missing join operator")
	}

	found := false
	for _, err := range errs {
		if strings.Contains(err, "expected join operator") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error about missing join operator, got: %v", errs)
	}
}
