package parser

import (
	"github.com/derekmartinsdev/sieve/ast"
	"github.com/derekmartinsdev/sieve/token"
)

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
