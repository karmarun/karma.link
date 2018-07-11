// Copyright 2018 karma.run AG. All rights reserved.

package ast // import "github.com/karmarun/karma.link/ast"

import (
	"encoding/json"
	"log"
)

// ContractKind represents a contract's definition type.
type ContractKind string

const (
	ContractKindContract  ContractKind = "contract"
	ContractKindInterface              = "interface"
	ContractKindLibrary                = "library"
)

// Visibility represents a Solidity function's visibility.
type Visibility string

const (
	VisibilityPublic   Visibility = "public"
	VisibilityInternal            = "internal"
	VisibilityExternal            = "external"
	VisibilityPrivate             = "private"
)

// StateMutability represents a Solidity function's state mutability.
type StateMutability string

const (
	StateMutabilityPure       StateMutability = "pure"
	StateMutabilityView                       = "view"
	StateMutabilityNonpayable                 = "nonpayable"
)

// StorageLocation represents a Solidity variables's storage location.
type StorageLocation string

const (
	StorageLocationDefault StorageLocation = "default"
	StorageLocationMemory                  = "memory"
	StorageLocationStorage                 = "storage"
)

// CompiledContract represents a compiled Solidity contract's binary payload in hex.
type CompiledContract struct {
	Binary string `json:"bin"`
}

// Combined is the top-most node in a Solidity AST.
// It holds all source files involved in the compilation process.
type Combined struct {
	Contracts  map[string]CompiledContract `json:"contracts"`
	SourceList []string                    `json:"sourceList"`
	Sources    map[string]CombinedSource   `json:"sources"`
	Version    string                      `json:"version"`
}

// CombinedSource is the raw JSON analog to Combined.
type CombinedSource struct {
	AST json.RawMessage `json:"AST"`
}

// Header holds the common fields every Solidity AST node has.
type Header struct {
	Id         int               `json:"id"`
	Name       string            `json:"name"`
	Source     string            `json:"src"`
	Attributes json.RawMessage   `json:"attributes"`
	Children   []json.RawMessage `json:"children"`
}

// Node represents a Solidity AST node.
type Node interface {
	Header() Header
	Children() []Node
}

// SourceUnit bundles all Solidity definitions from a single file.
type SourceUnit struct {
	header          Header
	children        []Node
	AbsolutePath    string           `json:"absolutePath"`
	ExportedSymbols map[string][]int `json:"exportedSymbols"`
}

// PragmaDirective represents a Solidity file-level pragma declaration.
type PragmaDirective struct {
	header   Header
	children []Node
	Literals []string `json:"literals"`
}

// ContractDefinition represents a contract definition in a Solidity AST.
type ContractDefinition struct {
	header                  Header
	children                []Node
	Name                    string       `json:"name"`
	Scope                   int          `json:"scope"`
	FullyImplemented        bool         `json:"fullyImplemented"`
	LinearizedBaseContracts []int        `json:"linearizedBaseContracts"`
	Documentation           string       `json:"documentation"`
	ContractKind            ContractKind `json:"contractKind"`
	// BaseContracts        json.RawMessage `json:"baseContracts"`
	// ContractDependencies json.RawMessage `json:"contractDependencies"`
}

// StructDefinition represents a struct definition in a Solidity AST.
type StructDefinition struct {
	header        Header
	children      []Node
	CanonicalName string     `json:"canonicalName"`
	Name          string     `json:"name"`
	Scope         int        `json:"scope"`
	Visibility    Visibility `json:"visibility"`
}

// VariableDeclaration represents a variable declaration in a Solidity AST.
type VariableDeclaration struct {
	header          Header
	children        []Node
	Constant        bool       `json:"constant"`
	Name            string     `json:"name"`
	Scope           int        `json:"scope"`
	StateVariable   bool       `json:"stateVariable"`
	StorageLocation string     `json:"storageLocation"`
	Type            string     `json:"type"`
	Visibility      Visibility `json:"visibility"`
	// Value        json.RawMessage `json:"value"`
}

