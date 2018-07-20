// Copyright 2018 karma.run AG. All rights reserved.

package auth // import "github.com/karmarun/karma.link/auth"

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
	"sync"
)

// Key associates an ecdsa.PrivateKey with an Ethereum address.
type Key struct {
	Address    common.Address
	PrivateKey *ecdsa.PrivateKey
}

var (
	zeroWords    = make([]big.Word, 32, 32)
	zeroKeyBytes = KeyBytes{}
)

// Destroy erases the private key from memory, overwriting it with zeroes.
// Auth clients must always safely dispose of keys this way.
func (k *Key) Destroy() {
	k.Address = common.Address{}
	DestroyEcdsaPrivateKey(k.PrivateKey)
}

// Authenticator is the interface implemented by authentication providers.
type Authenticator interface {

	// Authenticate validates a JSON-encoded credential structure and returns a JSON-encoded bearer token.
	// If the credentials are either wrong or structurally invalid, Authenticate should return a non-nil error.
	Authenticate(credentials json.RawMessage) (token json.RawMessage, e error)

	// RenewToken exchanges an existing token (commonly the one returned by Authenticate) for a new token with a new life time.
	RenewToken(oldToken json.RawMessage) (newToken json.RawMessage, e error)

	// ExchangeToken validates a JSON-encoded token and exchanges it for a *Key if its valid.
	// Otherwise it should return a non-nil error.
	ExchangeToken(token json.RawMessage) (*Key, error)
}

var registered = &sync.Map{}

// RegisterAuthenticator registers an authenticator under the given name.
// It panics if there already is an Authenticator registered with the same name.
func RegisterAuthenticator(name string, implementation Authenticator) {
	if _, loaded := registered.LoadOrStore(name, implementation); loaded {
		panic(`already registered authenticator with name: ` + name)
	}
}

// Authenticate uses the Authenticator registered as name to authenticate credentials.
// It panics if there is no Authenticator registered under name.
func Authenticate(name string, credentials json.RawMessage) (json.RawMessage, error) {
	authenticator, ok := registered.Load(name)
	if !ok {
		return nil, fmt.Errorf(`no authenticator registered with name: %s`, name)
	}
	return authenticator.(Authenticator).Authenticate(credentials)
}

// ExchangeToken uses the Authenticator registered as name to exchange token for a key.
// It panics if there is no Authenticator registered under name.
func ExchangeToken(name string, token json.RawMessage) (*Key, error) {
	authenticator, ok := registered.Load(name)
	if !ok {
		return nil, fmt.Errorf(`no authenticator registered with name: %s`, name)
	}
	return authenticator.(Authenticator).ExchangeToken(token)
}

const KeyBytesLen = 32*3 + 20

// KeyBytes represents a byte-serialized Key
type KeyBytes [KeyBytesLen]byte

// Destroy overwrites the *KeyBytes' backing storage with zeroes.
func (bs *KeyBytes) Destroy() {
	copy(bs[:], zeroKeyBytes[:])
}

// KeyToBytes converts a *Key to *KeyBytes and calls Destroy() on key.
func KeyToBytes(key *Key) *KeyBytes {
	bs := KeyBytes{}
	priv := key.PrivateKey
	x, y, d := priv.X.Bytes(), priv.Y.Bytes(), priv.D.Bytes()
	copy(bs[0+32-len(x):], x)
	copy(bs[32+32-len(y):], y)
	copy(bs[64+32-len(d):], d)
	copy(bs[32*3:], key.Address[:])
	key.Destroy()
	return &bs
}

var zeroBytes = make([]byte, KeyBytesLen, KeyBytesLen)

// BytesToKey converts *KeyBytes to a *Key and calls Destroy() on bs.
func BytesToKey(bs *KeyBytes) *Key {
	if len(bs) != KeyBytesLen {
		panic("BytesToPrivateKey called with byte slice of wrong length, must be KeyBytesLen")
	}
	x, y, d := new(big.Int).SetBytes(bs[0*32:1*32]), new(big.Int).SetBytes(bs[1*32:2*32]), new(big.Int).SetBytes(bs[2*32:3*32])
	addr := common.Address{}
	copy(addr[:], bs[32*3:])
	bs.Destroy()
	pub := ecdsa.PublicKey{X: x, Y: y, Curve: crypto.S256()}
	return &Key{Address: addr, PrivateKey: &ecdsa.PrivateKey{PublicKey: pub, D: d}}
}

// DestroyEcdsaPrivateKey overwrites key's backing storage with zeroes.
func DestroyEcdsaPrivateKey(key *ecdsa.PrivateKey) {
	for _, words := range [][]big.Word{
		key.D.Bits(),
		key.X.Bits(),
		key.Y.Bits(),
	} {
		copy(words[:cap(words)], zeroWords)
	}
}
