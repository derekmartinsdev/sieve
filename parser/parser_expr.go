package parser

import (
	"strings"

	"github.com/derekmartinsdev/sieve/ast"
	"github.com/derekmartinsdev/sieve/token"
)
func (p *Parser) parseSelect() *ast.SelectStmt {
	p.nextToken()

	stmt := &ast.SelectStmt{}
	stmt.Fields = p.parseSelectFields()

	return stmt
}

func (p *Parser) parseSelectFields() []ast.SelectField {
	var fields []ast.SelectField
	justComma := false

	for !p.isEOF() && !p.isSectionStart() && !p.isBodyKeyword() {
		if p.curTokenIs(token.COMMA) {
			p.nextToken()
			justComma = true
			continue
		}
		if !p.isIdentOrDateFunc() {
			break
		}

		pathLine := p.curToken.Line
		path := p.parseDottedPath()

		function := ""

		if isDateFunc(p.curToken) {
			function = p.curToken.Literal
			p.nextToken()
			if p.curTokenIs(token.LPAREN) {
				p.nextToken()
				if p.curTokenIs(token.RPAREN) {
					p.nextToken()
				}
			}
		}

		if p.curTokenIs(token.COMMA) {
			p.nextToken()
			justComma = true
		}

		alias := ""
		if !justComma {
			alias = p.parseAlias(pathLine)
		}
		justComma = false

		if p.curTokenIs(token.COMMA) {
			p.nextToken()
			justComma = true
		}

		dataType := ""
		if p.isTypeKeyword(p.curToken.Type) {
			dataType = p.parseDataType()
		}

		if p.curTokenIs(token.OR) {
			p.nextToken()
			if isDateFunc(p.curToken) {
				p.nextToken()
				if p.curTokenIs(token.LPAREN) {
					p.nextToken()
					if p.curTokenIs(token.RPAREN) {
						p.nextToken()
					}
				}
			}
		}

		if alias == "" {
			if function != "" {
				alias = function
			} else {
				parts := strings.Split(path, ".")
				alias = parts[len(parts)-1]
			}
		}

		fields = append(fields, ast.SelectField{
			Name:     path,
			Alias:    alias,
			DataType: dataType,
			Function: function,
		})
	}

	return fields
}
func isTransformKeyword(t token.TokenType) bool {
	switch t {
	case token.CAST, token.DEFAULT_KW, token.REPLACE, token.PREFIX, token.HASH:
		return true
	}
	return false
}

func (p *Parser) parseTransformBlock() []ast.TransformChain {
	p.nextToken()

	var transforms []ast.TransformChain

	for !p.isEOF() && !p.isSectionStart() &&
		!p.curTokenIs(token.SELECT) &&
		!p.curTokenIs(token.TO) &&
		!p.curTokenIs(token.FROM) &&
		!p.curTokenIs(token.EXTRACT) &&
		!p.curTokenIs(token.ASSIGN) {

		if p.curTokenIs(token.IDENT) {
			alias := p.curToken.Literal
			p.nextToken()

			if !p.curTokenIs(token.ASSIGN) {
				continue
			}
			p.nextToken()

			sourcePath := p.parseDottedPath()

			chain := ast.TransformChain{Alias: alias}
			chain.Steps = append(chain.Steps, ast.TransformStep{
				Type: "col",
				Args: []string{sourcePath},
			})

			if p.isTypeKeyword(p.curToken.Type) {
				chain.Steps = append(chain.Steps, ast.TransformStep{
					Type: "cast",
					Args: []string{p.parseDataType()},
				})
			}

			for p.curTokenIs(token.PIPE) {
				p.nextToken()

				if !p.curTokenIs(token.IDENT) && !isTransformKeyword(p.curToken.Type) {
					break
				}
				stepType := p.curToken.Literal
				p.nextToken()

				var args []string
				if p.curTokenIs(token.LPAREN) {
					args = p.parseArgsList()
				} else if p.isTypeKeyword(p.curToken.Type) {
					args = []string{p.parseDataType()}
				}

				if p.isTypeKeyword(p.curToken.Type) {
					p.parseDataType()
				}

				chain.Steps = append(chain.Steps, ast.TransformStep{
					Type: stepType,
					Args: args,
				})
			}

			transforms = append(transforms, chain)
		} else {
			break
		}
	}

	return transforms
}

func (p *Parser) parseArgsList() []string {
	var args []string
	p.nextToken()

	for !p.isEOF() && !p.curTokenIs(token.RPAREN) {
		args = append(args, p.curToken.Literal)
		p.nextToken()
		if p.curTokenIs(token.COMMA) {
			p.nextToken()
		}
	}

	if p.curTokenIs(token.RPAREN) {
		p.nextToken()
	}

	return args
}
