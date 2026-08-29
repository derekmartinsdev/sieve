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
	cur    token.Token
	peek   token.Token
	errors []string
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{l: l, errors: []string{}}
	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) Errors() []string { return p.errors }

func (p *Parser) nextToken() {
	p.cur = p.peek
	p.peek = p.l.NextToken()
}

func (p *Parser) curTokenIs(t token.TokenType) bool {
	return p.cur.Type == t
}

func (p *Parser) peekTokenIs(t token.TokenType) bool {
	return p.peek.Type == t
}

func (p *Parser) expectPeek(t token.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.errors = append(p.errors, fmt.Sprintf("expected %s got %s (line %d)", t, p.peek.Type, p.peek.Line))
	return false
}

func (p *Parser) ParseProgram() *ast.Program {
	prog := &ast.Program{Statements: []ast.Statement{}}

	for !p.curTokenIs(token.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			prog.Statements = append(prog.Statements, stmt)
		}
		p.nextToken()
	}
	return prog
}

func (p *Parser) parseStatement() ast.Statement {
	switch p.cur.Type {
	case token.POSITION, token.TRADE, token.PERACQUISITION, token.TAXES:
		return p.parseSourceStatement()
	case token.IDENT:
		return p.parseAssignmentOrIdent()
	default:
		if isSourceKeyword(p.cur.Type) {
			return p.parseSourceStatement()
		}
		return nil
	}
}

func isSourceKeyword(t token.TokenType) bool {
	return t == token.POSITION || t == token.TRADE || t == token.PERACQUISITION || t == token.TAXES
}

func (p *Parser) parseSourceStatement() *ast.SourceStatement {
	stmt := &ast.SourceStatement{
		Token: p.cur.Literal,
		Name:  p.cur.Literal,
	}

	if !p.expectPeek(token.FROM) {
		return nil
	}
	p.nextToken()

	if p.curTokenIs(token.S3) {
		stmt.S3 = p.parseS3Source()
	} else if p.curTokenIs(token.IDENT) {
		stmt.Name = p.cur.Literal
	}

	for p.peekTokenIs(token.EXTRACT) {
		p.nextToken()
		extract := p.parseExtract()
		if extract != nil {
			stmt.Extracts = append(stmt.Extracts, *extract)
		}
	}

	return stmt
}

func (p *Parser) parseS3Source() *ast.S3Source {
	s3 := &ast.S3Source{}

	for !p.peekTokenIs(token.EXTRACT) && !p.peekTokenIs(token.EOF) && !p.peekTokenIs(token.IDENT) {
		p.nextToken()
		switch p.cur.Type {
		case token.BUCKET:
			if p.expectPeek(token.STRING) || p.peekTokenIs(token.IDENT) {
				p.nextToken()
				s3.Bucket = p.cur.Literal
			}
		case token.REGION:
			if p.expectPeek(token.STRING) || p.peekTokenIs(token.IDENT) {
				p.nextToken()
				s3.Region = p.cur.Literal
			}
		case token.PREFIX:
			if p.expectPeek(token.STRING) || p.peekTokenIs(token.IDENT) {
				p.nextToken()
				s3.Prefix = p.cur.Literal
			}
		case token.FORMAT:
			p.nextToken()
			s3.Format = p.cur.Literal
		default:
		}
	}

	return s3
}

func (p *Parser) parseExtract() *ast.ExtractStatement {
	ext := &ast.ExtractStatement{Token: "extract"}

	if !p.expectPeek(token.JSON) {
		return nil
	}
	p.nextToken()

	if p.curTokenIs(token.SELECT) || p.curTokenIs(token.EXTRACT) {
		ext.JsonType = p.cur.Literal
		p.nextToken()
	} else if p.curTokenIs(token.EXPLODE) {
		ext.JsonType = "explode"
		ext.IsArray = true
		p.nextToken()
	}

	if p.curTokenIs(token.IDENT) || p.curTokenIs(token.STRING) {
		ext.Path = p.cur.Literal
		p.nextToken()
	}

	if p.curTokenIs(token.ARRAY) {
		ext.IsArray = true
		p.nextToken()
	}

	for p.curTokenIs(token.IDENT) || p.curTokenIs(token.ASTERISK) {
		if p.peekTokenIs(token.IDENT) {
			field := p.parseFieldMapping()
			ext.Fields = append(ext.Fields, field)
		} else {
			break
		}
		p.nextToken()
	}

	return ext
}

