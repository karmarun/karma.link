// Copyright 2018 karma.run AG. All rights reserved.

package extract // import "github.com/karmarun/karma.link/ast/extract"

import (
	"bytes"
	"fmt"
	"github.com/karmarun/karma.link/ast"
	"github.com/karmarun/karma.link/types"
	"strconv"
)

// ContractDefinitions extracts all ast.ContractDefinition's from an ast.SourceUnit.
func ContractDefinitions(root ast.SourceUnit) []ast.ContractDefinition {
	children := root.Children()
	definitions := make([]ast.ContractDefinition, 0, len(children))
	for _, child := range children {
		if contractDefinition, ok := child.(ast.ContractDefinition); ok {
			definitions = append(definitions, contractDefinition)
		}
	}
	return definitions
}

// ContractAPI extracts all functions from a ast.ContractDefinition.
// Remember that public top-level variables in Solidity export automatic getter functions.
// These functions are also part of the extracted API.
// typeMap is used to resolve type references in the process.
func ContractAPI(contractDefinition ast.ContractDefinition, typeMap types.Map) ([]types.Function, error) {
	children := contractDefinition.Children()
	extracted := make([]types.Function, 0, len(children))
	for _, child := range children {

		if variableDeclaration, ok := child.(ast.VariableDeclaration); ok {
			function := VariableAPI(variableDeclaration, typeMap)
			extracted = append(extracted, function)
		}

		if functionDefinition, ok := child.(ast.FunctionDefinition); ok {
			if functionDefinition.IsConstructor {
				continue // constructor is not part of the API
			}
			function, e := FunctionAPI(functionDefinition, typeMap)
			if e != nil {
				return nil, e
			}
			extracted = append(extracted, function)
		}

	}
	return extracted, nil
}

// VariableAPI extracts the generated getter function that public top-level
// contract variables get automatically in Solidity.
func VariableAPI(variableDeclaration ast.VariableDeclaration, typeMap types.Map) types.Function {

	typeId := variableDeclaration.Children()[0].Header().Id
	typ := typeMap.Deref(types.Reference(typeId))

	// mapping and array variables accept parameters:
	// - arrays take index as uint256 + accessor for $valueType (recursively)
	// - mappings take index as $keyType + accessor for $valueType (recursively)
	// they return a single value of the last "concrete" type

	inputs, output := variableAccessor(typ, typeMap, nil)

	return types.Function{
		Name:       variableDeclaration.Name,
		Visibility: variableDeclaration.Visibility,
		Inputs:     inputs,
		Outputs:    []types.Type{output},
		Definition: variableDeclaration,
	}
}

func variableAccessor(typ types.Type, typeMap types.Map, prev []types.Type) ([]types.Type, types.Type) {
	concreteType := typ
	if mapping, ok := concreteType.(types.Mapping); ok {
		return variableAccessor(mapping.Value, typeMap, append(prev, mapping.Key))
	}
	if array, ok := concreteType.(types.Array); ok {
		return variableAccessor(array.Type, typeMap, append(prev, types.Elementary("uint256")))
	}
	return prev, concreteType
}

// FunctionAPI extracts an ast.FunctionDefinition's type information.
func FunctionAPI(functionDefinition ast.FunctionDefinition, typeMap types.Map) (types.Function, error) {

	children := functionDefinition.Children()

	inParamList, ok := children[0].(ast.ParameterList)
	if !ok {
		return types.Function{}, fmt.Errorf(`functionDefinition's first child expected to be ParameterList`)
	}

	outParamList, ok := children[1].(ast.ParameterList)
	if !ok {
		return types.Function{}, fmt.Errorf(`functionDefinition's second child expected to be ParameterList`)
	}

	inParams, outParams := inParamList.Children(), outParamList.Children()
	inputs, outputs := make([]types.Type, len(inParams), len(inParams)), make([]types.Type, len(outParams), len(outParams))

	for i, child := range inParams {
		variableDeclaration, ok := child.(ast.VariableDeclaration)
		if !ok {
			return types.Function{}, fmt.Errorf(`paramList's children expected to be VariableDeclarations`)
		}
		typeId := variableDeclaration.Children()[0].Header().Id
		inputs[i] = typeMap.Deref(types.Reference(typeId))
	}

	for i, child := range outParams {
		variableDeclaration, ok := child.(ast.VariableDeclaration)
		if !ok {
			return types.Function{}, fmt.Errorf(`paramList's children expected to be VariableDeclarations`)
		}
		typeId := variableDeclaration.Children()[0].Header().Id
		outputs[i] = typeMap.Deref(types.Reference(typeId))
	}

	for _, input := range inputs {
		if _, ok := input.(types.Reference); ok {
			panic("")
		}
	}

	return types.Function{
		Name:            functionDefinition.Name,
		Visibility:      functionDefinition.Visibility,
		StateMutability: functionDefinition.StateMutability,
		NatSpec:         functionDefinition.Documentation,
		Inputs:          inputs,
		Outputs:         outputs,
		Definition:      functionDefinition,
	}, nil
}

