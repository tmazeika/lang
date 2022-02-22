package parser

import "lang/scanner"

type Expr interface{}

type Stmt interface{}

type Block struct {
	Stmts []Stmt
}

type FunctionParam struct {
	Kind scanner.Token
	Name scanner.Token
}

type FunctionStmt struct {
	ReturnKind scanner.Token
	Name       scanner.Token
	Params     []FunctionParam
	Body       Block
}

type ReturnStmt struct{ Expr }

type AssignStmt struct {
	Target Expr
	Expr
}

type VarStmt struct {
	Kind scanner.Token
	Name scanner.Token
	Expr
}

type IfStmt struct {
	Cond Expr
	Then Block
	Els  Block
}

type WhileStmt struct {
	Cond Expr
	Body Block
}

type MemberAccess struct {
	Parent Expr
	Name   scanner.Token
}

type FunctionCall struct {
	Callee Expr
	Args   []Expr
}

type UnaryOp struct {
	Op scanner.Token
	Expr
}

type BinaryOp struct {
	Op    scanner.Token
	Left  Expr
	Right Expr
}

type LiteralStr struct{ Value string }

type LiteralNum struct{ Value string }

type LiteralBool struct{ Value bool }

type LiteralNull struct{}

type IdentExpr struct{ Name scanner.Token }
