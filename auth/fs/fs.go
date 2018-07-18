// Copyright 2018 karma.run AG. All rights reserved.

package fs // import "github.com/karmarun/karma.link/auth/fs"

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/karmarun/karma.link/auth"
	"github.com/karmarun/karma.link/config"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var logger = log.New(config.LogWriter, `auth/fs`, config.LogFlags)

const (
	tokenExpiration = (15 * time.Minute)
)

// Folder is an implementation of auth.Authenticator that serves private keys from a folder path.
type Folder string

var (
	_ auth.Authenticator = Folder("")
)

// Credentials is the authentication JSON structure used in Folder.Authenticate
type Credentials struct {
	FilePath   []string `json:"filepath"`
	Passphrase string   `json:"passphrase"`
}

// Token represents the carrier token structure returned by Folder.Authenticate
type Token struct {
	Address string `json:"address"`
	Secret  string `json:"secret"`
	Expires string `json:"expires"`
}

const maxKeyFileSize = 1024 * 1024 // 1MB

var authenticated = &sync.Map{}

// Authenticate parses credentials as Credentials and attempts to authenticate them.
// It follows the rules specified in auth.Authenticator.
func (f Folder) Authenticate(credentials json.RawMessage) (json.RawMessage, error) {
	creds := Credentials{}
	if e := json.Unmarshal(credentials, &creds); e != nil {
		return nil, fmt.Errorf(`invalid credential structure`)
	}
	path := filepath.Join(append([]string{string(f)}, creds.FilePath...)...)
	file, e := os.Open(path)
	if e != nil {
		return nil, fmt.Errorf(`error opening key file`) // intentionally vague
	}
	defer file.Close()
	stat, e := file.Stat()
	if e != nil {
		logger.Println("error stat-ing file", path, e)
		return nil, fmt.Errorf(`internal error stat-ing key file (has been logged)`)
	}
	if stat.Size() > maxKeyFileSize {
		return nil, fmt.Errorf(`error opening key file`) // intentionally vague
	}
	bs, e := ioutil.ReadAll(file)
	if e != nil {
		logger.Println("error reading key file", path, e)
		return nil, fmt.Errorf(`error opening key file`) // intentionally vague
	}
	key, e := keystore.DecryptKey(bs, creds.Passphrase)
	if e != nil {
		return nil, fmt.Errorf(`invalid credentials`)
	}
	randomness := make([]byte, 128, 128)
	if _, e := rand.Read(randomness); e != nil {
		logger.Println("rand.Read returned error", e)
		return nil, fmt.Errorf(`internal error`)
	}
	secret := base64.StdEncoding.EncodeToString(randomness)
	authenticated.Store(secret, key)
	time.AfterFunc(tokenExpiration, func() {
		authenticated.Delete(secret)
	})
	bs, e = json.Marshal(Token{
		Address: key.Address.Hex(),
		Secret:  secret,
		Expires: time.Now().Add(tokenExpiration).Format(time.RFC3339),
	})
	if e != nil {
		authenticated.Delete(secret)
		logger.Println("failed marshalling token", e)
		return nil, fmt.Errorf(`internal error`)
	}
	return bs, nil
}

func (f Folder) RenewToken(oldToken json.RawMessage) (json.RawMessage, error) {
	tok, e := parseToken(oldToken)
	if e != nil {
		return nil, e
	}
	loaded, ok := authenticated.Load(tok.Secret)
	if !ok {
		return nil, fmt.Errorf(`invalid token`)
	}
	randomness := make([]byte, 128, 128)
	if _, e := rand.Read(randomness); e != nil {
		logger.Println("rand.Read returned error", e)
		return nil, fmt.Errorf(`internal error`)
	}
	secret := base64.StdEncoding.EncodeToString(randomness)
	key := loaded.(*auth.Key)
	authenticated.Store(secret, key)
	time.AfterFunc(tokenExpiration, func() {
		authenticated.Delete(secret)
	})
	bs, e := json.Marshal(Token{
		Address: key.Address.Hex(),
		Secret:  secret,
		Expires: time.Now().Add(tokenExpiration).Format(time.RFC3339),
	})
	if e != nil {
		authenticated.Delete(secret)
		logger.Println("failed marshalling token", e)
		return nil, fmt.Errorf(`internal error`)
	}
	return bs, nil
}

// ExchangeToken exchanges a previously issued Token for an auth.Key.
// It follows the rules specified in auth.Authenticator.
func (f Folder) ExchangeToken(token json.RawMessage) (*auth.Key, error) {
	tok, e := parseToken(token)
	if e != nil {
		return nil, e
	}
	loaded, ok := authenticated.Load(tok.Secret)
	if !ok {
		return nil, fmt.Errorf(`invalid token`)
	}
	key := loaded.(*keystore.Key)
	return &auth.Key{
		Address:    key.Address,
		PrivateKey: key.PrivateKey,
	}, nil
}

func parseToken(token json.RawMessage) (*Token, error) {
	tok := Token{}
	if e := json.Unmarshal(token, &tok); e != nil {
		return nil, fmt.Errorf(`invalid token`)
	}
	expiry, e := time.Parse(time.RFC3339, tok.Expires)
	if e != nil {
		return nil, fmt.Errorf(`invalid token expiration`)
	}
	if time.Now().After(expiry) {
		return nil, fmt.Errorf(`token expired`)
	}
	return &tok, nil
}
