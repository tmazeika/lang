package parser

import "lang/scanner"

type Parser struct {
	i      int
	Tokens []scanner.Token
}

func (p *Parser) peek() scanner.Token {
	if p.i >= len(p.Tokens) {
		panic("Reached unexpected EOF")
	}
	return p.Tokens[p.i]
}

func (p *Parser) peekN(n int) scanner.Token {
	p.i += n
	t := p.peek()
	p.i -= n
	return t
}

func (p *Parser) consumeOne() scanner.Token {
	t := p.peek()
	p.i++
	return t
}

func (p *Parser) consume(kind scanner.TokenKind, msg string) scanner.Token {
	t := p.consumeOne()
	if t.Kind != kind {
		panic(msg)
	}
	return t
}

func (p *Parser) consumeKeyword(keyword string, msg string) {
	if p.consume(scanner.Ident, msg).Lexeme != keyword {
		panic(msg)
	}
}

func (p *Parser) matchKeyword(keyword string) bool {
	if p.i >= len(p.Tokens) {
		return false
	}
	t := p.peek()
	return t.Kind == scanner.Ident && t.Lexeme == keyword
}

func (p *Parser) matchKeywordN(n int, keyword string) bool {
	p.i += n
	ok := p.matchKeyword(keyword)
	p.i -= n
	return ok
}

func (p *Parser) match(kinds ...scanner.TokenKind) bool {
	if p.i >= len(p.Tokens) {
		return false
	}
	t := p.peek()
	for _, kind := range kinds {
		if t.Kind == kind {
			return true
		}
	}
	return false
}

func (p *Parser) matchN(n int, kinds ...scanner.TokenKind) bool {
	p.i += n
	ok := p.match(kinds...)
	p.i -= n
	return ok
}

func (p *Parser) previous() scanner.Token {
	if p.i == 0 {
		panic("There is no previous token")
	}
	return p.Tokens[p.i-1]
}

func (p *Parser) consumeAtomExpr() expr {
	if p.matchKeyword("true") {
		p.consumeOne()
		return literalBool{true}
	} else if p.matchKeyword("false") {
		p.consumeOne()
		return literalBool{false}
	} else if p.matchKeyword("null") {
		p.consumeOne()
		return literalNull{}
	}
	t := p.peek()
	if t.Kind == scanner.Str {
		p.consumeOne()
		return literalStr{t.Lexeme}
	} else if t.Kind == scanner.Num {
		p.consumeOne()
		return literalNum{t.Lexeme}
	} else if t.Kind == scanner.LParen {
		return p.consumeGroupExpr()
	} else if t.Kind == scanner.Ident {
		p.consumeOne()
		return identExpr{t}
	} else {
		panic("Unknown atom")
	}
}

func (p *Parser) consumeCallExpr() expr {
	e := p.consumeAtomExpr()
	for {
		if p.match(scanner.Dot) {
			p.consumeOne()
			e = memberAccess{e, p.consume(scanner.Ident, "Expected member name")}
		} else if p.match(scanner.LParen) {
			p.consumeOne()
			var args []expr
			for !p.match(scanner.RParen) {
				arg := p.consumeExpr()
				args = append(args, arg)
				if p.match(scanner.Comma) {
					p.consumeOne()
				} else if !p.match(scanner.RParen) {
					panic("Expected ')' or ',' after function call argument")
				}
			}
			p.consumeOne()
			e = functionCall{e, args}
		} else {
			break
		}
	}
	return e
}

func (p *Parser) consumeUnaryExpr() expr {
	if p.match(scanner.Minus, scanner.LNot) {
		op := p.consumeOne()
		e := p.consumeUnaryExpr()
		return unaryOp{op, e}
	} else {
		return p.consumeCallExpr()
	}
}

func (p *Parser) consumeFactorExpr() expr {
	e := p.consumeUnaryExpr()
	for p.match(scanner.Star, scanner.Slash) {
		op := p.consumeOne()
		right := p.consumeUnaryExpr()
		e = binaryOp{op, e, right}
	}
	return e
}

func (p *Parser) consumeTermExpr() expr {
	e := p.consumeFactorExpr()
	for p.match(scanner.Plus, scanner.Minus) {
		op := p.consumeOne()
		right := p.consumeFactorExpr()
		e = binaryOp{op, e, right}
	}
	return e
}

func (p *Parser) consumeComparisonExpr() expr {
	e := p.consumeTermExpr()
	for p.match(scanner.Gt, scanner.Gte, scanner.Lt, scanner.Lte) {
		op := p.consumeOne()
		right := p.consumeTermExpr()
		e = binaryOp{op, e, right}
	}
	return e
}

func (p *Parser) consumeEqualityExpr() expr {
	e := p.consumeComparisonExpr()
	for p.match(scanner.EqEq, scanner.Ne) {
		op := p.consumeOne()
		right := p.consumeComparisonExpr()
		e = binaryOp{op, e, right}
	}
	return e
}

func (p *Parser) consumeLAndExpr() expr {
	e := p.consumeEqualityExpr()
	for p.match(scanner.LAnd) {
		op := p.consumeOne()
		right := p.consumeEqualityExpr()
		e = binaryOp{op, e, right}
	}
	return e
}