// Types extracts all type definitions and references from an ast.SourceUnit.
func Types(path string, root ast.SourceUnit) (types.Map, error) {
	extracted := make(types.Map, 64)
	contractName := "" // NOTE: we make delicate use of pre-order traversal to fix EventDefinition's canonicalName
	err := (error)(nil)
	ast.PreTraverse(root, func(node ast.Node) {
		if err != nil {
			return
		}
		ref := types.Reference(node.Header().Id)
		if node, ok := node.(ast.ContractDefinition); ok {
			contractName = node.Name
			t, e := Type(path, node)
			if e != nil {
				err = e
				return
			}
			extracted[ref] = t
		}
		if node, ok := node.(ast.ElementaryTypeName); ok {
			t, e := Type(path, node)
			if e != nil {
				err = e
				return
			}
			extracted[ref] = t
		}
		if node, ok := node.(ast.ArrayTypeName); ok {
			t, e := Type(path, node)
			if e != nil {
				err = e
				return
			}
			extracted[ref] = t
		}
		if node, ok := node.(ast.EnumDefinition); ok {
			t, e := Type(path, node)
			if e != nil {
				err = e
				return
			}
			extracted[ref] = t
		}
		if node, ok := node.(ast.EventDefinition); ok {
			node.CanonicalName = contractName + "." + node.Name
			t, e := Type(path, node)
			if e != nil {
				err = e
				return
			}
			extracted[ref] = t
		}
		if node, ok := node.(ast.StructDefinition); ok {
			t, e := Type(path, node)
			if e != nil {
				err = e
				return
			}
			extracted[ref] = t
		}
		if node, ok := node.(ast.Mapping); ok {
			t, e := Type(path, node)
			if e != nil {
				err = e
				return
			}
			extracted[ref] = t
		}
		if node, ok := node.(ast.UserDefinedTypeName); ok {
			t, e := Type(path, node)
			if e != nil {
				err = e
				return
			}
			extracted[ref] = t
		}
	})
	if err != nil {
		return nil, err
	}
	return extracted, nil
}

// Type extracts type information from an AST node which may be any of:
// ast.ContractDefinition,
// ast.UserDefinedTypeName,
// ast.ElementaryTypeName,
// ast.ArrayTypeName,
// ast.EnumDefinition,
// ast.StructDefinition,
// ast.EventDefinition,
// ast.Mapping.
// It returns an error for everything else.
func Type(path string, node ast.Node) (types.Type, error) {
	if node, ok := node.(ast.ContractDefinition); ok {
		switch node.ContractKind {

		case ast.ContractKindContract:
			return types.ContractAddress(path + ":" + node.Name), nil

		case ast.ContractKindInterface:
			return types.InterfaceAddress(path + ":" + node.Name), nil

		case ast.ContractKindLibrary:
			return types.LibraryAddress(path + ":" + node.Name), nil

		default:
			return nil, fmt.Errorf(`unexpected contract kind: %s`, node.ContractKind)
		}
	}
	if node, ok := node.(ast.UserDefinedTypeName); ok {
		return types.Reference(node.ReferencedDeclaration), nil
	}
	if node, ok := node.(ast.ElementaryTypeName); ok {
		return types.Elementary(node.Type), nil
	}
	if node, ok := node.(ast.ArrayTypeName); ok {
		return ArrayType(path, node)
	}
	if node, ok := node.(ast.EnumDefinition); ok {
		return EnumType(path, node)
	}
	if node, ok := node.(ast.StructDefinition); ok {
		return StructType(path, node)
	}
	if node, ok := node.(ast.EventDefinition); ok {
		return EventType(path, node)
	}
	if node, ok := node.(ast.Mapping); ok {
		return MappingType(path, node)
	}
	return nil, fmt.Errorf(`unexpected ast.Node type in extract.Type: %T`, node)
}

