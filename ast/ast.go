package ast

import (
	"encoding/json"
	"log"
)

type ContractKind string

const (
	ContractKindContract  ContractKind = "contract"
	ContractKindInterface              = "interface"
	ContractKindLibrary                = "library"
)

type Visibility string

const (
	VisibilityPublic   Visibility = "public"
	VisibilityInternal            = "internal"
	VisibilityExternal            = "external"
	VisibilityPrivate             = "private"
)

type StateMutability string

const (
	StateMutabilityPure       StateMutability = "pure"
	StateMutabilityView                       = "view"
	StateMutabilityNonpayable                 = "nonpayable"
)

type StorageLocation string

const (
	StorageLocationDefault StorageLocation = "default"
	StorageLocationMemory                  = "memory"
	StorageLocationStorage                 = "storage"
)

type CompiledContract struct {
	Binary string `json:"bin"`
}

type Combined struct {
	Contracts  map[string]CompiledContract `json:"contracts"`
	SourceList []string                    `json:"sourceList"`
	Sources    map[string]CombinedSource   `json:"sources"`
	Version    string                      `json:"version"`
}

type CombinedSource struct {
	AST json.RawMessage `json:"AST"`
}

type Header struct {
	Id         int               `json:"id"`
	Name       string            `json:"name"`
	Source     string            `json:"src"`
	Attributes json.RawMessage   `json:"attributes"`
	Children   []json.RawMessage `json:"children"`
}
type Node interface {
	Header() Header
	Children() []Node
}

type SourceUnit struct {
	header          Header
	children        []Node
	AbsolutePath    string           `json:"absolutePath"`
	ExportedSymbols map[string][]int `json:"exportedSymbols"`
}

type PragmaDirective struct {
	header   Header
	children []Node
	Literals []string `json:"literals"`
}

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

type StructDefinition struct {
	header        Header
	children      []Node
	CanonicalName string     `json:"canonicalName"`
	Name          string     `json:"name"`
	Scope         int        `json:"scope"`
	Visibility    Visibility `json:"visibility"`
}

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

type ElementaryTypeName struct {
	header Header
	Name   string `json:"name"`
	Type   string `json:"type"`
}

type ModifierDefinition struct {
	header     Header
	children   []Node
	Name       string     `json:"name"`
	Visibility Visibility `json:"visibility"`
	// Documentation json.RawMessage `json:"documentation"`
}

type ParameterList struct {
	header   Header
	children []Node
}

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

type UserDefinedTypeName struct {
	header                Header
	children              []Node
	Name                  string `json:"name"`
	ReferencedDeclaration int    `json:"referencedDeclaration"`
	Type                  string `json:"type"`
	// ContractScope         json.RawMessage `json:"contractScope"`
}

type ModifierInvocation struct {
	header   Header
	children []Node
	// Arguments json.RawMessage `json:"arguments"`
}

type Identifier struct {
	header                Header
	children              []Node
	Value                 string `json:"value"`
	ReferencedDeclaration int    `json:"referencedDeclaration"`
	// ArgumentTypes          json.RawMessage `json:"argumentTypes"`
	// OverloadedDeclarations json.RawMessage `json:"overloadedDeclarations"`
	// Type                   json.RawMessage `json:"type"`
}

type InheritanceSpecifier struct {
	header   Header
	children []Node
	// Arguments json.RawMessage `json:"arguments"`
}

type EnumDefinition struct {
	header        Header
	children      []Node
	CanonicalName string `json:"canonicalName"`
	Name          string `json:"name"`
}

type EnumValue struct {
	header   Header
	children []Node
	Name     string `json:"name"`
}

type Mapping struct {
	header   Header
	children []Node
	Type     string `json:"type"`
}

type ArrayTypeName struct {
	header   Header
	children []Node
	Type     string `json:"type"`
}

type UsingForDirective struct {
	header   Header
	children []Node
}

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

type ImportDirective struct {
	header        Header
	SourceUnit    json.RawMessage `json:"SourceUnit"`
	AbsolutePath  json.RawMessage `json:"absolutePath"`
	File          json.RawMessage `json:"file"`
	Scope         json.RawMessage `json:"scope"`
	SymbolAliases json.RawMessage `json:"symbolAliases"`
	UnitAlias     json.RawMessage `json:"unitAlias"`
}

type IgnoredNode struct {
	header Header
}

type Block struct {
	header Header
}

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

func PreTraverse(root Node, f func(Node)) {
	f(root)
	for _, child := range root.Children() {
		PreTraverse(child, f)
	}
}

func PostTraverse(root Node, f func(Node)) {
	for _, child := range root.Children() {
		PostTraverse(child, f)
	}
	f(root)
}

