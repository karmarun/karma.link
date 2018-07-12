// Copyright 2018 karma.run AG. All rights reserved.

package types // import "github.com/karmarun/karma.link/types"

import (
	"github.com/karmarun/karma.link/ast"
)

type Project struct {
	Path  string
	Files map[string]map[string]*Contract // "subdir/Example.sol" -> "Example" -> *Contract{...}
}

type Contract struct {
	File       string
	Name       string
	Parents    []*Contract
	NatSpec    string
	Kind       ast.ContractKind
	API        map[string]Function // signature -> Function{...}
	Types      map[string]Type
	Definition ast.ContractDefinition
	Binary     []byte
}

func (c Contract) Overloads(name string) []Function {
	functions := make([]Function, 0, 8)
	for _, function := range c.API {
		if function.Name == name {
			functions = append(functions, function)
		}
	}
	if len(functions) == 0 {
		return nil
	}
	return functions
}

const FallbackFunctionName = ""

type Function struct {
	Name            string
	NatSpec         string
	Visibility      ast.Visibility
	StateMutability ast.StateMutability
	Inputs          []Type
	Outputs         []Type
	Definition      ast.Node
}

func (f Function) SoliditySignature() []byte {
	bs := []byte(f.Name + `(`)
	for i, input := range f.Inputs {
		if i > 0 {
			bs = append(bs, ',')
		}
		bs = append(bs, input.SoliditySignature()...)
	}
	return append(bs, ')')
}

func (f Function) IsFallback() bool {
	return f.Name == FallbackFunctionName
}
