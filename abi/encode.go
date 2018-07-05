package abi

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"types"
)

func Encode(typ types.Type, arg json.RawMessage) (Code, error) {
	head, tail, e := encode(typ, arg, 0, make([]byte, 0, 1024), make([]byte, 0, 1024))
	if e != nil {
		return nil, e
	}
	return append(head, tail...), nil
}

func encode(typ types.Type, arg json.RawMessage, tailOffset int, head, tail []byte) ([]byte, []byte, error) {

	switch t := typ.(type) {

	case types.Named:
		return encode(t.Type, arg, tailOffset, head, tail)

	case types.ContractAddress:
		return encode(addressType, arg, tailOffset, head, tail)

	case types.InterfaceAddress:
		return encode(addressType, arg, tailOffset, head, tail)

	case types.LibraryAddress:
		return encode(addressType, arg, tailOffset, head, tail)

	case types.Tuple:
		temp := make([]json.RawMessage, 0, len(t))
		if e := json.Unmarshal(arg, &temp); e != nil {
			return nil, nil, fmt.Errorf(`expected array of %d elements`, len(t))
		}
		if len(temp) != len(t) {
			return nil, nil, fmt.Errorf(`expected array of %d elements, have %d`, len(t), len(temp)) // TODO: pathed errors
		}
		// tuples are function argument lists, they determine the tail offset
		tailOffset += width(t)
		for i, typ := range t {
			h, t, e := encode(typ, temp[i], tailOffset, head, tail)
			if e != nil {
				return nil, nil, fmt.Errorf(`[%d] %s`, i, e)
			}
			head, tail = h, t
		}
		return head, tail, nil

	case types.Enum:
		temp := ""
		if e := json.Unmarshal(arg, &temp); e != nil {
			return nil, nil, fmt.Errorf(`expected string`)
		}
		idx := -1
		for i, name := range t {
			if temp == name {
				idx = i
				break
			}
		}
		if idx == -1 {
			return nil, nil, fmt.Errorf(`unexpected enum case: %s, expected one of: %s`, temp, strings.Join([]string(t), ", "))
		}
		return append(head, encodeInt256(big.NewInt(int64(idx)))...), tail, nil

	case types.Struct:
		temp := make(map[string]json.RawMessage, len(t.Types))
		if e := json.Unmarshal(arg, &temp); e != nil {
			return nil, nil, fmt.Errorf(`expected object`)
		}
		if len(temp) != len(t.Keys) {
			return nil, nil, fmt.Errorf(`too many or too few keys in object: %d, expected keys: %s`, len(temp), strings.Join(t.Keys, ", "))
		}
		for i, key := range t.Keys {
			if _, ok := temp[key]; !ok {
				return nil, nil, fmt.Errorf(`missing key in object: %s`, key)
			}
			typ := t.Types[i]
			h, t, e := encode(typ, temp[key], tailOffset, head, tail)
			if e != nil {
				return nil, nil, fmt.Errorf(`["%s"] %s`, key, e)
			}
			head, tail = h, t
		}
		return head, tail, nil

	case types.Array: // TODO: support passing strings for e.g. byte[]

		temp := make([]json.RawMessage, 0, maxInt(t.Length, 0))
		if e := json.Unmarshal(arg, &temp); e != nil {
			return nil, nil, fmt.Errorf(`expected array`)
		}
		if t.IsDynamic() {

			// offset -> length, args...
			head = append(head, encodeInt256(big.NewInt(int64(tailOffset+len(tail))))...)
			tail = append(tail, encodeInt256(big.NewInt(int64(len(temp))))...)

			itemWidth := width(t.Type)
			subHead := make([]byte, 0, itemWidth*len(temp))
			subTailOffset := itemWidth * len(temp) // offsets are relative, mirroring remix.ethereum.org
			subTail := make([]byte, 0, 1024)

			for i, arg := range temp {
				h, t, e := encode(t.Type, arg, subTailOffset, subHead, subTail)
				if e != nil {
					return nil, nil, fmt.Errorf(`[%d] %s`, i, e)
				}
				subHead, subTail = h, t
			}

			if len(subHead) != cap(subHead) {
				logger.Panicln(len(subHead), cap(subHead))
			}

			return head, append(tail, append(subHead, subTail...)...), nil

		}
		// fixed-size case
		if t.Length != len(temp) {
			return nil, nil, fmt.Errorf(`expected array of length %d, have %d elements`, t.Length, len(temp))
		}
		for i, arg := range temp {
			h, t, e := encode(t.Type, arg, tailOffset, head, tail)
			if e != nil {
				return nil, nil, fmt.Errorf(`[%d] %s`, i, e)
			}
			head, tail = h, t
		}
		return head, tail, nil

	case types.Elementary:
		id := string(normalizeElementaryTypeName(t))
		if strings.HasPrefix(id, `fixed`) || strings.HasPrefix(id, `ufixed`) {
			// TODO: support fixed<M>x<N> and ufixed<M>x<N>
			return nil, nil, fmt.Errorf(`fixed/ufixed types not supported yet`)
		}
		// TODO: suspected bug in large integers; differenciate between int and uint
		if strings.HasPrefix(id, `int`) || strings.HasPrefix(id, `uint`) {
			bits := 0
			{
				prefixLength := 3
				if id[0] == 'u' {
					prefixLength++
				}
				n, e := strconv.Atoi(id[prefixLength:])
				if e != nil {
					logger.Panicln(e)
				}
				if n%8 != 0 {
					logger.Panicln("precondition violation: n%8 != 0")
				}
				bits = n
			}
			str, base, val := "", 0, big.NewInt(0)
			// either JSON number or "0x..." string
			switch peekNonWhitespaceByte(arg) {
			case '"':
				temp := ""
				if e := json.Unmarshal(arg, &temp); e != nil {
					return nil, nil, fmt.Errorf(`invalid JSON string`)
				}
				if !strings.HasPrefix(temp, `0x`) {
					return nil, nil, fmt.Errorf(`expected "0x" prefix on %s string.`, typ)
				}
				str, base = string(temp), 0

			case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				temp := json.Number("")
				if e := json.Unmarshal(arg, &temp); e != nil {
					return nil, nil, fmt.Errorf(`invalid JSON number`)
				}
				if strings.ContainsAny(string(temp), `eE.`) {
					return nil, nil, fmt.Errorf(`unexpected exponent or decimal separator in number: %s`, temp)
				}
				str, base = string(temp), 10

			default:
				return nil, nil, fmt.Errorf(`expected JSON string or number`)
			}
			if _, ok := val.SetString(str, base); !ok {
				return nil, nil, fmt.Errorf(`invalid hex number for type %s: %s`, typ, str)
			}
			if bs := val.Bytes(); len(bs) > (bits / 8) {
				return nil, nil, fmt.Errorf(`value too large for type %s: %s`, typ, str)
			}
			return append(head, encodeInt256(val)...), tail, nil

		}
		if id == `bytes` {
			bytes := ([]byte)(nil)
			// arg is either array of numbers or string
			switch peekNonWhitespaceByte(arg) {
			case '[':
				temp := make([]byte, 0, 32)
				if e := json.Unmarshal(arg, &temp); e != nil {
					return nil, nil, fmt.Errorf(`invalid byte array`)
				}
				bytes = temp

			case '"':
				temp := ""
				if e := json.Unmarshal(arg, &temp); e != nil {
					return nil, nil, fmt.Errorf(`invalid JSON string`)
				}
				bytes = []byte(temp)

			default:
				return nil, nil, fmt.Errorf(`expected string or array of numbers`)
			}

			length := big.NewInt(int64(len(bytes)))
			padded := append(bytes, make([]byte, 32-len(bytes)%32, 32-len(bytes)%32)...)
			offset := big.NewInt(int64(tailOffset + len(tail)))

			tail = append(tail, encodeInt256(length)...)
			tail = append(tail, padded...)
			head = append(head, encodeInt256(offset)...)

			return head, tail, nil

		}
		if id != `bytes` && strings.HasPrefix(id, `bytes`) { // bytes1, bytes2, ... bytes32
			n, e := strconv.Atoi(id[len(`bytes`):])
			if e != nil || n < 0 || n > 32 {
				logger.Panicln(n, e)
			}
			// arg is either array of numbers or string
			switch peekNonWhitespaceByte(arg) {
			case '[':
				temp := make([]byte, 0, 32)
				if e := json.Unmarshal(arg, &temp); e != nil {
					return nil, nil, fmt.Errorf(`invalid array of bytes`)
				}
				if len(temp) != n {
					return nil, nil, fmt.Errorf(`expected array of length %d, got %d elements`, n, len(temp))
				}
				return append(head, temp[:32]...), tail, nil

			case '"':
				temp := ""
				if e := json.Unmarshal(arg, &temp); e != nil {
					return nil, nil, fmt.Errorf(`invalid JSON string`)
				}
				bytes := []byte(temp)
				if len(bytes) > n {
					return nil, nil, fmt.Errorf(`string too long for %s`, typ)
				}
				out := make([]byte, 32, 32)
				copy(out, bytes)
				return append(head, out...), tail, nil

			default:
				return nil, nil, fmt.Errorf(`expected string or array of numbers`)
			}
		}

	}
	logger.Panicf("unexpected type in abi.Encode: %T\n", typ)
	return nil, nil, nil // shut up compiler
}
