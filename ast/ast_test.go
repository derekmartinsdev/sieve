package ast

import "testing"

func TestSectionImplementsNodeAndStatement(t *testing.T) {
	sec := &Section{Name: "test"}

	var n Node = sec
	var s Statement = sec

	_ = n
	_ = s

	if sec.Name != "test" {
		t.Errorf("expected name 'test', got %q", sec.Name)
	}
}

func TestNodeInterface(t *testing.T) {
	_ = Node(&Section{})
	_ = Node(&Program{})
}

func TestStatementInterface(t *testing.T) {
	_ = Statement(&Section{})
}

func TestASTStructure(t *testing.T) {
	prog := &Program{
		Statements: []Statement{
			&Section{
				Name: "s1",
				Source: &Source{
					From:   "s3",
					Bucket: "b1",
				},
				Select: &SelectStmt{
					Fields: []SelectField{
						{Name: "f1", Alias: "a1"},
					},
				},
				Extracts: []*Extract{
					{
						JsonSelect: &JsonSelect{
							Path: "message",
							Fields: []*FieldDef{
								{Name: "field1", Source: "field1", JsonColumn: "message"},
							},
						},
					},
				},
				Transforms: []TransformChain{
					{
						Alias: "alias",
						Steps: []TransformStep{
							{Type: "col", Args: []string{"src"}},
							{Type: "cast", Args: []string{"int"}},
						},
					},
				},
				Sink: &Sink{
					To:     "s3",
					Bucket: "out",
					Format: "delta",
					Mode:   "overwrite",
				},
			},
			&Section{
				Name: "s2",
				Joins: []*Join{
					{
						Left:     "s1",
						Right:    "s3",
						JoinType: "INNER",
						Condition: &JoinCondition{
							LeftRefs:  []string{"s1.id"},
							RightRefs: []string{"s3.id"},
						},
					},
				},
			},
		},
	}

	if len(prog.Statements) != 2 {
		t.Errorf("expected 2 statements, got %d", len(prog.Statements))
	}

	s1 := prog.Statements[0].(*Section)
	if s1.Source.From != "s3" {
		t.Errorf("expected from 's3', got %q", s1.Source.From)
	}
	if len(s1.Extracts) != 1 {
		t.Errorf("expected 1 extract, got %d", len(s1.Extracts))
	}
	if len(s1.Transforms) != 1 {
		t.Errorf("expected 1 transform chain, got %d", len(s1.Transforms))
	}
	if s1.Transforms[0].Steps[0].Type != "col" {
		t.Errorf("expected step type 'col', got %q", s1.Transforms[0].Steps[0].Type)
	}
	if s1.Sink.Mode != "overwrite" {
		t.Errorf("expected mode 'overwrite', got %q", s1.Sink.Mode)
	}

	s2 := prog.Statements[1].(*Section)
	if len(s2.Joins) != 1 {
		t.Errorf("expected 1 join, got %d", len(s2.Joins))
	}
	if s2.Joins[0].JoinType != "INNER" {
		t.Errorf("expected join type 'INNER', got %q", s2.Joins[0].JoinType)
	}
}

func TestBinaryExpr(t *testing.T) {
	expr := &BinaryExpr{Left: "a", Operator: "+", Right: "b"}
	if expr.Left != "a" || expr.Operator != "+" || expr.Right != "b" {
		t.Error("BinaryExpr fields not set correctly")
	}
}