// ElementaryTypeName represents an elementary type name in a Solidity AST.
type ElementaryTypeName struct {
	header Header
	Name   string `json:"name"`
	Type   string `json:"type"`
}

// ModifierDefinition represents a modifier definition in a Solidity AST.
type ModifierDefinition struct {
	header     Header
	children   []Node
	Name       string     `json:"name"`
	Visibility Visibility `json:"visibility"`
	// Documentation json.RawMessage `json:"documentation"`
}

// ParameterList represents a list of types in a Solidity AST.
// It is not used exclusively in function parameter declarations.
type ParameterList struct {
	header   Header
	children []Node
}

// FunctionDefinition represents a function definition in a Solidity AST.
type FunctionDefinition struct {
	header          Header
	children        []Node
	Constant        bool            `json:"constant"`
	Implemented     bool            `json:"implemented"`
	IsConstructor   bool            `json:"isConstructor"`
	Name            string          `json:"name"`
	Payable         bool            `json:"payable"`
	Scope           int             `json:"scope"`
	StateMutability StateMutability `json:"stateMutability"`
	Visibility      Visibility      `json:"visibility"`
	Documentation   string          `json:"documentation"` //": null,
	// SuperFunction   json.RawMessage `json:"superFunction"` //": null,
}

// UserDefinedTypeName represents a user defined type name (e.g. enums, structs) in a Solidity AST.
type UserDefinedTypeName struct {
	header                Header
	children              []Node
	Name                  string `json:"name"`
	ReferencedDeclaration int    `json:"referencedDeclaration"`
	Type                  string `json:"type"`
	// ContractScope         json.RawMessage `json:"contractScope"`
}

// ModifierInvocation represents a modifier invokation in a Solidity AST.
type ModifierInvocation struct {
	header   Header
	children []Node
	// Arguments json.RawMessage `json:"arguments"`
}

// Identifier represents any identifier in a Solidity AST.
type Identifier struct {
	header                Header
	children              []Node
	Value                 string `json:"value"`
	ReferencedDeclaration int    `json:"referencedDeclaration"`
	// ArgumentTypes          json.RawMessage `json:"argumentTypes"`
	// OverloadedDeclarations json.RawMessage `json:"overloadedDeclarations"`
	// Type                   json.RawMessage `json:"type"`
}

// InheritanceSpecifier represents an inheritance specifier (`contract X is Y, Z`) in a Solidity AST.
type InheritanceSpecifier struct {
	header   Header
	children []Node
	// Arguments json.RawMessage `json:"arguments"`
}

// EnumDefinition represents an enum definition in a Solidity AST.
type EnumDefinition struct {
	header        Header
	children      []Node
	CanonicalName string `json:"canonicalName"`
	Name          string `json:"name"`
}

// EnumValue represents each element in an enum definition in a Solidity AST.
type EnumValue struct {
	header   Header
	children []Node
	Name     string `json:"name"`
}

// Mapping represents a mapping definition in a Solidity AST.
type Mapping struct {
	header   Header
	children []Node
	Type     string `json:"type"`
}

// ArrayTypeName represents an array-type's name (e.g. "int32[8]") in a Solidity AST.
type ArrayTypeName struct {
	header   Header
	children []Node
	Type     string `json:"type"`
}

// UsingForDirective represents the `using X for Y` directive in a Solidity AST.
type UsingForDirective struct {
	header   Header
	children []Node
}

// Literal represents a literal in a Solidity AST.
type Literal struct {
	header          Header
	Hexvalue        string `json:"hexvalue"`
	Value           string `json:"value"`
	IsConstant      bool   `json:"isConstant"`
	IsLValue        bool   `json:"isLValue"`
	IsPure          bool   `json:"isPure"`
	LValueRequested bool   `json:"lValueRequested"`
	Type            string `json:"type"`
	// Token           json.RawMessage `json:"token"`
	// Subdenomination json.RawMessage `json:"subdenomination"`
	// ArgumentTypes   json.RawMessage `json:"argumentTypes"`
}

