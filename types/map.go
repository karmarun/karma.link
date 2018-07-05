// Copyright 2018 karma.run AG. All rights reserved.
package types

type Map map[Reference]Type

func (m Map) Deref(ref Reference) Type {
	typ, ok := m[ref]
	if !ok {
		panic("missing typemap key")
	}
	if ref, ok := typ.(Reference); ok {
		return m.Deref(ref)
	}
	return typ
}
