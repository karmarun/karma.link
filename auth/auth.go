// Copyright 2018 karma.run AG. All rights reserved.

package auth // import "github.com/karmarun/karma.link/auth"

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"sync"
)

// Key associates an ecdsa.PrivateKey with an Ethereum address.
type Key struct {
	Address    common.Address
	PrivateKey *ecdsa.PrivateKey
}

// An implementation of Authenticator must be able to parse credentials as JSON,
// validate them and return a non-nil error in case they are invalid.
// On success, an implementation of Authenticator must return some kind of carrier
// token that the client may use later in ExchangeToken.
type Authenticator interface {
	Authenticate(credentials json.RawMessage) (token json.RawMessage, e error)
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