func UnserializeJSON(raw json.RawMessage) Node {
	header := Header{}
	if e := json.Unmarshal(raw, &header); e != nil {
		log.Fatalln(e)
	}
	switch header.Name {
	case "SourceUnit":
		sourceUnit := SourceUnit{header: header}
		if e := json.Unmarshal(header.Attributes, &sourceUnit); e != nil {
			log.Fatalln(e)
		}
		for _, child := range header.Children {
			sourceUnit.children = append(sourceUnit.children, UnserializeJSON(child))
		}
		return sourceUnit

	case "PragmaDirective":
		pragmaDirective := PragmaDirective{header: header}
		if e := json.Unmarshal(header.Attributes, &pragmaDirective); e != nil {
			log.Fatalln(e)
		}
		for _, child := range header.Children {
			pragmaDirective.children = append(pragmaDirective.children, UnserializeJSON(child))
		}
		return pragmaDirective

	case "ContractDefinition":
		contractDefinition := ContractDefinition{header: header}
		if e := json.Unmarshal(header.Attributes, &contractDefinition); e != nil {
			log.Fatalln(e)
		}
		for _, child := range header.Children {
			contractDefinition.children = append(contractDefinition.children, UnserializeJSON(child))
		}
		return contractDefinition

	case "EventDefinition":
		eventDefinition := EventDefinition{header: header}
		if e := json.Unmarshal(header.Attributes, &eventDefinition); e != nil {
			log.Fatalln(e)
		}
		for _, child := range header.Children {
			eventDefinition.children = append(eventDefinition.children, UnserializeJSON(child))
		}
		return eventDefinition

	case "StructDefinition":
		structDefinition := StructDefinition{header: header}
		if e := json.Unmarshal(header.Attributes, &structDefinition); e != nil {
			log.Fatalln(e)
		}
		for _, child := range header.Children {
			structDefinition.children = append(structDefinition.children, UnserializeJSON(child))
		}
		return structDefinition

	case "VariableDeclaration":
		variableDeclaration := VariableDeclaration{header: header}
		if e := json.Unmarshal(header.Attributes, &variableDeclaration); e != nil {
			log.Fatalln(e)
		}
		for _, child := range header.Children {
			variableDeclaration.children = append(variableDeclaration.children, UnserializeJSON(child))
		}
		return variableDeclaration

	case "ElementaryTypeName":
		elementaryTypeName := ElementaryTypeName{header: header}
		if e := json.Unmarshal(header.Attributes, &elementaryTypeName); e != nil {
			log.Fatalln(e)
		}
		return elementaryTypeName

	case "ModifierDefinition":
		modifierDefinition := ModifierDefinition{header: header}
		if e := json.Unmarshal(header.Attributes, &modifierDefinition); e != nil {
			log.Fatalln(e)
		}
		for _, child := range header.Children {
			modifierDefinition.children = append(modifierDefinition.children, UnserializeJSON(child))
		}
		return modifierDefinition

	case "ParameterList":
		parameterList := ParameterList{header: header}
		for _, child := range header.Children {
			parameterList.children = append(parameterList.children, UnserializeJSON(child))
		}
		return parameterList

	case "FunctionDefinition":
		functionDefinition := FunctionDefinition{}
		if e := json.Unmarshal(header.Attributes, &functionDefinition); e != nil {
			log.Fatalln(e)
		}
		for _, child := range header.Children {
			functionDefinition.children = append(functionDefinition.children, UnserializeJSON(child))
		}
		return functionDefinition

	case "ModifierInvocation":
		modifierInvocation := ModifierInvocation{header: header}
		if e := json.Unmarshal(header.Attributes, &modifierInvocation); e != nil {
			log.Fatalln(e)
		}
		for _, child := range header.Children {
			modifierInvocation.children = append(modifierInvocation.children, UnserializeJSON(child))
		}
		return modifierInvocation

	case "UserDefinedTypeName":
		userDefinedTypeName := UserDefinedTypeName{header: header}
		if e := json.Unmarshal(header.Attributes, &userDefinedTypeName); e != nil {
			log.Fatalln(e)
		}
		return userDefinedTypeName

	case "Identifier":
		identifier := Identifier{header: header}
		if e := json.Unmarshal(header.Attributes, &identifier); e != nil {
			log.Fatalln(e)
		}
		for _, child := range header.Children {
			identifier.children = append(identifier.children, UnserializeJSON(child))
		}
		return identifier

	case "InheritanceSpecifier":
		inheritanceSpecifier := InheritanceSpecifier{header: header}
		for _, child := range header.Children {
			inheritanceSpecifier.children = append(inheritanceSpecifier.children, UnserializeJSON(child))
		}
		return inheritanceSpecifier

	case "UsingForDirective":
		usingForDirective := UsingForDirective{header: header}
		for _, child := range header.Children {
			usingForDirective.children = append(usingForDirective.children, UnserializeJSON(child))
		}
		return usingForDirective

	case "EnumDefinition":
		enumDefinition := EnumDefinition{header: header}
		if e := json.Unmarshal(header.Attributes, &enumDefinition); e != nil {
			log.Fatalln(e)
		}
		for _, child := range header.Children {
			enumDefinition.children = append(enumDefinition.children, UnserializeJSON(child))
		}
		return enumDefinition

	case "Mapping":
		mapping := Mapping{header: header}
		if e := json.Unmarshal(header.Attributes, &mapping); e != nil {
			log.Fatalln(e)
		}
		for _, child := range header.Children {
			mapping.children = append(mapping.children, UnserializeJSON(child))
		}
		return mapping

	case "ArrayTypeName":
		arrayTypeName := ArrayTypeName{header: header}
		if e := json.Unmarshal(header.Attributes, &arrayTypeName); e != nil {
			log.Fatalln(e)
		}
		for _, child := range header.Children {
			arrayTypeName.children = append(arrayTypeName.children, UnserializeJSON(child))
		}
		return arrayTypeName

	case "ImportDirective":
		importDirective := ImportDirective{header: header}
		if e := json.Unmarshal(header.Attributes, &importDirective); e != nil {
			log.Fatalln(e)
		}
		return importDirective

	case "EnumValue":
		enumValue := EnumValue{header: header}
		if e := json.Unmarshal(header.Attributes, &enumValue); e != nil {
			log.Fatalln(e)
		}
		return enumValue

	case "Literal":
		literal := Literal{header: header}
		if e := json.Unmarshal(header.Attributes, &literal); e != nil {
			log.Fatalln(e)
		}
		return literal

	case "Block":
		return Block{header: header}

	}
	log.Println("ignoring AST node type:", header.Name)
	return IgnoredNode{header: header}
}