// ImportDirective represents an import declaration in a Solidity AST.
type ImportDirective struct {
	header        Header
	SourceUnit    json.RawMessage `json:"SourceUnit"`
	AbsolutePath  json.RawMessage `json:"absolutePath"`
	File          json.RawMessage `json:"file"`
	Scope         json.RawMessage `json:"scope"`
	SymbolAliases json.RawMessage `json:"symbolAliases"`
	UnitAlias     json.RawMessage `json:"unitAlias"`
}

// IgnoredNode represents a node that we ignored in a Solidity AST.
type IgnoredNode struct {
	header Header
}

// Block represents a function's command-list in a Solidity AST.
type Block struct {
	header Header
}

// EventDefinition represents an even definition in a Solidity AST.
type EventDefinition struct {
	header        Header
	children      []Node
	CanonicalName string `json:"canonicalName"` // NOTE: not present in json file, added in post.
	Name          string `json:"name"`
}

func (n SourceUnit) Header() Header           { return n.header }
func (n PragmaDirective) Header() Header      { return n.header }
func (n ContractDefinition) Header() Header   { return n.header }
func (n StructDefinition) Header() Header     { return n.header }
func (n VariableDeclaration) Header() Header  { return n.header }
func (n ElementaryTypeName) Header() Header   { return n.header }
func (n ModifierDefinition) Header() Header   { return n.header }
func (n ParameterList) Header() Header        { return n.header }
func (n FunctionDefinition) Header() Header   { return n.header }
func (n UserDefinedTypeName) Header() Header  { return n.header }
func (n ModifierInvocation) Header() Header   { return n.header }
func (n Identifier) Header() Header           { return n.header }
func (n InheritanceSpecifier) Header() Header { return n.header }
func (n EnumDefinition) Header() Header       { return n.header }
func (n EnumValue) Header() Header            { return n.header }
func (n Block) Header() Header                { return n.header }
func (n Mapping) Header() Header              { return n.header }
func (n ArrayTypeName) Header() Header        { return n.header }
func (n UsingForDirective) Header() Header    { return n.header }
func (n Literal) Header() Header              { return n.header }
func (n ImportDirective) Header() Header      { return n.header }
func (n IgnoredNode) Header() Header          { return n.header }
func (n EventDefinition) Header() Header      { return n.header }

func (n SourceUnit) Children() []Node           { return n.children }
func (n PragmaDirective) Children() []Node      { return n.children }
func (n ContractDefinition) Children() []Node   { return n.children }
func (n StructDefinition) Children() []Node     { return n.children }
func (n VariableDeclaration) Children() []Node  { return n.children }
func (n ElementaryTypeName) Children() []Node   { return nil }
func (n ModifierDefinition) Children() []Node   { return n.children }
func (n ParameterList) Children() []Node        { return n.children }
func (n FunctionDefinition) Children() []Node   { return n.children }
func (n UserDefinedTypeName) Children() []Node  { return n.children }
func (n ModifierInvocation) Children() []Node   { return n.children }
func (n Identifier) Children() []Node           { return n.children }
func (n InheritanceSpecifier) Children() []Node { return n.children }
func (n EnumDefinition) Children() []Node       { return n.children }
func (n EnumValue) Children() []Node            { return n.children }
func (n Block) Children() []Node                { return nil }
func (n Mapping) Children() []Node              { return n.children }
func (n ArrayTypeName) Children() []Node        { return n.children }
func (n UsingForDirective) Children() []Node    { return n.children }
func (n Literal) Children() []Node              { return nil }
func (n ImportDirective) Children() []Node      { return nil }
func (n IgnoredNode) Children() []Node          { return nil }
func (n EventDefinition) Children() []Node      { return n.children }

// PreTraverse traverses a Node-tree in pre-order.
func PreTraverse(root Node, f func(Node)) {
	f(root)
	for _, child := range root.Children() {
		PreTraverse(child, f)
	}
}

