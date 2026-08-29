package lexer

import (
	"github.com/derekmartinsdev/sieve/token"
)

type Lexer struct {
	input        string
	position     int
	readPosition int
	ch           byte
	line         int
	column       int
}

func New(input string) *Lexer {
	l := &Lexer{input: input, line: 1, column: 0}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++
	l.column++
}

func (l *Lexer) NextToken() token.Token {
	var tok token.Token

	l.skipWhitespace()
	l.skipComments()

	switch l.ch {
	case '|':
		tok = l.makeToken(token.PIPE, "|")
	case '-':
		if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			lit := string(ch) + string(l.ch)
			tok = l.makeToken(token.ARROW, lit)
		} else {
			tok = l.makeToken(token.MINUS, "-")
		}
	case '=':
		if l.peekChar() == '>' {
			l.readChar()
			tok = l.makeToken(token.ARROW, "=>")
		} else {
			tok = l.makeToken(token.ASSIGN, "=")
		}
	case ',':
		tok = l.makeToken(token.COMMA, ",")
	case '.':
		tok = l.makeToken(token.DOT, ".")
	case '(':
		tok = l.makeToken(token.LPAREN, "(")
	case ')':
		tok = l.makeToken(token.RPAREN, ")")
	case '[':
		tok = l.makeToken(token.LBRACKET, "[")
	case ']':
		tok = l.makeToken(token.RBRACKET, "]")
	case ':':
		if l.peekChar() == '=' {
			l.readChar()
			tok = l.makeToken(token.ASSIGN, ":=")
		} else {
			tok = l.makeToken(token.COLON, ":")
		}
	case '+':
		tok = l.makeToken(token.PLUS, "+")
	case '*':
		tok = l.makeToken(token.ASTERISK, "*")
	case '/':
		if l.peekChar() == '\\' {
			l.readChar()
			tok = l.makeToken(token.JOIN, "/\\")
		} else {
			tok = l.makeToken(token.SLASH, "/")
		}
	case '"':
		tok.Type = token.STRING
		tok.Literal = l.readString()
	case 0:
		tok = l.makeToken(token.EOF, "")
	default:
		if isLetter(l.ch) {
			lit := l.readIdentifier()
			tok.Type = token.LookupIdent(lit)
			tok.Literal = lit
			return tok
		} else if isDigit(l.ch) {
			lit := l.readNumber()
			tok.Type = token.NUMBER
			tok.Literal = lit
			return tok
		} else {
			tok = l.makeToken(token.ILLEGAL, string(l.ch))
		}
	}

	l.readChar()
	return tok
}

func (l *Lexer) makeToken(t token.TokenType, lit string) token.Token {
	return token.Token{Type: t, Literal: lit, Line: l.line, Column: l.column}
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		if l.ch == '\n' {
			l.line++
			l.column = 0
		}
		l.readChar()
	}
}

func (l *Lexer) skipComments() {
	for l.ch == '/' && l.peekChar() == '/' {
		for l.ch != '\n' && l.ch != 0 {
			l.readChar()
		}
		l.skipWhitespace()
	}
}

func (l *Lexer) readIdentifier() string {
	pos := l.position
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[pos:l.position]
}

func (l *Lexer) readNumber() string {
	pos := l.position
	for isDigit(l.ch) || l.ch == '.' {
		l.readChar()
	}
	return l.input[pos:l.position]
}

func (l *Lexer) readString() string {
	l.readChar()
	pos := l.position
	for l.ch != '"' && l.ch != 0 {
		if l.ch == '\\' {
			l.readChar()
		}
		l.readChar()
	}
	str := l.input[pos:l.position]
	return str
}

func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_' || ch > 127
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}