package parser

import "lang/scanner"

type expr interface{}

type Stmt interface{}

type block struct {
	stmts []Stmt
}

type functionParam struct {
	kind scanner.Token
	name scanner.Token
}

type functionStmt struct {
	returnKind scanner.Token
	name       scanner.Token
	params     []functionParam
	body       block
}

type returnStmt struct{ expr }

type assignStmt struct {
	target expr
	expr
}

type varStmt struct {
	kind scanner.Token
	name scanner.Token
	expr
}

type ifStmt struct {
	cond expr
	then block
	els  block
}

type whileStmt struct {
	cond expr
	body block
}

type memberAccess struct {
	parent expr
	name   scanner.Token
}

type functionCall struct {
	callee expr
	args   []expr
}

type unaryOp struct {
	op scanner.Token
	expr
}

type binaryOp struct {
	op    scanner.Token
	left  expr
	right expr
}

type literalStr struct{ value string }

type literalNum struct{ value string }

type literalBool struct{ value bool }

type literalNull struct{}

type identExpr struct{ name scanner.Token }