func (p *Parser) parseFieldMapping() ast.FieldMapping {
	field := ast.FieldMapping{}

	field.SourcePath = p.cur.Literal
	p.nextToken()

	for p.curTokenIs(token.DOT) {
		field.SourcePath += "."
		p.nextToken()
		field.SourcePath += p.cur.Literal
		p.nextToken()
	}

	if p.curTokenIs(token.ASTERISK) {
		left := &ast.Identifier{Value: field.SourcePath}
		op := p.cur.Literal
		p.nextToken()
		right := &ast.Identifier{Value: p.cur.Literal}
		field.Expression = &ast.BinaryExpression{
			Left: left, Operator: op, Right: right,
		}
		field.SourcePath = field.SourcePath + " * " + p.cur.Literal
		p.nextToken()
	}

	if p.curTokenIs(token.IDENT) {
		field.Alias = p.cur.Literal
		p.nextToken()
	} else {
		field.Alias = field.SourcePath
	}

	field.Type = p.parseFieldType()

	return field
}

func (p *Parser) parseFieldType() ast.FieldType {
	ft := ast.FieldType{}

	if !p.curTokenIs(token.STRING_TYPE) && !p.curTokenIs(token.BIGINT) &&
		!p.curTokenIs(token.DATE) && !p.curTokenIs(token.DECIMAL_TYPE) &&
		!p.curTokenIs(token.IDENT) {
		return ft
	}

	ft.BaseType = p.cur.Literal
	p.nextToken()

	if p.curTokenIs(token.LPAREN) {
		p.nextToken()
		params := []string{}
		params = append(params, p.cur.Literal)
		p.nextToken()
		for p.curTokenIs(token.COMMA) {
			p.nextToken()
			params = append(params, p.cur.Literal)
			p.nextToken()
		}
		if p.curTokenIs(token.RPAREN) {
			p.nextToken()
		}
		ft.Params = params
	}

	return ft
}

func (p *Parser) parseAssignmentOrIdent() ast.Statement {
	name := p.cur.Literal
	if !p.peekTokenIs(token.ASSIGN) {
		return nil
	}
	p.nextToken()
	p.nextToken()

	return p.parseAssignment(name)
}

func (p *Parser) parseAssignment(name string) ast.Statement {
	stmt := &ast.AssignmentStatement{
		Token: name,
		Name:  name,
	}

	leftName := p.cur.Literal
	if p.peekTokenIs(token.JOIN) {
		p.nextToken()
		join := &ast.JoinStatement{Left: leftName}
		p.nextToken()
		join.Right = p.cur.Literal

		if p.peekTokenIs(token.LPAREN) {
			p.nextToken()
			cond := ""
			p.nextToken()
			for !p.curTokenIs(token.RPAREN) && !p.curTokenIs(token.EOF) {
				cond += p.cur.Literal + " "
				p.nextToken()
			}
			join.Condition = strings.TrimSpace(cond)
			p.nextToken()

			if p.curTokenIs(token.COMMA) {
				p.nextToken()
				if p.curTokenIs(token.LEFT) {
					join.JoinType = "left"
					p.nextToken()
				}
			}
		} else if p.peekTokenIs(token.ARROW) {
			p.nextToken()
			p.nextToken()
			cond := ""
			for !p.curTokenIs(token.EOF) && !p.curTokenIs(token.IDENT) && p.cur.Type != token.SELECT {
				cond += p.cur.Literal + " "
				p.nextToken()
			}
			join.Condition = strings.TrimSpace(cond)
		}

		if p.curTokenIs(token.SELECT) || p.peekTokenIs(token.SELECT) {
			if p.curTokenIs(token.SELECT) == false {
				p.nextToken()
			}
			join.Selects = p.parseSelectFields()
		}

		stmt.Value = join
	}

	return stmt
}

func (p *Parser) parseSelectFields() []ast.SelectField {
	fields := []ast.SelectField{}

	p.nextToken()
	for p.curTokenIs(token.IDENT) || p.cur.Literal == "." {
		field := ast.SelectField{}

		field.Source = p.cur.Literal

		for p.peekTokenIs(token.DOT) {
			p.nextToken()
			p.nextToken()
			field.Path += field.Source + "."
			field.Source = p.cur.Literal
		}

		if p.peekTokenIs(token.IDENT) && p.cur.Type == token.IDENT {
			field.Alias = field.Source
			field.Path = ""
			field.Source = ""

			if p.peekTokenIs(token.DOT) {
				p.nextToken()
				p.nextToken()
				source := field.Alias
				path := p.cur.Literal
				field.Source = source
				field.Path = path
				p.nextToken()
				for p.curTokenIs(token.DOT) {
					p.nextToken()
					field.Path += "." + p.cur.Literal
					p.nextToken()
				}
			}
		}

		if p.peekTokenIs(token.COMMA) {
			p.nextToken()
		}

		fields = append(fields, field)
		p.nextToken()
	}

	return fields
}