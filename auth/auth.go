// Copyright 2018 karma.run AG. All rights reserved.

package auth // import "github.com/karmarun/karma.link/auth"

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
	"sync"
	"time"
)

// Key associates an ecdsa.PrivateKey with an Ethereum address.
type Key struct {
	Address    common.Address
	PrivateKey *ecdsa.PrivateKey
}

var (
	zeroWords    = make([]big.Word, 256, 256) // large enough for most keys
	zeroKeyBytes = make(KeyBytes, 256, 256)   // large enough for most keys
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
	// The old token may or may not continue to be valid. Normally not.
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

// KeyBytes represents a byte-serialized Key
type KeyBytes []byte

// Destroy overwrites bs' backing storage with zeroes.
func (bs KeyBytes) Destroy() {
	copy(bs[:cap(bs)], zeroKeyBytes)
}

// Copy copies bs.
func (bs KeyBytes) Copy() KeyBytes {
	cp := make(KeyBytes, len(bs), cap(bs))
	copy(cp, bs)
	return cp
}

// KeyToBytes converts a *Key to KeyBytes and calls Destroy() on key.
func KeyToBytes(key *Key) KeyBytes {
	dump := crypto.FromECDSA(key.PrivateKey)
	key.Destroy()
	return KeyBytes(dump)
}

// BytesToKey converts KeyBytes to a *Key and calls Destroy() on bs (only if successful).
// It returns a non-nil error for invalid KeyBytes.
func BytesToKey(bs KeyBytes) (*Key, error) {
	priv, e := crypto.ToECDSA(bs)
	if e != nil {
		return nil, fmt.Errorf(`invalid private key`)
	}
	bs.Destroy()
	return &Key{Address: crypto.PubkeyToAddress(priv.PublicKey), PrivateKey: priv}, nil
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

// KeyStore implements a quick-and-simple, in-memory encrypted key store.
// It utilizes a one-time pad scheme.
type KeyStore struct {
	store *sync.Map
}

// NewKeyStore makes a new KeyStore
func NewKeyStore() KeyStore {
	return KeyStore{&sync.Map{}}
}

// Write stores plaintext bs for ttl time. If ttl == 0, its stored forever.
// It returns an index as [32]byte and random bit-mask to use in Read.
func (s KeyStore) Write(bs KeyBytes, ttl time.Duration) ([32]byte, []byte) {
	randomness := make([]byte, len(bs)+32, len(bs)+32)
	if _, e := rand.Read(randomness); e != nil {
		panic(e)
	}
	index := [32]byte{}
	copy(index[:], randomness[:32])
	bs = bs.Copy()
	if e := xorKeyBytes(bs, randomness[32:]); e != nil {
		panic(e)
	}
	s.store.Store(index, bs)
	if ttl != 0 {
		time.AfterFunc(ttl, func() {
			s.store.Delete(index)
		})
	}
	return index, randomness[32:]
}

// Read reads the encrypted key at index and decrypts it using mask.
// It returns a non-nil error if the index wasn't found or mask is not of the correct length.
func (s KeyStore) Read(index [32]byte, mask []byte) (KeyBytes, error) {
	loaded, ok := s.store.Load(index)
	if !ok {
		return nil, fmt.Errorf(`index not found`)
	}
	bs := loaded.(KeyBytes).Copy()
	if e := xorKeyBytes(bs, mask); e != nil {
		return nil, e
	}
	return bs, nil
}

// Delete removes an index from the store immediately.
func (s KeyStore) Delete(index [32]byte) {
	s.store.Delete(index)
}

func xorKeyBytes(bs KeyBytes, mask []byte) error {
	if len(bs) != len(mask) {
		return fmt.Errorf(`key / mask length mismatch`)
	}
	i, l := 0, len(bs)
	for ; i < l-(l%8); i += 8 {
		bs[i+0] ^= mask[i+0]
		bs[i+1] ^= mask[i+1]
		bs[i+2] ^= mask[i+2]
		bs[i+3] ^= mask[i+3]
		bs[i+4] ^= mask[i+4]
		bs[i+5] ^= mask[i+5]
		bs[i+6] ^= mask[i+6]
		bs[i+7] ^= mask[i+7]
	}
	for ; i < l; i++ {
		bs[i] ^= mask[i]
	}
	return nil
}
