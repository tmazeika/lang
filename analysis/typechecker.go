package analysis

import (
	"fmt"
	"lang/parser"
	"lang/scanner"
	"strings"
)

type Symbol string

type Type interface{}

type PrimitiveType int

const (
	Bool PrimitiveType = iota
	Float
	Int
	String
)

type FunctionType struct {
	Return Type
	Params []Type
}

type CompoundType map[Symbol]Type

type SymbolTypesTable struct {
	Parent  *SymbolTypesTable
	Symbols map[Symbol]Type
}

func (t *SymbolTypesTable) find(name Symbol) Type {
	if symType, ok := t.Symbols[name]; ok {
		return symType
	} else if t.Parent == nil {
		return nil
	} else {
		return t.Parent.find(name)
	}
}

func (t *SymbolTypesTable) contains(name Symbol) bool {
	return t.find(name) != nil
}

type Env struct {
	Vars  SymbolTypesTable
	Types SymbolTypesTable
}

func (e *Env) addFunction(f parser.FunctionStmt) error {
	retSym := Symbol(f.ReturnKind.Lexeme)
	if !e.Types.contains(retSym) {
		return fmt.Errorf("unknown return type %q", retSym)
	}
	nameSym := Symbol(f.Name.Lexeme)
	var paramTypes []Type
	for _, param := range f.Params {
		sym := Symbol(param.Kind.Lexeme)
		if !e.Types.contains(sym) {
			return fmt.Errorf("unknown parameter type %q", sym)
		}
		paramTypes = append(paramTypes, e.Types.find(sym))
	}
	e.Vars.Symbols[nameSym] = FunctionType{
		Return: e.Types.find(retSym),
		Params: paramTypes,
	}
	return nil
}

func (e *Env) addVar(v parser.VarStmt) error {
	typeSym := Symbol(v.Kind.Lexeme)
	if !e.Types.contains(typeSym) {
		return fmt.Errorf("unknown variable type %q", typeSym)
	}
	nameSym := Symbol(v.Name.Lexeme)
	e.Vars.Symbols[nameSym] = e.Types.find(typeSym)
	return nil
}

func newEnv(env Env) Env {
	return Env{
		Vars:  SymbolTypesTable{Parent: &env.Vars, Symbols: map[Symbol]Type{}},
		Types: SymbolTypesTable{Parent: &env.Types, Symbols: map[Symbol]Type{}},
	}
}

func Check(stmts []parser.Stmt) bool {
	env := Env{
		Vars: SymbolTypesTable{
			Symbols: map[Symbol]Type{
				"testVars": CompoundType{
					"someBool": Bool,
					"someInt":  Int,
				},
			},
		},
		Types: SymbolTypesTable{
			Symbols: map[Symbol]Type{
				"bool":   Bool,
				"float":  Float,
				"int":    Int,
				"string": String,
			},
		},
	}
	ok := true
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case parser.FunctionStmt:
			if err := env.addFunction(s); err != nil {
				fmt.Println(err)
				ok = false
			}
		case parser.VarStmt:
			if err := env.addVar(s); err != nil {
				fmt.Println(err)
				ok = false
			}
		}
	}
	for _, stmt := range stmts {
		if !IsType(env, stmt, nil) {
			ok = false
		}
	}
	return ok
}

func IsType(env Env, node interface{}, expected Type) bool {
	switch n := node.(type) {
	case parser.LiteralBool:
		return expected == Bool
	case parser.LiteralStr:
		return expected == String
	case parser.LiteralNum:
		if strings.ContainsRune(n.Value, '.') {
			return expected == Float
		}
		return expected == Int
	case parser.IdentExpr:
		sym := Symbol(n.Name.Lexeme)
		return env.Vars.contains(sym) && env.Vars.find(sym) == expected
	case parser.BinaryOp:
		switch n.Op.Kind {
		case scanner.Plus, scanner.Minus, scanner.Star, scanner.Slash, scanner.Gt, scanner.Gte, scanner.Lt, scanner.Lte:
			return IsType(env, n.Left, Int) && IsType(env, n.Right, Int) ||
				IsType(env, n.Left, Float) && IsType(env, n.Right, Float)
		case scanner.EqEq:
			return IsType(env, n.Left, Int) && IsType(env, n.Right, Int) ||
				IsType(env, n.Left, Float) && IsType(env, n.Right, Float) ||
				IsType(env, n.Left, Bool) && IsType(env, n.Right, Bool) ||
				IsType(env, n.Left, String) && IsType(env, n.Right, String)
		case scanner.LAnd, scanner.LOr:
			return IsType(env, n.Left, Bool) && IsType(env, n.Right, Bool)
		default:
			panic("Unknown binary op")
		}
	case parser.UnaryOp:
		switch n.Op.Kind {
		case scanner.LNot:
			return IsType(env, n.Expr, Bool)
		case scanner.Minus:
			return IsType(env, n.Expr, Int) || IsType(env, n.Expr, Float)
		default:
			panic("Unknown unary op")
		}
	case parser.FunctionStmt:
		e := newEnv(env)
		retType := e.Types.find(Symbol(n.ReturnKind.Lexeme))
		for _, param := range n.Params {
			typeSym := Symbol(param.Kind.Lexeme)
			nameSym := Symbol(param.Name.Lexeme)
			e.Vars.Symbols[nameSym] = e.Types.find(typeSym)
		}
		return IsType(e, n.Body, retType)
	case parser.Block:
		ok := true
		e := newEnv(env)
		for _, stmt := range n.Stmts {
			switch s := stmt.(type) {
			case parser.VarStmt:
				typeSym := Symbol(s.Kind.Lexeme)
				nameSym := Symbol(s.Name.Lexeme)
				if !e.Types.contains(typeSym) {
					fmt.Printf("unknown variable type %q\n", typeSym)
					ok = false
					continue
				}
				t := e.Types.find(typeSym)
				e.Vars.Symbols[nameSym] = t
				if !IsType(e, s.Expr, t) {
					ok = false
				}
			case parser.ReturnStmt:
				if !IsType(e, s.Expr, expected) {
					ok = false
				}
			default:
				if !IsType(e, s, expected) {
					ok = false
				}
			}
		}
		return ok
	case parser.IfStmt:
		ok := IsType(env, n.Cond, Bool)
		if !IsType(env, n.Then, expected) {
			ok = false
		}
		if !IsType(env, n.Els, expected) {
			ok = false
		}
		return ok
	case parser.WhileStmt:
		ok := IsType(env, n.Cond, Bool)
		if !IsType(env, n.Body, expected) {
			ok = false
		}
		return ok
	case parser.MemberAccess:
		t := getType(env, n)
		if expected != nil {
			return t == expected
		} else {
			return true
		}
	default:
		return false
	}
}

func getType(env Env, node interface{}) Type {
	switch n := node.(type) {
	case parser.IdentExpr:
		sym := Symbol(n.Name.Lexeme)
		return env.Vars.find(sym)
	case parser.MemberAccess:
		parentType := getType(env, n.Parent)
		memberSym := Symbol(n.Name.Lexeme)
		if comp, ok := parentType.(CompoundType); ok {
			return comp[memberSym]
		} else {
			panic("Tried to access a member of a non-compound type")
		}
	}
	panic("Tried to get type of non-member access")
}
