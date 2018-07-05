package abi

import (
	"config"
	"encoding/json"
	"log"
	"math/big"
	"types"
)

type Code []byte

var logger = log.New(config.LogWriter, `abi`, config.LogFlags)

const addressType = types.Elementary(`address`)

func width(typ types.Type) int {
	switch t := typ.(type) {

	case types.Named:
		return width(t.Type)

	case types.Enum,
		types.ContractAddress,
		types.InterfaceAddress,
		types.LibraryAddress,
		types.Elementary:
		return 32

	case types.Tuple:
		w := 0
		for _, typ := range t {
			w += width(typ)
		}
		return w

	case types.Struct:
		w := 0
		for _, typ := range t.Types {
			w += width(typ)
		}
		return w

	case types.Array:
		if t.Length == types.DynamicArrayLength {
			return 32 // = pointer into tail
		}
		return width(t.Type) * t.Length

	}
	logger.Panicf("unexpected type in width: %T\n", typ)
	return 0 // shut up compiler
}

func peekNonWhitespaceByte(json json.RawMessage) byte {
	for len(json) > 0 && (json[0] == '\t' || json[0] == '\n' || json[0] == '\r' || json[0] == ' ') {
		json = json[1:]
	}
	if len(json) == 0 {
		return 0
	}
	return json[0]
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func encodeInt64(i int64) []byte {
	return encodeInt256(big.NewInt(i))
}

func encodeInt256(i *big.Int) []byte {
	bs := i.Bytes()
	cs := make([]byte, 32-len(bs), 32)
	cs = append(cs, bs...)
	if i.Sign() >= 0 {
		return cs
	}
	return manualTwosComplement(cs)
}

func normalizeElementaryTypeName(id types.Elementary) types.Elementary {
	switch id { // alias mapping, synonyms
	case `byte`:
		return `bytes1`
	case `int`:
		return `int256`
	case `uint`:
		return `uint256`
	case `address`:
		return `uint160`
	case `fixed`:
		return `fixed128x18`
	case `ufixed`:
		return `ufixed128x18`
	case `string`:
		return `bytes`
	}
	return id
}

// bs[0] MSB ... bs[len(bs)-1] LSB
func manualTwosComplement(bs []byte) []byte {
	cs := make([]byte, len(bs), len(bs))
	copy(cs, bs)
	// manual two's-complement
	for i := 0; i < 32; i += 8 {
		cs[i+0] = ^cs[i+0]
		cs[i+1] = ^cs[i+1]
		cs[i+2] = ^cs[i+2]
		cs[i+3] = ^cs[i+3]
		cs[i+4] = ^cs[i+4]
		cs[i+5] = ^cs[i+5]
		cs[i+6] = ^cs[i+6]
		cs[i+7] = ^cs[i+7]
	}
	for i := 31; i > 0; i-- {
		cs[i]++
		if cs[i] != 0 {
			break
		}
	}
	return cs
}
