package ast

import "fmt"

type Node interface {
	nodeMarker()
}

type Program struct {
	Statements []Statement
}

func (p *Program) nodeMarker() {}

type Statement interface {
	Node
	statementMarker()
}

type Section struct {
	Name       string
	Source     *Source
	Extracts   []*Extract
	Joins      []*Join
	Select     *SelectStmt
	Sink       *Sink
	Transforms []TransformChain
}

func (s *Section) nodeMarker()      {}
func (s *Section) statementMarker() {}

type Source struct {
	From   string // "s3", or a section name for derived sources
	Bucket string
	Region string
	Prefix string
	Format string // "delta"
}

func (s *Source) nodeMarker() {}

type Extract struct {
	Explode     *ExtractExplode
	JsonSelect  *JsonSelect
	JsonExtract *JsonExtract
}

func (e *Extract) nodeMarker() {}

type ExtractExplode struct {
	Path   string // e.g. "message.perAcquisition"
	As     string // "array"
	Fields []*FieldDef
}

type JsonSelect struct {
	Path   string // e.g. "message"
	Fields []*FieldDef
}

type JsonExtract struct {
	Path   string // e.g. "perAcquisition.taxes"
	As     string // "array"
	Fields []*FieldDef
}

type BinaryExpr struct {
	Left     string
	Operator string // "*", "+", "-", "/"
	Right    string
}

type FieldDef struct {
	Name         string
	Source       string // e.g. "client.name"
	Alias        string
	DataType     string // "string", "bigint", "date(YYYY-MM-DD)", "decimal(18,2)"
	ComputedExpr *BinaryExpr
	JsonColumn   string // json column to extract from (e.g. "message", "exploded")
}

type Join struct {
	Left       string
	Right      string
	LeftAlias  string
	RightAlias string
	Condition  *JoinCondition
	JoinType   string // "inner", "left", "right", ""
	Select     *SelectStmt
}

func (j *Join) nodeMarker() {}

type JoinCondition struct {
	LeftRefs  []string
	RightRefs []string
}

type SelectStmt struct {
	Fields []SelectField
	Source string // optional name for source if select is at section level
}

func (s *SelectStmt) nodeMarker() {}

type SelectField struct {
	Name      string
	Alias     string
	DataType  string
	Transform *TransformChain
	Function  string // "" or "year"/"month"/"day"
}

type TransformChain struct {
	Steps []TransformStep
	Alias string
}

type TransformStep struct {
	Type string // "cast", "default", "replace", "prefix", "hash"
	Args []string
}

type Sink struct {
	To          string // "s3"
	Bucket      string
	Region      string
	Prefix      string
	Format      string
	Mode        string // "overwrite", "append", "merge"
	PartitionBy []string
}

func (s *Sink) nodeMarker() {}

func (s *Section) String() string { return fmt.Sprintf("Section(%s)", s.Name) }

func (j *Join) String() string {
	return fmt.Sprintf("Join(%s/%s->%s)", j.Left, j.LeftAlias, j.Right)
}

type Visitor interface {
	Visit(node Node) bool
}

func Walk(v Visitor, node Node) {
	if !v.Visit(node) {
		return
	}
	switch n := node.(type) {
	case *Program:
		for _, s := range n.Statements {
			Walk(v, s)
		}
	case *Section:
		if n.Source != nil {
			Walk(v, n.Source)
		}
		for _, e := range n.Extracts {
			Walk(v, e)
		}
		for _, jn := range n.Joins {
			Walk(v, jn)
		}
		if n.Select != nil {
			Walk(v, n.Select)
		}
		if n.Sink != nil {
			Walk(v, n.Sink)
		}
	}
}
