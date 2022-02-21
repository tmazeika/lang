package scanner

import (
	"io/ioutil"
	"strconv"
	"strings"
	"unicode"
)

//go:generate stringer -type=TokenKind
type TokenKind int

const (
	Ident TokenKind = iota

	LParen
	RParen
	LBrace
	RBrace
	Dot
	Comma
	Semicolon

	Plus
	Minus
	Star
	Slash

	EqEq
	Ne
	Gt
	Gte
	Lt
	Lte

	LNot
	LAnd
	LOr

	Eq

	Str
	Num
	Eof
)

type Token struct {
	Kind   TokenKind
	Lexeme string
	Row    int
	Col    int
}

type consumeState int

const (
	none consumeState = iota
	consumingComment
	consumingStr
	consumingStrEscape
	consumingWholeNum
	consumingFracNum
	consumingIdent
)

func Scan() []Token {
	var (
		src       = getSource() + "\n"
		tokens    []Token
		row       = 0
		col       = -1
		addLexeme = func(kind TokenKind, lexeme string) {
			tokens = append(tokens, Token{kind, lexeme, row, col - len(lexeme)})
		}
		addToken = func(kind TokenKind) {
			addLexeme(kind, "")
		}
		state      = none
		literalBuf = strings.Builder{}
	)

	for i := 0; i < len(src)-1; i++ {
		col++
		ch := rune(src[i])

		switch state {
		case consumingComment:
			if ch == '\n' {
				row++
				col = -1
				state = none
			} else {
				continue
			}
		case consumingStr:
			if ch == '"' {
				addLexeme(Str, literalBuf.String())
				literalBuf.Reset()
				state = none
				continue
			} else if ch == '\n' {
				panic("Unclosed string")
			} else if ch == '\\' {
				state = consumingStrEscape
				continue
			} else {
				literalBuf.WriteRune(ch)
				continue
			}
		case consumingStrEscape:
			switch ch {
			case '"':
				literalBuf.WriteRune('"')
			case 'r':
				literalBuf.WriteRune('\r')
			case 't':
				literalBuf.WriteRune('\t')
			case 'n':
				literalBuf.WriteRune('\n')
			default:
				panic("unknown escape sequence \\" + string(ch))
			}
			state = consumingStr
			continue
		case consumingWholeNum:
			if unicode.IsDigit(ch) {
				literalBuf.WriteRune(ch)
				continue
			} else if ch == '.' {
				literalBuf.WriteRune(ch)
				state = consumingFracNum
				continue
			} else {
				addLexeme(Num, literalBuf.String())
				literalBuf.Reset()
				i--
				col--
				state = none
				continue
			}
		case consumingFracNum:
			if unicode.IsDigit(ch) {
				literalBuf.WriteRune(ch)
				continue
			} else {
				addLexeme(Num, literalBuf.String())
				literalBuf.Reset()
				i--
				col--
				state = none
				continue
			}
		case consumingIdent:
			if 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_' || '0' <= ch && ch <= '9' {
				literalBuf.WriteRune(ch)
				continue
			} else {
				addLexeme(Ident, literalBuf.String())
				literalBuf.Reset()
				i--
				col--
				state = none
				continue
			}
		}

		switch ch {
		case '\n':
			row++
			col = -1
		case ' ', '\r', '\t':
		case '(':
			addToken(LParen)
		case ')':
			addToken(RParen)
		case '{':
			addToken(LBrace)
		case '}':
			addToken(RBrace)
		case '.':
			addToken(Dot)
		case ',':
			addToken(Comma)
		case ';':
			addToken(Semicolon)
		case '+':
			addToken(Plus)
		case '-':
			addToken(Minus)
		case '*':
			addToken(Star)
		case '/':
			if src[i+1] == '/' {
				state = consumingComment
				i++
				col++
			} else {
				addToken(Slash)
			}
		case '=':
			if src[i+1] == '=' {
				addToken(EqEq)
				i++
				col++
			} else {
				addToken(Eq)
			}
		case '!':
			if src[i+1] == '=' {
				addToken(Ne)
				i++
				col++
			} else {
				addToken(LNot)
			}
		case '>':
			if src[i+1] == '=' {
				addToken(Gte)
				i++
				col++
			} else {
				addToken(Gt)
			}
		case '<':
			if src[i+1] == '=' {
				addToken(Lte)
				i++
				col++
			} else {
				addToken(Lt)
			}
		case '&':
			if src[i+1] == '&' {
				addToken(LAnd)
				i++
				col++
			} else {
				panic("Bitwise LAnd is NYI")
			}
		case '|':
			if src[i+1] == '|' {
				addToken(LOr)
				i++
				col++
			} else {
				panic("Bitwise LOr is NYI")
			}
		case '"':
			state = consumingStr
		default:
			if unicode.IsDigit(ch) {
				literalBuf.WriteRune(ch)
				state = consumingWholeNum
			} else if 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_' {
				literalBuf.WriteRune(ch)
				state = consumingIdent
			} else {
				panic("unknown char " + string(ch) + " at row " + strconv.Itoa(row) + ", col " + strconv.Itoa(col))
			}
		}
	}
	addToken(Eof)

	return tokens
}

func getSource() string {
	b, err := ioutil.ReadFile("test.c")
	if err != nil {
		panic(err)
	}
	return string(b)
}