// EventType extracts a named types.Event from an ast.EventDefinition.
func EventType(path string, eventDefinition ast.EventDefinition) (types.Named, error) {

	children := eventDefinition.Children()
	if len(children) != 1 {
		return types.Named{}, fmt.Errorf(`expected eventDefinition to have exactly one child`)
	}

	paramList, ok := children[0].(ast.ParameterList)
	if !ok {
		return types.Named{}, fmt.Errorf(`eventDefinition's child expected to be ParameterList`)
	}

	params := paramList.Children()
	args := make([]types.Type, len(params), len(params))

	for i, param := range params {
		variableDeclaration, ok := param.(ast.VariableDeclaration)
		if !ok {
			return types.Named{}, fmt.Errorf(`eventDefinition's ParameterList's children expected to be VariableDeclarations`)
		}
		varChildren := variableDeclaration.Children()
		if len(varChildren) != 1 {
			return types.Named{}, fmt.Errorf(`variableDeclaration expected to have 1 child`)
		}
		t, e := Type(path, varChildren[0])
		if e != nil {
			return types.Named{}, e
		}
		args[i] = t
	}

	return types.Named{
		Name: path + ":" + eventDefinition.CanonicalName,
		Type: types.Event{
			Name: eventDefinition.Name,
			Args: args,
		},
	}, nil

}

// StructType extracts a named types.Struct from an ast.StructDefinition.
func StructType(path string, structDefinition ast.StructDefinition) (types.Named, error) {
	children := structDefinition.Children()
	strct := types.Struct{
		Keys:  make([]string, len(children), len(children)),
		Types: make([]types.Type, len(children), len(children)),
	}
	for i, child := range children {
		variableDeclaration, ok := child.(ast.VariableDeclaration)
		if !ok {
			return types.Named{}, fmt.Errorf(`structDefinition's children expected to be VariableDeclarations`)
		}
		varChildren := variableDeclaration.Children()
		if len(varChildren) != 1 {
			return types.Named{}, fmt.Errorf(`variableDeclaration expected to have 1 child`)
		}
		strct.Keys[i] = variableDeclaration.Name
		t, e := Type(path, varChildren[0])
		if e != nil {
			return types.Named{}, e
		}
		strct.Types[i] = t
	}
	return types.Named{
		Name: path + ":" + structDefinition.CanonicalName,
		Type: strct,
	}, nil
}

// EnumType extracts a named types.Enum from an ast.EnumDefinition.
func EnumType(path string, enumDefinition ast.EnumDefinition) (types.Named, error) {
	children := enumDefinition.Children()
	enum := make(types.Enum, len(children), len(children))
	for i, child := range children {
		enumValue, ok := child.(ast.EnumValue)
		if !ok {
			return types.Named{}, fmt.Errorf(`enumDefinition's children expected to be EnumValues`)
		}
		enum[i] = enumValue.Name
	}
	return types.Named{
		Name: path + ":" + enumDefinition.CanonicalName,
		Type: enum,
	}, nil
}

// EnumType extracts a types.Array from an ast.ArrayTypeName.
func ArrayType(path string, arrayTypeName ast.ArrayTypeName) (types.Array, error) {
	children := arrayTypeName.Children()
	if len(children) == 1 {
		t, e := Type(path, children[0])
		if e != nil {
			return types.Array{}, e
		}
		return types.Array{
			Length: types.DynamicArrayLength,
			Type:   t,
		}, nil
	}
	if len(children) == 2 {
		// second child is literal of fixed array length, which can be a static expression.
		// we parse the length out of the type name instead of evaluating arbitrary expressions here.
		nameBytes := ([]byte)(arrayTypeName.Type)
		lenBytes := nameBytes[bytes.LastIndex(nameBytes, []byte("["))+1 : len(nameBytes)-1]
		length, e := strconv.ParseInt(string(lenBytes), 10, 64)
		if e != nil {
			return types.Array{}, e
		}
		t, e := Type(path, children[0])
		if e != nil {
			return types.Array{}, e
		}
		return types.Array{
			Length: int(length),
			Type:   t,
		}, nil
	}
	return types.Array{}, fmt.Errorf(`arrayTypeName expected to have 1 or 2 children`)
}

// MappingType extracts a types.Mapping from an ast.Mapping.
func MappingType(path string, mapping ast.Mapping) (types.Mapping, error) {
	children := mapping.Children()
	if len(children) != 2 {
		return types.Mapping{}, fmt.Errorf(`ast.Mapping expected to have exactly two children`)
	}
	t1, e := Type(path, children[0])
	if e != nil {
		return types.Mapping{}, e
	}
	t2, e := Type(path, children[1])
	if e != nil {
		return types.Mapping{}, e
	}
	return types.Mapping{t1, t2}, nil
}
