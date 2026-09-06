package parser

import (
	"fmt"
	"strings"

	"github.com/derekmartinsdev/sieve/ast"
	"github.com/derekmartinsdev/sieve/token"
)

func (p *Parser) parseDerivedSection(sec *ast.Section) *ast.Section {
	join := &ast.Join{}

	p.nextToken()

	if !p.curTokenIs(token.IDENT) {
		p.error("expected left section name after '=', got " + p.curToken.Literal)
		return sec
	}
	join.Left = p.curToken.Literal
	join.LeftAlias = join.Left
	p.nextToken()

	if p.curTokenIs(token.LPAREN) {
		p.nextToken()
		if p.curTokenIs(token.IDENT) {
			join.LeftAlias = p.curToken.Literal
			p.nextToken()
		}
		if p.curTokenIs(token.RPAREN) {
			p.nextToken()
		}
	}

	if !p.curTokenIs(token.JOIN) {
		p.error(fmt.Sprintf("expected join operator '/\\' after '%s', got %q. Did you mean '%s /\\ other_section'?", join.Left, p.curToken.Literal, join.Left))
		return sec
	}
	p.nextToken()

	if !p.curTokenIs(token.IDENT) {
		p.error("expected right section name after '/\\', got " + p.curToken.Literal)
		return sec
	}
	join.Right = p.curToken.Literal
	join.RightAlias = join.Right
	p.nextToken()

	if p.curTokenIs(token.LPAREN) {
		p.nextToken()
		if p.curTokenIs(token.IDENT) {
			join.RightAlias = p.curToken.Literal
			p.nextToken()
		}
		if p.curTokenIs(token.RPAREN) {
			p.nextToken()
		}
	}

	if !p.curTokenIs(token.ARROW) {
		p.error(fmt.Sprintf("expected '->' after join right side, got %q. Did you mean '%s /\\ %s -> condition'?", p.curToken.Literal, join.Left, join.Right))
		return sec
	}
	p.nextToken()

	condition := &ast.JoinCondition{}

	if p.curTokenIs(token.LPAREN) {
		p.nextToken()
	}

	if p.curTokenIs(token.IDENT) {
		ref := p.parseDottedPath()
		parts := strings.Split(ref, ".")
		if parts[0] == join.Left && join.LeftAlias != join.Left {
			parts[0] = join.LeftAlias
			ref = strings.Join(parts, ".")
		}
		condition.LeftRefs = []string{ref}
	}

	if p.curTokenIs(token.ASSIGN) {
		p.nextToken()
	}

	if p.curTokenIs(token.IDENT) {
		ref := p.parseDottedPath()
		parts := strings.Split(ref, ".")
		if parts[0] == join.Right && join.RightAlias != join.Right {
			parts[0] = join.RightAlias
			ref = strings.Join(parts, ".")
		}
		condition.RightRefs = []string{ref}
	}

	if p.curTokenIs(token.RPAREN) {
		p.nextToken()
	}

	if p.curTokenIs(token.COMMA) {
		p.nextToken()
		switch p.curToken.Type {
		case token.LEFT, token.RIGHT, token.INNER:
			join.JoinType = p.curToken.Literal
			p.nextToken()
		}
	}

	join.Condition = condition
	sec.Joins = append(sec.Joins, join)

	for !p.curTokenIs(token.EOF) && !p.isSectionStart() {
		switch {
		case p.curTokenIs(token.SELECT):
			sec.Select = p.parseSelect()
		case p.curTokenIs(token.TO):
			sec.Sink = p.parseSink()
		case p.curTokenIs(token.TRANSFORM):
			sec.Transforms = p.parseTransformBlock()
		default:
			p.nextToken()
		}
	}

	return sec
}
