package main

import (
	"fmt"
	"github.com/kr/pretty"
	"lang/parser"
	"lang/scanner"
)

func main() {
	tokens := scanner.Scan()

	for _, t := range tokens {
		fmt.Printf("%s\t%q\n", t.Kind, t.Lexeme)
	}

	p := parser.Parser{Tokens: tokens}
	stmts := p.ConsumeTopLevelStmts()
	for _, stmt := range stmts {
		_, err := pretty.Println(stmt)
		if err != nil {
			panic(err)
		}
	}
}
