package ast

import (
	"fmt"
	"strings"
)

type Node interface {
	TokenLiteral() string
	String() string
}

type Statement interface {
	Node
	statementNode()
}

type Expression interface {
	Node
	expressionNode()
}

// Program is the root of the AST
type Program struct {
	Statements []Statement
}

func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	}
	return ""
}

func (p *Program) String() string {
	var out strings.Builder
	for _, s := range p.Statements {
		out.WriteString(s.String())
		out.WriteString("\n")
	}
	return out.String()
}

// --- Source Statement ---
type SourceStatement struct {
	Token    string // the name token (position, trade, etc)
	Name     string
	S3       *S3Source
	Extracts []ExtractStatement
}

func (s *SourceStatement) statementNode()             {}
func (s *SourceStatement) TokenLiteral() string       { return s.Token }
func (s *SourceStatement) String() string             { return s.Name }

type S3Source struct {
	Bucket string
	Region string
	Prefix string
	Format string
}

// --- Extract Statement ---
type ExtractStatement struct {
	Token   string
	JsonType string   // "select" or "explode" or "extract"
	Path    string   // e.g., "message" or "message.perAcquisition"
	IsArray bool
	Name    string   // table name for "from" case
	Fields  []FieldMapping
}

func (e *ExtractStatement) statementNode()       {}
func (e *ExtractStatement) TokenLiteral() string { return e.Token }
func (e *ExtractStatement) String() string       { return fmt.Sprintf("extract json %s %s", e.JsonType, e.Path) }

type FieldMapping struct {
	SourcePath string // e.g., "client.name"
	Alias      string // e.g., "name"
	Type       FieldType
	Expression Expression // for computed fields like "quantity * price"
}

// --- Field Types ---
type FieldType struct {
	BaseType string   // string, bigint, date, decimal
	Params   []string // e.g., ["18", "2"] for decimal(18,2), ["YYYY-MM-DD"] for date
}

func (ft FieldType) String() string {
	if len(ft.Params) > 0 {
		return fmt.Sprintf("%s(%s)", ft.BaseType, strings.Join(ft.Params, ","))
	}
	return ft.BaseType
}

// --- Binary Expression ---
type BinaryExpression struct {
	Token    string
	Left     Expression
	Operator string
	Right    Expression
}

func (b *BinaryExpression) expressionNode()          {}
func (b *BinaryExpression) TokenLiteral() string     { return b.Token }
func (b *BinaryExpression) String() string {
	return fmt.Sprintf("(%s %s %s)", b.Left.String(), b.Operator, b.Right.String())
}

type Identifier struct {
	Token string
	Value string
}

func (i *Identifier) expressionNode()       {}
func (i *Identifier) TokenLiteral() string  { return i.Token }
func (i *Identifier) String() string        { return i.Value }

// --- Join Statement ---
type JoinStatement struct {
	Token     string
	Left      string        // left table name
	Right     string        // right table name
	Condition string        // join condition as string
	JoinType  string        // "inner" or "left"
	Selects   []SelectField
	Alias     string        // alias for right table
}

func (j *JoinStatement) statementNode()       {}
func (j *JoinStatement) TokenLiteral() string { return j.Token }
func (j *JoinStatement) String() string {
	return fmt.Sprintf("%s /\\ %s -> %s", j.Left, j.Right, j.Condition)
}

// --- Assignment Statement ---
type AssignmentStatement struct {
	Token string
	Name  string
	Value Expression // usually a JoinStatement or SourceStatement reference
}

func (a *AssignmentStatement) statementNode()       {}
func (a *AssignmentStatement) TokenLiteral() string { return a.Token }
func (a *AssignmentStatement) String() string       { return fmt.Sprintf("%s = %s", a.Name, a.Value.String()) }

// --- Select ---
type SelectStatement struct {
	Token  string
	Fields []SelectField
	Output *OutputTarget
}

func (s *SelectStatement) statementNode()       {}
func (s *SelectStatement) TokenLiteral() string { return s.Token }
func (s *SelectStatement) String() string       { return "select ..." }

type SelectField struct {
	Source   string        // table alias or empty
	Path     string        // field path
	Alias    string        // output column name
	FieldType  FieldType
	Pipeline []PipelineStage
}

// --- Pipeline ---
type PipelineStage struct {
	Name        string          // replace, prefix, hash, coalesce, default, cast
	Args        []string        // arguments
	SubPipeline []PipelineStage // nested pipeline in parenthesized blocks
}

// --- Output Target ---
type OutputTarget struct {
	S3          *S3Source
	Mode        string   // overwrite, append, merge
	PartitionBy []string
}

// --- Pipeline Expression ---
type PipelineExpression struct {
	Token  string
	Left   Expression
	Stages []PipelineStage
}

func (p *PipelineExpression) expressionNode()       {}
func (p *PipelineExpression) TokenLiteral() string  { return p.Token }
func (p *PipelineExpression) String() string        { return p.Left.String() + " | ..." }

// --- DatePart Expression ---
type DatePartExpression struct {
	Token  string
	Field  string       // the date field name
	Part   string       // "year", "month", "day"
	Format string       // e.g., "YYYY", "YYYY-MM"
}

func (d *DatePartExpression) expressionNode()       {}
func (d *DatePartExpression) TokenLiteral() string  { return d.Token }
func (d *DatePartExpression) String() string {
	return fmt.Sprintf("%s(%s)", d.Part, d.Field)
}