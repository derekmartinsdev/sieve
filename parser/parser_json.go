package parser

import (
	"fmt"
	"strings"

	"github.com/derekmartinsdev/sieve/ast"
	"github.com/derekmartinsdev/sieve/token"
)

func (p *Parser) parseExtract() *ast.Extract {
	ext := &ast.Extract{}
	p.nextToken()

	if !p.curTokenIs(token.JSON) {
		p.error("'extract' must be followed by 'json'. Did you mean 'extract json select ...' or 'extract json explode ...'?")
		return ext
	}

	for p.curTokenIs(token.JSON) {
		p.nextToken()

		switch p.curToken.Type {
		case token.SELECT:
			js := p.parseJsonSelect()
			if js != nil {
				ext.JsonSelect = js.JsonSelect
			}
		case token.EXPLODE:
			ex := p.parseExplode()
			if ex != nil {
				ext.Explode = ex.Explode
			}
		case token.EXTRACT:
			je := p.parseJsonExtract()
			if je != nil {
				ext.JsonExtract = je.JsonExtract
			}
		default:
			p.error(fmt.Sprintf("after 'json' expected 'select', 'explode', or 'extract', got %q. Did you mean 'json select message' or 'json explode path array'?", p.curToken.Literal))
			return ext
		}
	}

	return ext
}

func (p *Parser) parseJsonSelect() *ast.Extract {
	p.nextToken()

	js := &ast.JsonSelect{
		Path: p.curToken.Literal,
	}

	p.nextToken()
	js.Fields = p.parseExtractFields(js.Path)

	return &ast.Extract{JsonSelect: js}
}

func (p *Parser) parseExplode() *ast.Extract {
	p.nextToken()

	path := p.parseDottedPath()

	if !p.curTokenIs(token.ARRAY) {
		p.error("expected 'array' after json explode path. Did you mean 'json explode message.items array'?")
		return nil
	}
	p.nextToken()

	explode := &ast.ExtractExplode{
		Path:   path,
		As:     "array",
		Fields: p.parseExtractFields("exploded"),
	}

	return &ast.Extract{Explode: explode}
}

func (p *Parser) parseJsonExtract() *ast.Extract {
	p.nextToken()

	path := p.parseDottedPath()

	if !p.curTokenIs(token.ARRAY) {
		p.error("expected 'array' after json extract path. Did you mean 'json extract message.items array'?")
		return nil
	}
	p.nextToken()

	je := &ast.JsonExtract{
		Path:   path,
		As:     "array",
		Fields: p.parseExtractFields("exploded"),
	}

	return &ast.Extract{JsonExtract: je}
}

func (p *Parser) parseExtractFields(jsonCol string) []*ast.FieldDef {
	var fields []*ast.FieldDef
	justComma := false

	for !p.isEOF() && !p.isSectionStart() && !p.isBodyKeyword() {
		if p.curTokenIs(token.COMMA) {
			p.nextToken()
			justComma = true
			continue
		}
		if p.curTokenIs(token.IDENT) {
			sourceLine := p.curToken.Line
			source := p.parseDottedPath()

			computedExpr := (*ast.BinaryExpr)(nil)
			if p.curTokenIs(token.ASTERISK) {
				left := source
				p.nextToken()
				right := p.curToken.Literal
				p.nextToken()
				computedExpr = &ast.BinaryExpr{Left: left, Operator: "*", Right: right}
				source = ""
			}

			alias := ""
			if !justComma {
				alias = p.parseAlias(sourceLine)
			}
			justComma = false

			dataType := ""
			if p.isTypeKeyword(p.curToken.Type) {
				dataType = p.parseDataType()
			}

			name := alias
			if name == "" {
				parts := strings.Split(source, ".")
				name = parts[len(parts)-1]
			}

			fields = append(fields, &ast.FieldDef{
				Name:         name,
				Source:       source,
				Alias:        alias,
				DataType:     dataType,
				ComputedExpr: computedExpr,
				JsonColumn:   jsonCol,
			})
		} else {
			break
		}
	}

	return fields
}
