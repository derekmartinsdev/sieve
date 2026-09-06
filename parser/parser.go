package parser

import (
	"fmt"
	"strings"

	"github.com/derekmartinsdev/sieve/ast"
	"github.com/derekmartinsdev/sieve/lexer"
	"github.com/derekmartinsdev/sieve/token"
)

type Parser struct {
	l      *lexer.Lexer
	errors []string

	curToken  token.Token
	peekToken token.Token
}

func New(input string) *Parser {
	p := &Parser{
		l:      lexer.New(input),
		errors: []string{},
	}

	p.nextToken()
	p.nextToken()

	return p
}

func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}

	for !p.curTokenIs(token.EOF) {
		if p.curTokenIs(token.IDENT) && p.curToken.Column <= 1 {
			section := p.parseSection()
			if section != nil {
				program.Statements = append(program.Statements, section)
			}
		} else {
			p.nextToken()
		}
	}

	return program
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) curTokenIs(t token.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t token.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) error(msg string) {
	p.errors = append(p.errors, fmt.Sprintf("line %d col %d: %s", p.curToken.Line, p.curToken.Column, msg))
}

func (p *Parser) isSectionStart() bool {
	return p.curTokenIs(token.IDENT) && p.curToken.Column <= 1
}

func (p *Parser) isEOF() bool {
	return p.curTokenIs(token.EOF)
}

func (p *Parser) isBodyKeyword() bool {
	switch p.curToken.Type {
	case token.FROM, token.EXTRACT, token.SELECT, token.TO, token.TRANSFORM:
		return true
	default:
		return false
	}
}

func (p *Parser) isTypeKeyword(t token.TokenType) bool {
	switch t {
	case token.TYPE_STRING, token.BIGINT, token.TYPE_DATE, token.DECIMAL, token.FLOAT, token.INT:
		return true
	default:
		return false
	}
}

func (p *Parser) isIdentOrDateFunc() bool {
	return p.curTokenIs(token.IDENT) || isDateFunc(p.curToken)
}

func (p *Parser) parseSection() *ast.Section {
	section := &ast.Section{Name: p.curToken.Literal}

	p.nextToken()

	for !p.isEOF() && !p.isSectionStart() {
		switch p.curToken.Type {
		case token.FROM:
			section.Source = p.parseFrom()
		case token.EXTRACT:
			ext := p.parseExtract()
			if ext != nil {
				section.Extracts = append(section.Extracts, ext)
			}
		case token.SELECT:
			section.Select = p.parseSelect()
		case token.TO:
			section.Sink = p.parseSink()
		case token.TRANSFORM:
			section.Transforms = p.parseTransformBlock()
		case token.ASSIGN:
			return p.parseDerivedSection(section)
		default:
			p.nextToken()
		}
	}

	return section
}

func (p *Parser) parseFrom() *ast.Source {
	source := &ast.Source{}

	p.nextToken()

	if p.curTokenIs(token.S3) {
		source.From = "s3"
		p.parseS3Source(source)
	} else if p.curTokenIs(token.IDENT) {
		source.From = p.curToken.Literal
		p.nextToken()
	}

	return source
}

func (p *Parser) parseS3Source(source *ast.Source) {
	p.nextToken()

	for !p.isEOF() && !p.isSectionStart() && !p.isBodyKeyword() {
		switch p.curToken.Type {
		case token.BUCKET:
			p.nextToken()
			source.Bucket = p.curToken.Literal
			p.nextToken()
		case token.REGION:
			p.nextToken()
			source.Region = p.curToken.Literal
			p.nextToken()
		case token.PREFIX:
			p.nextToken()
			source.Prefix = p.curToken.Literal
			p.nextToken()
		case token.FORMAT:
			p.nextToken()
			source.Format = p.curToken.Literal
			p.nextToken()
		default:
			p.nextToken()
		}
	}
}

func isDateFunc(t token.Token) bool {
	return t.Type == token.YEAR || t.Type == token.MONTH || t.Type == token.DAY
}

func (p *Parser) parseDottedPath() string {
	var parts []string
	parts = append(parts, p.curToken.Literal)
	p.nextToken()

	for p.curTokenIs(token.DOT) {
		p.nextToken()
		if p.curTokenIs(token.IDENT) {
			parts = append(parts, p.curToken.Literal)
			p.nextToken()
		} else {
			break
		}
	}

	return strings.Join(parts, ".")
}

func (p *Parser) parseAlias(sameLine int) string {
	if !p.isIdentOrDateFunc() || p.isTypeKeyword(p.curToken.Type) || p.peekTokenIs(token.DOT) || p.curToken.Column <= 1 {
		return ""
	}
	if isDateFunc(p.peekToken) {
		return ""
	}
	if sameLine > 0 && p.curToken.Line != sameLine {
		return ""
	}
	alias := p.curToken.Literal
	p.nextToken()
	return alias
}

func (p *Parser) parseDataType() string {
	dataType := p.curToken.Literal
	p.nextToken()

	if p.curTokenIs(token.LPAREN) {
		p.nextToken()
		var args []string
		for !p.isEOF() && !p.curTokenIs(token.RPAREN) {
			args = append(args, p.curToken.Literal)
			p.nextToken()
			if p.curTokenIs(token.COMMA) {
				args = append(args, ",")
				p.nextToken()
			}
		}
		if p.curTokenIs(token.RPAREN) {
			p.nextToken()
		}
		dataType = dataType + "(" + strings.Join(args, "") + ")"
	}

	return dataType
}
