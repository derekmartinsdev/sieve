package lexer

import "github.com/derekmartinsdev/sieve/token"

type Lexer struct {
	input        []rune
	position     int
	readPosition int
	ch           rune
	line         int
	column       int
}

func New(input string) *Lexer {
	l := &Lexer{input: []rune(input), line: 1, column: 0}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
		l.column++
	}
	l.position = l.readPosition
	l.readPosition++
}

func (l *Lexer) peekChar() rune {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

func (l *Lexer) NextToken() token.Token {
	var tok token.Token

	l.skipWhitespace()
	l.skipComment()

	switch l.ch {
	case '=':
		tok = l.newToken(token.ASSIGN, "=")
	case '|':
		tok = l.newToken(token.PIPE, "|")
	case ',':
		tok = l.newToken(token.COMMA, ",")
	case '(':
		tok = l.newToken(token.LPAREN, "(")
	case ')':
		tok = l.newToken(token.RPAREN, ")")
	case ':':
		tok = l.newToken(":", ":")
	case '*':
		tok = l.newToken(token.ASTERISK, "*")
	case '.':
		if l.peekChar() == '.' {
			ch := l.ch
			l.readChar()
			tok = l.newToken(token.TYPE_DATE, string(ch)+string(l.ch))
		} else {
			tok = l.newToken(token.DOT, ".")
		}
	case '-':
		if l.peekChar() == '>' {
			l.readChar()
			tok = l.newToken(token.ARROW, "->")
		} else {
			tok = l.newToken(token.ILLEGAL, string(l.ch))
		}
	case '/':
		if l.peekChar() == '\\' {
			l.readChar()
			tok = l.newToken(token.JOIN, "/\\")
		} else {
			illegalChar := l.ch
			l.readChar()
			tok = l.newToken(token.ILLEGAL, string(illegalChar))
			return tok
		}
	case '"':
		tok.Type = token.STRING
		tok.Literal = l.readDelimitedString('"')
		tok.Line = l.line
		tok.Column = l.column
		return tok
	case '\'':
		tok.Type = token.STRING
		tok.Literal = l.readDelimitedString('\'')
		tok.Line = l.line
		tok.Column = l.column
		return tok
	case 0:
		tok.Literal = ""
		tok.Type = token.EOF
		tok.Line = l.line
		tok.Column = l.column
		return tok
	default:
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = token.LookupIdent(tok.Literal)
			tok.Line = l.line
			tok.Column = l.column - len(tok.Literal)
			return tok
		} else if isDigit(l.ch) || l.ch == '-' {
			num, isFloat, isDate := l.readNumber()
			if isDate {
				tok.Type = token.STRING
				tok.Literal = num
			} else if isFloat {
				tok.Type = token.FLOAT
				tok.Literal = num
			} else {
				tok.Type = token.INT
				tok.Literal = num
			}
			tok.Line = l.line
			tok.Column = l.column - len(tok.Literal)
			return tok
		} else {
			tok = l.newToken(token.ILLEGAL, string(l.ch))
		}
	}

	l.readChar()
	return tok
}

func (l *Lexer) newToken(tokenType token.TokenType, literal string) token.Token {
	return token.Token{Type: tokenType, Literal: literal, Line: l.line, Column: l.column}
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		switch l.ch {
		case '\n':
			l.line++
			l.column = 0
		case '\r':
			l.line++
			l.column = 0
			if l.peekChar() == '\n' {
				l.readChar()
			}
		case '\t':
			l.column += 3
		}
		l.readChar()
	}
}

func (l *Lexer) skipComment() {
	if l.ch == '/' && l.peekChar() == '/' {
		for l.ch != '\n' && l.ch != 0 {
			l.readChar()
		}
		l.skipWhitespace()
	}
}

func (l *Lexer) readIdentifier() string {
	start := l.position
	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '_' {
		l.readChar()
	}
	return string(l.input[start:l.position])
}

func (l *Lexer) readNumber() (string, bool, bool) {
	start := l.position
	isFloat := false
	isDate := false
	hasDash := false

	for isDigit(l.ch) || l.ch == '.' || l.ch == '-' {
		if l.ch == '.' {
			isFloat = true
		}
		if l.ch == '-' {
			if hasDash {
				return string(l.input[start:l.position]), isFloat, true
			}
			hasDash = true
		}
		l.readChar()
	}

	if isFloat && hasDash {
		isDate = true
		isFloat = false
	}

	return string(l.input[start:l.position]), isFloat, isDate
}

func (l *Lexer) readDelimitedString(delim rune) string {
	l.readChar()
	start := l.position
	for l.ch != delim && l.ch != 0 {
		if l.ch == '\\' {
			l.readChar()
		}
		l.readChar()
	}
	end := l.position
	if l.ch != 0 {
		l.readChar()
	}
	return string(l.input[start:end])
}

func isLetter(ch rune) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z'
}

func isDigit(ch rune) bool {
	return '0' <= ch && ch <= '9'
}
