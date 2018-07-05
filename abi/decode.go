// Copyright 2018 karma.run AG. All rights reserved.
package abi // import "github.com/karmarun/karma.link/abi"

import (
	"encoding/json"
	"fmt"
	"github.com/karmarun/karma.link/types"
	"math/big"
	"strconv"
	"strings"
	"unicode/utf8"
)

func Decode(typ types.Type, code Code) (json.RawMessage, error) {
	value, _, e := decode(typ, code, 0)
	if e != nil {
		return nil, e
	}
	return value, nil
}

// offset = offset from head start
// parsed, remainder, error
func decode(typ types.Type, code Code, offset int) (json.RawMessage, Code, error) {
	switch t := typ.(type) {

	case types.Named:
		return decode(t.Type, code, offset)

	case types.ContractAddress:
		return decode(addressType, code, offset)

	case types.InterfaceAddress:
		return decode(addressType, code, offset)

	case types.LibraryAddress:
		return decode(addressType, code, offset)

	case types.Enum:
		idx := new(big.Int).SetBytes(code[:32]).Int64()
		bs, _ := json.Marshal(t[idx])
		return bs, code[32:], nil

	case types.Tuple:
		out := make([]json.RawMessage, len(t), len(t))
		for i, typ := range t {
			p, c, e := decode(typ, code, offset)
			if e != nil {
				return nil, nil, e
			}
			offset += len(code) - len(c)
			out[i], code = p, c
		}
		bs, _ := json.Marshal(out)
		return bs, code, nil

	case types.Struct:
		out := make(map[string]json.RawMessage, len(t.Keys))
		for i, key := range t.Keys {
			typ := t.Types[i]
			p, c, e := decode(typ, code, offset)
			if e != nil {
				return nil, nil, e
			}
			offset += len(code) - len(c)
			out[key], code = p, c
		}
		bs, _ := json.Marshal(out)
		return bs, code, nil

	case types.Array:

		if t.IsDynamic() {
			ref := int(new(big.Int).SetBytes(code[:32]).Int64())
			tail := code[ref-offset:]
			lng := int(new(big.Int).SetBytes(tail[:32]).Int64())
			tuple := make(types.Tuple, lng, lng)
			for i := 0; i < lng; i++ {
				tuple[i] = t.Type
			}
			val, _, e := decode(tuple, tail[32:], 0) // NOTE: reset offset (multi-dimensional case)
			if e != nil {
				return nil, nil, e
			}
			return val, code[32:], nil
		}

		if t.Length == 0 {
			return json.RawMessage(`[]`), code, nil
		}

		out := make([]json.RawMessage, t.Length, t.Length)
		for i := 0; i < t.Length; i++ {
			p, c, e := decode(t.Type, code, offset)
			if e != nil {
				return nil, nil, e
			}
			offset += len(code) - len(c)
			out[i], code = p, c
		}
		bs, _ := json.Marshal(out)
		return bs, code, nil

	case types.Elementary:
		id := string(normalizeElementaryTypeName(t))
		if strings.HasPrefix(id, `fixed`) || strings.HasPrefix(id, `ufixed`) {
			// TODO: support fixed<M>x<N> and ufixed<M>x<N>
			return nil, nil, fmt.Errorf(`fixed/ufixed types not supported yet`)
		}
		if strings.HasPrefix(id, `uint`) {
			val := new(big.Int).SetBytes(code[:32])
			if val.BitLen() <= 32 {
				return json.RawMessage(val.Text(10)), code[32:], nil
			}
			return json.RawMessage(`"0x` + val.Text(16) + `"`), code[32:], nil
		}
		if strings.HasPrefix(id, `int`) {
			uval := new(big.Int).SetBytes(code[:32])
			if uval.Bit(255) == 0 {
				if uval.BitLen() <= 32 {
					return json.RawMessage(uval.Text(10)), code[32:], nil
				}
				return json.RawMessage(`"0x` + uval.Text(16) + `"`), code[32:], nil
			}
			sval := new(big.Int).SetBytes(manualTwosComplement(code[:32]))
			sval = sval.Neg(sval)
			if sval.BitLen() > 32 {
				return json.RawMessage(`"0x` + uval.Text(16) + `"`), code[32:], nil
			}
			return json.RawMessage(sval.Text(10)), code[32:], nil
		}
		if id == `bytes` {
			ref := int(new(big.Int).SetBytes(code[:32]).Int64())
			tail := code[ref-offset:]
			lng := int(new(big.Int).SetBytes(tail[:32]).Int64())
			bs := tail[32 : 32+lng]
			if utf8.Valid(bs) {
				val, _ := json.Marshal(string(bs))
				return val, code[32:], nil
			}
			val, _ := json.Marshal(bs)
			return val, code[32:], nil
		}
		if id != `bytes` && strings.HasPrefix(id, `bytes`) { // bytes1, bytes2, ... bytes32
			n, e := strconv.Atoi(id[len(`bytes`):])
			if e != nil || n < 0 || n > 32 {
				logger.Panicln(n, e)
			}
			bs := code[:32]
			code = code[32:]
			bs = bs[:n]
			if utf8.Valid(bs) {
				val, _ := json.Marshal(string(bs))
				return val, code, nil
			}
			val, _ := json.Marshal(bs)
			return val, code, nil
		}

	}
	logger.Panicf("unexpected type in abi.Decode: %#v\n", typ)
	return nil, nil, nil // shut up compiler
}