func (p *Parser) consumeLOrExpr() expr {
	e := p.consumeLAndExpr()
	for p.match(scanner.LOr) {
		op := p.consumeOne()
		right := p.consumeLAndExpr()
		e = binaryOp{op, e, right}
	}
	return e
}

func (p *Parser) consumeExpr() expr {
	return p.consumeLOrExpr()
}

func (p *Parser) consumeGroupExpr() expr {
	p.consume(scanner.LParen, "Expected '('")
	e := p.consumeExpr()
	p.consume(scanner.RParen, "Expected ')'")
	return e
}

func (p *Parser) consumeIfStmt() Stmt {
	p.consumeKeyword("if", "Expected 'if' statement")
	cond := p.consumeGroupExpr()
	then := p.consumeBlock()
	if p.matchKeyword("else") {
		p.consumeOne()
		if p.matchKeyword("if") {
			return ifStmt{cond, then, block{[]Stmt{p.consumeIfStmt()}}}
		} else {
			return ifStmt{cond, then, p.consumeBlock()}
		}
	} else {
		return ifStmt{cond, then, block{}}
	}
}

func (p *Parser) consumeWhileStmt() Stmt {
	p.consumeKeyword("while", "Expected 'while' statement")
	cond := p.consumeGroupExpr()
	body := p.consumeBlock()
	return whileStmt{cond, body}
}

func (p *Parser) consumeReturnStmt() Stmt {
	p.consumeKeyword("return", "Expected 'return' statement")
	e := p.consumeExpr()
	p.consume(scanner.Semicolon, "Expected ';' after return statement")
	return returnStmt{e}
}

func (p *Parser) consumeVarStmt() Stmt {
	kind := p.consume(scanner.Ident, "Expected variable declaration type")
	name := p.consume(scanner.Ident, "Expected variable declaration name")
	if p.match(scanner.Semicolon) {
		p.consumeOne()
		return varStmt{kind, name, nil}
	} else {
		p.consume(scanner.Eq, "Expected ';' or '=' after variable declaration")
		e := p.consumeExpr()
		p.consume(scanner.Semicolon, "Expected ';' after variable initialization")
		return varStmt{kind, name, e}
	}
}

func (p *Parser) consumeAssignStmt() Stmt {
	target := p.consume(scanner.Ident, "Expected variable assignment target")
	p.consume(scanner.Eq, "Expected '=' after variable assignment target")
	e := p.consumeExpr()
	p.consume(scanner.Semicolon, "Expected ';' after variable assignment")
	return assignStmt{target, e}
}

func (p *Parser) consumeFunctionStmt() Stmt {
	returnKind := p.consume(scanner.Ident, "Expected function return type")
	name := p.consume(scanner.Ident, "Expected function name")
	p.consume(scanner.LParen, "Expected function parameters")
	var params []functionParam
	for !p.match(scanner.RParen) {
		pkind := p.consume(scanner.Ident, "Expected function parameter type")
		pname := p.consume(scanner.Ident, "Expected function parameter name")
		params = append(params, functionParam{pkind, pname})
		if p.match(scanner.Comma) {
			p.consumeOne()
		} else if !p.match(scanner.RParen) {
			panic("Expected ')' or ',' after function parameter")
		}
	}
	p.consumeOne()
	body := p.consumeBlock()
	return functionStmt{returnKind, name, params, body}
}

func (p *Parser) consumeStmt() Stmt {
	if p.matchKeyword("return") {
		return p.consumeReturnStmt()
	} else if p.matchKeyword("while") {
		return p.consumeWhileStmt()
	} else if p.matchKeyword("if") {
		return p.consumeIfStmt()
	} else if p.matchN(1, scanner.Eq) {
		return p.consumeAssignStmt()
	} else if p.matchN(2, scanner.LParen) {
		return p.consumeFunctionStmt()
	} else if p.matchN(2, scanner.Eq, scanner.Semicolon) {
		return p.consumeVarStmt()
	} else {
		e := p.consumeCallExpr()
		p.consume(scanner.Semicolon, "Expected ';' after function call statement")
		return e
	}
}

func (p *Parser) consumeTopLevelStmt() Stmt {
	if p.matchN(2, scanner.LParen) {
		return p.consumeFunctionStmt()
	} else if p.matchN(2, scanner.Eq, scanner.Semicolon) {
		return p.consumeVarStmt()
	} else {
		panic("Unknown top-level statement")
	}
}

func (p *Parser) ConsumeTopLevelStmts() []Stmt {
	var stmts []Stmt
	for !p.match(scanner.Eof) {
		stmts = append(stmts, p.consumeTopLevelStmt())
	}
	return stmts
}

func (p *Parser) consumeBlock() block {
	var blk block
	p.consume(scanner.LBrace, "Required block")
	for !p.match(scanner.RBrace) {
		// Ignore lone semicolons
		if p.match(scanner.Semicolon) {
			p.consumeOne()
			continue
		}
		blk.stmts = append(blk.stmts, p.consumeStmt())
	}
	p.consumeOne()
	return blk
}
