// Copyright 2018 karma.run AG. All rights reserved.
package types // import "github.com/karmarun/karma.link/types"

import (
	"strconv"
)

type Type interface {
	SoliditySignature() []byte
	Map(func(Type) Type) Type
}

// represents the length of dynamically sized array
// NOTE: Solidity accepts 0-length array types e.g. uint256[0] is valid.
const DynamicArrayLength = -1

type Reference int

func (t Reference) SoliditySignature() []byte {
	panic("types.Reference.SoliditySignature() called!")
}

func (t Reference) Map(f func(Type) Type) Type {
	return f(t)
}

type Elementary string

func (t Elementary) SoliditySignature() []byte {
	return []byte(t)
}

func (t Elementary) Map(f func(Type) Type) Type {
	return f(t)
}

type Event struct {
	Name string
	Args []Type
}

func (t Event) SoliditySignature() []byte {
	bs := []byte(t.Name + `(`)
	for i, subType := range t.Args {
		if i > 0 {
			bs = append(bs, ',')
		}
		bs = append(bs, subType.SoliditySignature()...)
	}
	return append(bs, ')')
}

func (t Event) Map(f func(Type) Type) Type {
	length := len(t.Args)
	args := make([]Type, length, length)
	for i := 0; i < length; i++ {
		args[i] = t.Args[i].Map(f)
	}
	return Event{Name: t.Name, Args: args} // NOTE: no f()
}

type Tuple []Type

func (t Tuple) SoliditySignature() []byte {
	bs := make([]byte, 1, 2+len(t)*8)
	bs[0] = '('
	for i, subType := range t {
		if i > 0 {
			bs = append(bs, ',')
		}
		bs = append(bs, subType.SoliditySignature()...)
	}
	return append(bs, ')')
}

func (t Tuple) Map(f func(Type) Type) Type {
	length := len(t)
	out := make(Tuple, length, length)
	for i := 0; i < length; i++ {
		out[i] = t[i].Map(f)
	}
	return out // NOTE: no f()
}

type Struct struct {
	Keys  []string
	Types []Type
}

func (t Struct) SoliditySignature() []byte {
	bs := make([]byte, 1, 2+len(t.Keys)*8)
	bs[0] = '('
	for i, subType := range t.Types {
		if i > 0 {
			bs = append(bs, ',')
		}
		bs = append(bs, subType.SoliditySignature()...)
	}
	return append(bs, ')')
}

func (t Struct) Map(f func(Type) Type) Type {
	length := len(t.Keys)
	out := Struct{
		Keys:  make([]string, length, length),
		Types: make([]Type, length, length),
	}
	for i := 0; i < length; i++ {
		out.Keys[i], out.Types[i] = t.Keys[i], t.Types[i].Map(f)
	}
	return out // NOTE: no f()
}

type Array struct {
	Length int
	Type   Type
}

func (a Array) IsDynamic() bool {
	return a.Length == DynamicArrayLength
}

func (t Array) SoliditySignature() []byte {
	bs := t.Type.SoliditySignature()
	bs = append(bs, '[')
	if !t.IsDynamic() {
		lenStr := strconv.FormatInt(int64(t.Length), 10)
		bs = append(bs, lenStr...)
	}
	return append(bs, ']')
}

func (t Array) Map(f func(Type) Type) Type {
	return Array{ // NOTE: no f()
		Length: t.Length,
		Type:   t.Type.Map(f),
	}
}

type Mapping struct {
	Key   Type
	Value Type
}

// NOTE: mappings can't be passed as parameters, nevertheless
//       we provide a Signature method for convenience.
func (t Mapping) SoliditySignature() []byte {
	bs := []byte("mapping(")
	bs = append(bs, t.Key.SoliditySignature()...)
	bs = append(bs, " => "...)
	bs = append(bs, t.Value.SoliditySignature()...)
	return append(bs, ')')
}

func (t Mapping) Map(f func(Type) Type) Type {
	return Mapping{ // NOTE: no f()
		Key:   t.Key.Map(f),
		Value: t.Value.Map(f),
	}
}

type Enum []string

func (t Enum) SoliditySignature() []byte {
	return []byte("uint8")
}

func (t Enum) Map(f func(Type) Type) Type {
	return f(t)
}

type Named struct {
	Name string
	Type Type
}

func (t Named) SoliditySignature() []byte {
	return t.Type.SoliditySignature()
}

func (t Named) Map(f func(Type) Type) Type {
	return Named{ // NOTE: no f()
		Name: t.Name,
		Type: t.Type.Map(f),
	}
}

type ContractAddress string

func (t ContractAddress) SoliditySignature() []byte {
	return []byte("address")
}

func (t ContractAddress) Map(f func(Type) Type) Type {
	return f(t)
}

type InterfaceAddress string

func (t InterfaceAddress) SoliditySignature() []byte {
	return []byte("address")
}

func (t InterfaceAddress) Map(f func(Type) Type) Type {
	return f(t)
}

type LibraryAddress string

func (t LibraryAddress) SoliditySignature() []byte {
	return []byte("address")
}

func (t LibraryAddress) Map(f func(Type) Type) Type {
	return f(t)
}
