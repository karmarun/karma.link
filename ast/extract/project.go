// Copyright 2018 karma.run AG. All rights reserved.
package extract

import (
	"encoding/hex"
	"github.com/karmarun/karma.link/ast"
	"github.com/karmarun/karma.link/types"
	"log"
	"strings"
)

// TODO: Windows support: normalize paths to forward slashes without drive letters, etc.
func Project(combined ast.Combined) types.Project {

	lpp := longestPathPrefix{}
	for _, path := range combined.SourceList {
		lpp.Observe(path)
	}

	typeMap, sourceUnits := make(types.Map, 128), make(map[string]ast.SourceUnit, len(combined.Sources))
	for path, source := range combined.Sources {
		path = lpp.RemovePrefix(path)
		sourceUnit := ast.UnserializeJSON(source.AST).(ast.SourceUnit)
		sourceUnits[path] = sourceUnit
		for id, typ := range Types(path, sourceUnit) {
			typeMap[id] = typ
		}
	}

	{ // resolve all type references
		var resolve func(t types.Type) types.Type
		resolve = func(t types.Type) types.Type {
			if ref, ok := t.(types.Reference); ok {
				return typeMap.Deref(ref).Map(resolve)
			}
			return t
		}
		for ref, typ := range typeMap {
			typeMap[ref] = typ.Map(resolve)
		}
	}

	contractMap := make(map[int]*types.Contract, len(sourceUnits)*2)

	for path, sourceUnit := range sourceUnits {

		contractDefinitions := ContractDefinitions(sourceUnit)

		for _, contractDefinition := range contractDefinitions {
			functions := ContractAPI(contractDefinition, typeMap)
			api := make(map[string]types.Function, len(functions))
			for _, function := range functions {
				api[string(function.SoliditySignature())] = function
			}
			bin := []byte(nil)
			if compiled, ok := combined.Contracts[lpp.PrependPrefix(path)+`:`+contractDefinition.Name]; ok {
				bs, e := hex.DecodeString(compiled.Binary)
				if e != nil {
					log.Fatalln(`invalid binary in contract`, path)
				}
				bin = bs
			}
			contractMap[contractDefinition.Header().Id] = &types.Contract{
				File:       path,
				Name:       contractDefinition.Name,
				Parents:    make([]*types.Contract, 0, len(contractDefinition.LinearizedBaseContracts)-1), // NOTE: filled below
				Types:      make(map[string]types.Type, 16),                                               // idem
				NatSpec:    contractDefinition.Documentation,
				Kind:       contractDefinition.ContractKind,
				API:        api,
				Definition: contractDefinition,
				Binary:     bin,
			}
		}

	}

	project := types.Project{
		Path:  "",
		Files: make(map[string]map[string]*types.Contract, len(contractMap)),
	}

	project.Path, _ = lpp.Prefix()

	for _, contract := range contractMap {
		// NOTE: first element of contract.Definition.LinearizedBaseContracts is own ID
		for _, parentId := range contract.Definition.LinearizedBaseContracts[1:] {
			parent, ok := contractMap[parentId]
			if !ok {
				log.Fatalln("missing contract parent definition")
			}
			contract.Parents = append(contract.Parents, parent)
		}
		for _, typ := range typeMap {
			// NOTE: event types have no canonicalName field in AST and can therefore not be associated
			// with a particular contract through this logic.
			if named, ok := typ.(types.Named); ok && strings.HasPrefix(named.Name, (contract.File+":"+contract.Name)) {
				typeName := named.Name[strings.LastIndex(named.Name, `.`)+1:]
				contract.Types[typeName] = named
			}
		}
		contracts := project.Files[contract.File]
		if contracts == nil {
			contracts = make(map[string]*types.Contract, 8)
		}
		contracts[contract.Name] = contract
		project.Files[contract.File] = contracts
	}

	return project

}

type longestPathPrefix struct {
	init bool
	prfx string
}

func (lpp *longestPathPrefix) Observe(path string) {
	if lpp.init {
		lpp.prfx = trimToLastSeparator(longestCommonPrefix(path, lpp.prfx))
	} else {
		lpp.init, lpp.prfx = true, trimToLastSeparator(path)
	}
}

func (lpp *longestPathPrefix) RemovePrefix(path string) string {
	l := len(lpp.prfx)
	if path[:l] != lpp.prfx {
		panic("")
	}
	return path[l:]
}

func (lpp *longestPathPrefix) PrependPrefix(path string) string {
	if len(path) > 0 && path[0] == '/' {
		return lpp.prfx + path[1:]
	}
	return lpp.prfx + path
}

func (lpp *longestPathPrefix) Prefix() (string, bool) {
	return lpp.prfx, lpp.init
}

// TODO: Windows support with \ and drive letters with colons.
func trimToLastSeparator(s string) string {
	for len(s) > 0 && s[len(s)-1] != '/' {
		s = s[:len(s)-1]
	}
	return s
}

func longestCommonPrefix(a, b string) string {
	if a == "" || b == "" {
		return ""
	}
	for i, l := 0, minInt(len(a), len(b)); i < l; i++ {
		if a[i] == b[i] {
			continue
		}
		return a[:i]
	}
	if len(a) < len(b) {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
