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

	for !p.isEOF() && !p.isSectionStart() && !p.isBodyKeyword() {
		if p.curTokenIs(token.IDENT) {
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

			alias := p.parseAlias()

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

func (p *Parser) parseSelect() *ast.SelectStmt {
	p.nextToken()

	stmt := &ast.SelectStmt{}
	stmt.Fields = p.parseSelectFields()

	return stmt
}

func (p *Parser) parseSelectFields() []ast.SelectField {
	var fields []ast.SelectField

	for !p.isEOF() && !p.isSectionStart() && !p.isBodyKeyword() {
		if p.curTokenIs(token.COMMA) {
			p.nextToken()
			continue
		}
		if !p.isIdentOrDateFunc() {
			break
		}

		path := p.parseDottedPath()

		alias := ""
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
		}

		alias = p.parseAlias()

		if p.curTokenIs(token.COMMA) {
			p.nextToken()
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

func (p *Parser) parseAlias() string {
	if !p.isIdentOrDateFunc() || p.isTypeKeyword(p.curToken.Type) || p.peekTokenIs(token.DOT) || p.curToken.Column <= 1 {
		return ""
	}
	if isDateFunc(p.peekToken) || p.isTypeKeyword(p.peekToken.Type) {
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

func (p *Parser) parseSink() *ast.Sink {
	sink := &ast.Sink{}

	p.nextToken()

	if !p.curTokenIs(token.S3) {
		p.error("'to' must be followed by 's3'. Did you mean 'to s3'?")
		return sink
	}

	sink.To = "s3"
	p.nextToken()

	for !p.isEOF() && !p.isSectionStart() && !p.isBodyKeyword() {
		switch p.curToken.Type {
		case token.BUCKET:
			p.nextToken()
			sink.Bucket = p.curToken.Literal
			p.nextToken()
		case token.REGION:
			p.nextToken()
			sink.Region = p.curToken.Literal
			p.nextToken()
		case token.PREFIX:
			p.nextToken()
			sink.Prefix = p.curToken.Literal
			p.nextToken()
		case token.FORMAT:
			p.nextToken()
			sink.Format = p.curToken.Literal
			p.nextToken()
		case token.MODE:
			p.nextToken()
			sink.Mode = p.curToken.Literal
			p.nextToken()
		case token.PARTITIONED:
			p.nextToken()
			if p.curTokenIs(token.BY) {
				p.nextToken()
				for !p.isEOF() && !p.isSectionStart() && !p.isBodyKeyword() {
					sink.PartitionBy = append(sink.PartitionBy, p.curToken.Literal)
					p.nextToken()
					if p.curTokenIs(token.COMMA) {
						p.nextToken()
					} else {
						break
					}
				}
			}
		default:
			p.nextToken()
		}
	}

	return sink
}

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
