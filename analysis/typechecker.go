package analysis

import (
	"lang/parser"
	"lang/scanner"
	"strings"
)

type SymbolTable struct {
	Parent  *SymbolTable
	Symbols map[string]string
}

func (t *SymbolTable) find(name string) bool {
	if _, ok := t.Symbols[name]; ok {
		return true
	} else if t.Parent == nil {
		return false
	} else {
		return t.Parent.find(name)
	}
}

type Env struct {
	Vars  SymbolTable
	Types SymbolTable
}

func newEnv(env Env) Env {
	return Env{
		Vars:  SymbolTable{Parent: &env.Vars, Symbols: map[string]string{}},
		Types: SymbolTable{Parent: &env.Types, Symbols: map[string]string{}},
	}
}

func Check(stmts []parser.Stmt) bool {
	env := Env{
		Vars: SymbolTable{
			Symbols: map[string]string{},
		},
		Types: SymbolTable{
			Symbols: map[string]string{
				"bool":   "",
				"string": "",
				"int":    "",
				"float":  "",
				"rune":   "",
			},
		},
	}
	ok := true
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case parser.FunctionStmt:
			env.Vars.Symbols[s.Name.Lexeme] = "" // TODO
		case parser.VarStmt:
			env.Vars.Symbols[s.Name.Lexeme] = s.Kind.Lexeme
		}
		ok = ok && IsType(env, stmt, "")
	}
	return ok
}

func IsType(env Env, node interface{}, expected string) bool {
	switch n := node.(type) {
	case parser.LiteralBool:
		return expected == "bool"
	case parser.LiteralStr:
		return expected == "string"
	case parser.LiteralNum:
		if strings.ContainsRune(n.Value, '.') {
			return expected == "float"
		}
		return expected == "int"
	case parser.IdentExpr:
		return env.Vars.find(n.Name.Lexeme) && env.Vars.Symbols[n.Name.Lexeme] == expected
	case parser.BinaryOp:
		switch n.Op.Kind {
		case scanner.Plus, scanner.Minus, scanner.Star, scanner.Slash, scanner.Gt, scanner.Gte, scanner.Lt, scanner.Lte:
			return IsType(env, n.Left, "int") && IsType(env, n.Right, "int") ||
				IsType(env, n.Left, "float") && IsType(env, n.Right, "float")
		case scanner.EqEq:
			return IsType(env, n.Left, "int") && IsType(env, n.Right, "int") ||
				IsType(env, n.Left, "float") && IsType(env, n.Right, "float") ||
				IsType(env, n.Left, "bool") && IsType(env, n.Right, "bool") ||
				IsType(env, n.Left, "string") && IsType(env, n.Right, "string")
		case scanner.LAnd, scanner.LOr:
			return IsType(env, n.Left, "bool") && IsType(env, n.Right, "bool")
		default:
			panic("Unknown binary op")
		}
	case parser.UnaryOp:
		switch n.Op.Kind {
		case scanner.LNot:
			return IsType(env, n.Expr, "bool")
		case scanner.Minus:
			return IsType(env, n.Expr, "int") || IsType(env, n.Expr, "float")
		default:
			panic("Unknown unary op")
		}
	case parser.FunctionStmt:
		e := newEnv(env)
		for _, param := range n.Params {
			if !env.Types.find(param.Kind.Lexeme) {
				return false
			}
			e.Vars.Symbols[param.Name.Lexeme] = param.Kind.Lexeme
		}
		return e.Types.find(n.ReturnKind.Lexeme) && IsType(e, n.Body, n.ReturnKind.Lexeme)
	case parser.Block:
		ok := true
		e := newEnv(env)
		for _, stmt := range n.Stmts {
			switch s := stmt.(type) {
			case parser.VarStmt:
				e.Vars.Symbols[s.Name.Lexeme] = s.Kind.Lexeme
				ok = ok && e.Types.find(s.Kind.Lexeme) && IsType(e, s.Expr, s.Kind.Lexeme)
			case parser.ReturnStmt:
				ok = ok && IsType(e, s.Expr, expected)
			default:
				ok = ok && IsType(e, s, expected)
			}
		}
		return ok
	case parser.IfStmt:
		ok := IsType(env, n.Cond, "bool")
		ok = ok && IsType(env, n.Then, expected)
		ok = ok && IsType(env, n.Els, expected)
		return ok
	case parser.WhileStmt:
		ok := IsType(env, n.Cond, "bool")
		ok = ok && IsType(env, n.Body, expected)
		return ok
	default:
		return false
	}
}