// PostTraverse traverses a Node-tree in post-order.
func PostTraverse(root Node, f func(Node)) {
	for _, child := range root.Children() {
		PostTraverse(child, f)
	}
	f(root)
}

// UnserializeJSON parses a raw JSON AST representation into a Node tree.
func UnserializeJSON(raw json.RawMessage) (Node, error) {
	header := Header{}
	if e := json.Unmarshal(raw, &header); e != nil {
		return nil, e
	}
	switch header.Name {
	case "SourceUnit":
		sourceUnit := SourceUnit{header: header}
		if e := json.Unmarshal(header.Attributes, &sourceUnit); e != nil {
			return nil, e
		}
		for _, child := range header.Children {
			u, e := UnserializeJSON(child)
			if e != nil {
				return nil, e
			}
			sourceUnit.children = append(sourceUnit.children, u)
		}
		return sourceUnit, nil

	case "PragmaDirective":
		pragmaDirective := PragmaDirective{header: header}
		if e := json.Unmarshal(header.Attributes, &pragmaDirective); e != nil {
			return nil, e
		}
		for _, child := range header.Children {
			u, e := UnserializeJSON(child)
			if e != nil {
				return nil, e
			}
			pragmaDirective.children = append(pragmaDirective.children, u)
		}
		return pragmaDirective, nil

	case "ContractDefinition":
		contractDefinition := ContractDefinition{header: header}
		if e := json.Unmarshal(header.Attributes, &contractDefinition); e != nil {
			return nil, e
		}
		for _, child := range header.Children {
			u, e := UnserializeJSON(child)
			if e != nil {
				return nil, e
			}
			contractDefinition.children = append(contractDefinition.children, u)
		}
		return contractDefinition, nil

	case "EventDefinition":
		eventDefinition := EventDefinition{header: header}
		if e := json.Unmarshal(header.Attributes, &eventDefinition); e != nil {
			return nil, e
		}
		for _, child := range header.Children {
			u, e := UnserializeJSON(child)
			if e != nil {
				return nil, e
			}
			eventDefinition.children = append(eventDefinition.children, u)
		}
		return eventDefinition, nil

	case "StructDefinition":
		structDefinition := StructDefinition{header: header}
		if e := json.Unmarshal(header.Attributes, &structDefinition); e != nil {
			return nil, e
		}
		for _, child := range header.Children {
			u, e := UnserializeJSON(child)
			if e != nil {
				return nil, e
			}
			structDefinition.children = append(structDefinition.children, u)
		}
		return structDefinition, nil

	case "VariableDeclaration":
		variableDeclaration := VariableDeclaration{header: header}
		if e := json.Unmarshal(header.Attributes, &variableDeclaration); e != nil {
			return nil, e
		}
		for _, child := range header.Children {
			u, e := UnserializeJSON(child)
			if e != nil {
				return nil, e
			}
			variableDeclaration.children = append(variableDeclaration.children, u)
		}
		return variableDeclaration, nil

	case "ElementaryTypeName":
		elementaryTypeName := ElementaryTypeName{header: header}
		if e := json.Unmarshal(header.Attributes, &elementaryTypeName); e != nil {
			return nil, e
		}
		return elementaryTypeName, nil

	case "ModifierDefinition":
		modifierDefinition := ModifierDefinition{header: header}
		if e := json.Unmarshal(header.Attributes, &modifierDefinition); e != nil {
			return nil, e
		}
		for _, child := range header.Children {
			u, e := UnserializeJSON(child)
			if e != nil {
				return nil, e
			}
			modifierDefinition.children = append(modifierDefinition.children, u)
		}
		return modifierDefinition, nil

	case "ParameterList":
		parameterList := ParameterList{header: header}
		for _, child := range header.Children {
			u, e := UnserializeJSON(child)
			if e != nil {
				return nil, e
			}
			parameterList.children = append(parameterList.children, u)
		}
		return parameterList, nil

	case "FunctionDefinition":
		functionDefinition := FunctionDefinition{}
		if e := json.Unmarshal(header.Attributes, &functionDefinition); e != nil {
			return nil, e
		}
		for _, child := range header.Children {
			u, e := UnserializeJSON(child)
			if e != nil {
				return nil, e
			}
			functionDefinition.children = append(functionDefinition.children, u)
		}
		return functionDefinition, nil

	case "ModifierInvocation":
		modifierInvocation := ModifierInvocation{header: header}
		if e := json.Unmarshal(header.Attributes, &modifierInvocation); e != nil {
			return nil, e
		}
		for _, child := range header.Children {
			u, e := UnserializeJSON(child)
			if e != nil {
				return nil, e
			}
			modifierInvocation.children = append(modifierInvocation.children, u)
		}
		return modifierInvocation, nil

	case "UserDefinedTypeName":
		userDefinedTypeName := UserDefinedTypeName{header: header}
		if e := json.Unmarshal(header.Attributes, &userDefinedTypeName); e != nil {
			return nil, e
		}
		return userDefinedTypeName, nil

	case "Identifier":
		identifier := Identifier{header: header}
		if e := json.Unmarshal(header.Attributes, &identifier); e != nil {
			return nil, e
		}
		for _, child := range header.Children {
			u, e := UnserializeJSON(child)
			if e != nil {
				return nil, e
			}
			identifier.children = append(identifier.children, u)
		}
		return identifier, nil

	case "InheritanceSpecifier":
		inheritanceSpecifier := InheritanceSpecifier{header: header}
		for _, child := range header.Children {
			u, e := UnserializeJSON(child)
			if e != nil {
				return nil, e
			}
			inheritanceSpecifier.children = append(inheritanceSpecifier.children, u)
		}
		return inheritanceSpecifier, nil

	case "UsingForDirective":
		usingForDirective := UsingForDirective{header: header}
		for _, child := range header.Children {
			u, e := UnserializeJSON(child)
			if e != nil {
				return nil, e
			}
			usingForDirective.children = append(usingForDirective.children, u)
		}
		return usingForDirective, nil

	case "EnumDefinition":
		enumDefinition := EnumDefinition{header: header}
		if e := json.Unmarshal(header.Attributes, &enumDefinition); e != nil {
			return nil, e
		}
		for _, child := range header.Children {
			u, e := UnserializeJSON(child)
			if e != nil {
				return nil, e
			}
			enumDefinition.children = append(enumDefinition.children, u)
		}
		return enumDefinition, nil

	case "Mapping":
		mapping := Mapping{header: header}
		if e := json.Unmarshal(header.Attributes, &mapping); e != nil {
			return nil, e
		}
		for _, child := range header.Children {
			u, e := UnserializeJSON(child)
			if e != nil {
				return nil, e
			}
			mapping.children = append(mapping.children, u)
		}
		return mapping, nil

	case "ArrayTypeName":
		arrayTypeName := ArrayTypeName{header: header}
		if e := json.Unmarshal(header.Attributes, &arrayTypeName); e != nil {
			return nil, e
		}
		for _, child := range header.Children {
			u, e := UnserializeJSON(child)
			if e != nil {
				return nil, e
			}
			arrayTypeName.children = append(arrayTypeName.children, u)
		}
		return arrayTypeName, nil

	case "ImportDirective":
		importDirective := ImportDirective{header: header}
		if e := json.Unmarshal(header.Attributes, &importDirective); e != nil {
			return nil, e
		}
		return importDirective, nil

	case "EnumValue":
		enumValue := EnumValue{header: header}
		if e := json.Unmarshal(header.Attributes, &enumValue); e != nil {
			return nil, e
		}
		return enumValue, nil

	case "Literal":
		literal := Literal{header: header}
		if e := json.Unmarshal(header.Attributes, &literal); e != nil {
			return nil, e
		}
		return literal, nil

	case "Block":
		return Block{header: header}, nil

	}
	log.Println("ignoring AST node type:", header.Name)
	return IgnoredNode{header: header}, nil
}
