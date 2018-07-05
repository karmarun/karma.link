package extract

import (
	"ast"
	"bytes"
	"log"
	"strconv"
	"types"
)

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

func ContractAPI(contractDefinition ast.ContractDefinition, typeMap types.Map) []types.Function {
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
			function := FunctionAPI(functionDefinition, typeMap)
			extracted = append(extracted, function)
		}

	}
	return extracted
}

// REMEMBER: public/external variables also part of API as (getter) functions
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

func FunctionAPI(functionDefinition ast.FunctionDefinition, typeMap types.Map) types.Function {

	children := functionDefinition.Children()

	inParamList, ok := children[0].(ast.ParameterList)
	if !ok {
		log.Fatalln("functionDefinition's first child expected to be ParameterList")
	}

	outParamList, ok := children[1].(ast.ParameterList)
	if !ok {
		log.Fatalln("functionDefinition's second child expected to be ParameterList")
	}

	inParams, outParams := inParamList.Children(), outParamList.Children()
	inputs, outputs := make([]types.Type, len(inParams), len(inParams)), make([]types.Type, len(outParams), len(outParams))

	for i, child := range inParams {
		variableDeclaration, ok := child.(ast.VariableDeclaration)
		if !ok {
			log.Fatalln("paramList's children expected to be VariableDeclarations")
		}
		typeId := variableDeclaration.Children()[0].Header().Id
		inputs[i] = typeMap.Deref(types.Reference(typeId))
	}

	for i, child := range outParams {
		variableDeclaration, ok := child.(ast.VariableDeclaration)
		if !ok {
			log.Fatalln("paramList's children expected to be VariableDeclarations")
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
	}
}

func Types(path string, root ast.SourceUnit) types.Map {
	extracted := make(types.Map, 64)
	contractName := "" // NOTE: we make delicate use of pre-order traversal to fix EventDefinition's canonicalName
	ast.PreTraverse(root, func(node ast.Node) {
		if node, ok := node.(ast.ContractDefinition); ok {
			contractName = node.Name
			extracted[types.Reference(node.Header().Id)] = Type(path, node)
		}
		if node, ok := node.(ast.ElementaryTypeName); ok {
			extracted[types.Reference(node.Header().Id)] = Type(path, node)
		}
		if node, ok := node.(ast.ArrayTypeName); ok {
			extracted[types.Reference(node.Header().Id)] = Type(path, node)
		}
		if node, ok := node.(ast.EnumDefinition); ok {
			extracted[types.Reference(node.Header().Id)] = Type(path, node)
		}
		if node, ok := node.(ast.EventDefinition); ok {
			node.CanonicalName = contractName + "." + node.Name
			extracted[types.Reference(node.Header().Id)] = Type(path, node)
		}
		if node, ok := node.(ast.StructDefinition); ok {
			extracted[types.Reference(node.Header().Id)] = Type(path, node)
		}
		if node, ok := node.(ast.Mapping); ok {
			extracted[types.Reference(node.Header().Id)] = Type(path, node)
		}
		if node, ok := node.(ast.UserDefinedTypeName); ok {
			extracted[types.Reference(node.Header().Id)] = Type(path, node)
		}
	})
	return extracted
}

func Type(path string, node ast.Node) types.Type {
	if node, ok := node.(ast.ContractDefinition); ok {
		switch node.ContractKind {

		case ast.ContractKindContract:
			return types.ContractAddress(path + ":" + node.Name)

		case ast.ContractKindInterface:
			return types.InterfaceAddress(path + ":" + node.Name)

		case ast.ContractKindLibrary:
			return types.LibraryAddress(path + ":" + node.Name)

		default:
			log.Fatalln("unexpected contract kind:", node.ContractKind)
		}
	}
	if node, ok := node.(ast.UserDefinedTypeName); ok {
		return types.Reference(node.ReferencedDeclaration)
	}
	if node, ok := node.(ast.ElementaryTypeName); ok {
		return types.Elementary(node.Type)
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
	log.Fatalf("unexpected ast.Node type in extract.Type: %T", node)
	return nil // shut up compiler
}

func EventType(path string, eventDefinition ast.EventDefinition) types.Named {

	children := eventDefinition.Children()
	if len(children) != 1 {
		log.Fatalln(`expected eventDefinition to have exactly one child`)
	}

	paramList, ok := children[0].(ast.ParameterList)
	if !ok {
		log.Fatalln("eventDefinition's child expected to be ParameterList")
	}

	params := paramList.Children()
	args := make([]types.Type, len(params), len(params))

	for i, param := range params {
		variableDeclaration, ok := param.(ast.VariableDeclaration)
		if !ok {
			log.Fatalln("eventDefinition's ParameterList's children expected to be VariableDeclarations")
		}
		varChildren := variableDeclaration.Children()
		if len(varChildren) != 1 {
			log.Fatalln("variableDeclaration expected to have 1 child")
		}
		args[i] = Type(path, varChildren[0])
	}

	return types.Named{
		Name: path + ":" + eventDefinition.CanonicalName,
		Type: types.Event{
			Name: eventDefinition.Name,
			Args: args,
		},
	}

}

func StructType(path string, structDefinition ast.StructDefinition) types.Named {
	children := structDefinition.Children()
	strct := types.Struct{
		Keys:  make([]string, len(children), len(children)),
		Types: make([]types.Type, len(children), len(children)),
	}
	for i, child := range children {
		variableDeclaration, ok := child.(ast.VariableDeclaration)
		if !ok {
			log.Fatalln("structDefinition's children expected to be VariableDeclarations")
		}
		varChildren := variableDeclaration.Children()
		if len(varChildren) != 1 {
			log.Fatalln("variableDeclaration expected to have 1 child")
		}
		strct.Keys[i], strct.Types[i] = variableDeclaration.Name, Type(path, varChildren[0])
	}
	return types.Named{
		Name: path + ":" + structDefinition.CanonicalName,
		Type: strct,
	}
}

func EnumType(path string, enumDefinition ast.EnumDefinition) types.Named {
	children := enumDefinition.Children()
	enum := make(types.Enum, len(children), len(children))
	for i, child := range children {
		enumValue, ok := child.(ast.EnumValue)
		if !ok {
			log.Fatalln("enumDefinition's children expected to be EnumValues")
		}
		enum[i] = enumValue.Name
	}
	return types.Named{
		Name: path + ":" + enumDefinition.CanonicalName,
		Type: enum,
	}
}

func ArrayType(path string, arrayTypeName ast.ArrayTypeName) types.Array {
	children := arrayTypeName.Children()
	if len(children) == 1 {
		return types.Array{
			Length: types.DynamicArrayLength,
			Type:   Type(path, children[0]),
		}
	}
	if len(children) == 2 {
		// second child is literal of fixed array length, which can be a static expression.
		// we parse the length out of the type name instead of evaluating arbitrary expressions here.
		nameBytes := ([]byte)(arrayTypeName.Type)
		lenBytes := nameBytes[bytes.LastIndex(nameBytes, []byte("["))+1 : len(nameBytes)-1]
		length, e := strconv.ParseInt(string(lenBytes), 10, 64)
		if e != nil {
			log.Fatalln(e)
		}
		return types.Array{
			Length: int(length),
			Type:   Type(path, children[0]),
		}
	}
	log.Fatalln("arrayTypeName expected to have 1 or 2 children")
	panic("")
}

func MappingType(path string, mapping ast.Mapping) types.Mapping {
	children := mapping.Children()
	if len(children) != 2 {
		log.Fatalln("mapping expected to have 2 children")
	}
	return types.Mapping{Type(path, children[0]), Type(path, children[1])}
}
