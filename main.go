package main

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"unicode"
)

type tokenKind = string

const (
	ident tokenKind = "ident"

	lparen = "("
	rparen = ")"
	lbrace = "{"
	rbrace = "}"
	dot    = "."
	comma  = ","
	semi   = ";"

	// arithmetic
	plus  = "+"
	minus = "-"
	mult  = "*"
	div   = "/"

	// comparison
	eqeq = "=="
	ne   = "!="
	gt   = ">"
	gte  = ">="
	lt   = "<"
	lte  = "<="

	// logical
	lnot = "!"
	land = "&&"
	lor  = "||"

	// assignment
	eq = "="

	str = "str"
	num = "num"
	eof = "eof"
)

type token struct {
	kind   tokenKind
	lexeme string
	row    int
	col    int
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

func main() {
	var (
		src       = getSource() + "\n"
		tokens    []token
		row       = 0
		col       = -1
		addLexeme = func(kind tokenKind, lexeme string) {
			tokens = append(tokens, token{kind, lexeme, row, col - len(lexeme)})
		}
		addToken = func(kind tokenKind) {
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
				addLexeme(str, literalBuf.String())
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
				addLexeme(num, literalBuf.String())
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
				addLexeme(num, literalBuf.String())
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
				addLexeme(ident, literalBuf.String())
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
			addToken(lparen)
		case ')':
			addToken(rparen)
		case '{':
			addToken(lbrace)
		case '}':
			addToken(rbrace)
		case '.':
			addToken(dot)
		case ',':
			addToken(comma)
		case ';':
			addToken(semi)
		case '+':
			addToken(plus)
		case '-':
			addToken(minus)
		case '*':
			addToken(mult)
		case '/':
			if src[i+1] == '/' {
				state = consumingComment
				i++
				col++
			} else {
				addToken(div)
			}
		case '=':
			if src[i+1] == '=' {
				addToken(eqeq)
				i++
				col++
			} else {
				addToken(eq)
			}
		case '!':
			if src[i+1] == '=' {
				addToken(ne)
				i++
				col++
			} else {
				addToken(lnot)
			}
		case '>':
			if src[i+1] == '=' {
				addToken(gte)
				i++
				col++
			} else {
				addToken(gt)
			}
		case '<':
			if src[i+1] == '=' {
				addToken(lte)
				i++
				col++
			} else {
				addToken(lt)
			}
		case '&':
			if src[i+1] == '&' {
				addToken(land)
				i++
				col++
			} else {
				panic("Bitwise AND is NYI")
			}
		case '|':
			if src[i+1] == '|' {
				addToken(lor)
				i++
				col++
			} else {
				panic("Bitwise OR is NYI")
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
	addToken(eof)

	for _, t := range tokens {
		fmt.Printf("%s\t%q\n", t.kind, t.lexeme)
	}

	var stmts []interface{}
	for i := 0; i < len(tokens); {
		if tokens[i].kind == eof {
			break
		}
		stmt, adv := consumeTopLevelStmt(tokens[i:])
		stmts = append(stmts, stmt)
		i += adv
	}
	for _, stmt := range stmts {
		fmt.Printf("%+v\n", stmt)
	}
}

func consumePrimaryExpr(tokens []token) (interface{}, int) {
	t0 := tokens[0]
	if t0.kind == str {
		return literalExpr{strLiteral, t0.lexeme}, 1
	}
	if t0.kind == num {
		return literalExpr{numLiteral, t0.lexeme}, 1
	}
	if t0.lexeme == "false" {
		return literalExpr{boolLiteral, t0.lexeme}, 1
	}
	if t0.lexeme == "true" {
		return literalExpr{boolLiteral, t0.lexeme}, 1
	}
	if t0.lexeme == "null" {
		return literalExpr{nullLiteral, t0.lexeme}, 1
	}
	if t0.kind == lparen {
		expr, adv := consumeExpr(tokens[1:])
		if tokens[1+adv].kind != rparen {
			panic("Expected ')' after expression")
		}
		return groupExpr{expr}, 2 + adv
	}
	return literalExpr{varLiteral, t0.lexeme}, 1
}

func consumeCallExpr(tokens []token) (interface{}, int) {
	i := 0
	expr, adv := consumePrimaryExpr(tokens)
	i += adv
	var args []interface{}
	for {
		if tokens[i].kind == lparen {
			i++
			for {
				if tokens[i].kind == rparen {
					i++
					expr = callExpr{expr, args}
					args = []interface{}{}
					break
				}
				argExpr, adv2 := consumeExpr(tokens[i:])
				args = append(args, argExpr)
				i += adv2
				if tokens[i].kind == rparen {
					i++
					expr = callExpr{expr, args}
					args = []interface{}{}
					break
				} else if tokens[i].kind != comma {
					panic("Required command between call args")
				}
				i++
			}
		} else if tokens[i].kind == dot {
			i++
			expr = memberExpr{expr, tokens[i].lexeme}
			i++
		} else {
			break
		}
	}
	return expr, i
}

func consumeUnaryExpr(tokens []token) (interface{}, int) {
	if tokens[0].kind == lnot || tokens[1].kind == minus {
		expr, adv := consumeUnaryExpr(tokens[1:])
		return unaryOp{tokens[0].kind, expr}, adv + 1
	}
	return consumeCallExpr(tokens)
}

func consumeFactorExpr(tokens []token) (interface{}, int) {
	i := 0
	expr, adv1 := consumeUnaryExpr(tokens)
	i += adv1
	for op := tokens[i].kind; op == mult || op == div; op = tokens[i].kind {
		expr2, adv2 := consumeUnaryExpr(tokens[i+1:])
		i += adv2 + 1
		expr = binaryOp{expr, op, expr2}
	}
	return expr, i
}

func consumeTermExpr(tokens []token) (interface{}, int) {
	i := 0
	expr, adv1 := consumeFactorExpr(tokens)
	i += adv1
	for op := tokens[i].kind; op == plus || op == minus; op = tokens[i].kind {
		expr2, adv2 := consumeTermExpr(tokens[i+1:])
		i += adv2 + 1
		expr = binaryOp{expr, op, expr2}
	}
	return expr, i
}

func consumeComparisonExpr(tokens []token) (interface{}, int) {
	i := 0
	expr, adv1 := consumeTermExpr(tokens)
	i += adv1
	for op := tokens[i].kind; op == plus || op == minus; op = tokens[i].kind {
		expr2, adv2 := consumeTermExpr(tokens[i+1:])
		i += adv2 + 1
		expr = binaryOp{expr, op, expr2}
	}
	return expr, i
}

func consumeEqualityExpr(tokens []token) (interface{}, int) {
	i := 0
	expr, adv1 := consumeComparisonExpr(tokens)
	i += adv1
	for op := tokens[i].kind; op == ne || op == eq; op = tokens[i].kind {
		expr2, adv2 := consumeComparisonExpr(tokens[i+1:])
		i += adv2 + 1
		expr = binaryOp{expr, op, expr2}
	}
	return expr, i
}

func consumeExpr(tokens []token) (interface{}, int) {
	return consumeEqualityExpr(tokens)
}

func consumeStmt(tokens []token) (interface{}, int) {
	t0 := tokens[0]
	t1 := tokens[1]
	if t0.lexeme == "return" {
		expr, adv := consumeExpr(tokens[1:])
		if tokens[1+adv].kind != semi {
			panic("Expected ';' after statement")
		}
		return returnStmt{expr}, adv + 2
	}
	if t1.kind == eq {
		expr, adv := consumeExpr(tokens[2:])
		if tokens[2+adv].kind != semi {
			panic("Expected ';' after statement")
		}
		return assignmentStmt{t0, expr}, adv + 3
	}
	t2 := tokens[2]
	if t2.kind == eq {
		expr, adv := consumeExpr(tokens[3:])
		if tokens[3+adv].kind != semi {
			panic("Expected ';' after statement")
		}
		return varStmt{t0, t1, expr}, adv + 4
	}
	cexpr, adv3 := consumeCallExpr(tokens)
	if tokens[adv3].kind != semi {
		panic("Expected ';' after statement")
	}
	return cexpr, adv3 + 1
}

func consumeBlock(tokens []token) (block, int) {
	var stmts []interface{}
	for i := 0; i < len(tokens); {
		t := tokens[i]
		if t.kind == rbrace {
			return block{stmts}, i + 1
		}
		stmt, adv := consumeStmt(tokens[i:])
		stmts = append(stmts, stmt)
		i += adv
	}
	panic("Unclosed block")
}

func consumeFunction(tokens []token) (functionStmt, int) {
	retKind := tokens[0]
	name := tokens[1]
	if tokens[2].kind != lparen {
		panic("Expected '(' at " + strconv.Itoa(tokens[2].row) + "," + strconv.Itoa(tokens[2].col))
	}
	var params []functionParam
	i := 3
	for ; i < len(tokens); i++ {
		t := tokens[i]
		if t.kind == rparen {
			i++
			break
		}
		params = append(params, functionParam{tokens[i], tokens[i+1]})
		i += 2
		if tokens[i].kind == rparen {
			i++
			break
		} else if tokens[i].kind != comma {
			panic("Required ',' or ')' after function param")
		}
	}
	b, adv := consumeBlock(tokens[i+1:])
	return functionStmt{retKind, name, params, b}, i + adv + 1
}

func consumeTopLevelStmt(tokens []token) (interface{}, int) {
	return consumeFunction(tokens)
}

type groupExpr struct {
	expr interface{}
}

type literalKind = int

const (
	strLiteral literalKind = iota
	numLiteral
	boolLiteral
	nullLiteral
	varLiteral
)

type literalExpr struct {
	kind   literalKind
	lexeme string
}

type callExpr struct {
	callee interface{}
	args   []interface{}
}

type memberExpr struct {
	parent     interface{}
	memberName string
}

type returnStmt struct {
	expr interface{}
}

type functionParam struct {
	kind token
	name token
}

type functionStmt struct {
	retKind token
	name    token
	params  []functionParam
	body    block
}

type varStmt struct {
	kind token
	name token
	expr interface{}
}

type assignmentStmt struct {
	name token
	expr interface{}
}

type whileStmt struct {
	conditionExpr interface{}
	body          block
}

type block struct {
	stmts []interface{}
}

type unaryOp struct {
	op   tokenKind
	expr interface{}
}

type binaryOp struct {
	leftExpr  interface{}
	op        tokenKind
	rightExpr interface{}
}

func getSource() string {
	b, err := ioutil.ReadFile("test.c")
	if err != nil {
		panic(err)
	}
	return string(b)
}
