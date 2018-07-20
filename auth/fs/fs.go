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
	Secret  string `json:"secret"`
	Expires string `json:"expires"`
}

const maxKeyFileSize = 1024 * 1024 // 1MB

var authenticated = &sync.Map{}

const secretLen = auth.KeyBytesLen + 32

// Authenticate parses credentials as Credentials and attempts to authenticate them.
// It follows the rules specified in auth.Authenticator.
func (f Folder) Authenticate(credentials json.RawMessage) (json.RawMessage, error) {
	creds := Credentials{}
	if e := json.Unmarshal(credentials, &creds); e != nil {
		return nil, fmt.Errorf(`invalid credentials`)
	}
	path := filepath.Join(append([]string{string(f)}, creds.FilePath...)...)
	file, e := os.Open(path)
	if e != nil {
		return nil, fmt.Errorf(`invalid credentials`) // intentionally vague
	}
	defer file.Close()
	stat, e := file.Stat()
	if e != nil {
		logger.Println("error stat-ing file", path, e)
		return nil, fmt.Errorf(`invalid credentials`) // intentionally vague
	}
	if stat.Size() > maxKeyFileSize {
		return nil, fmt.Errorf(`invalid credentials`) // intentionally vague
	}
	bs, e := ioutil.ReadAll(file)
	if e != nil {
		logger.Println("error reading key file", path, e)
		return nil, fmt.Errorf(`invalid credentials`) // intentionally vague
	}
	key := (*auth.Key)(nil)
	{
		decrypted, e := keystore.DecryptKey(bs, creds.Passphrase)
		if e != nil {
			return nil, fmt.Errorf(`invalid credentials`)
		}
		key = &auth.Key{Address: decrypted.Address, PrivateKey: decrypted.PrivateKey}
		decrypted = nil
	}
	keyBytes := auth.KeyToBytes(key)

	randomness := make([]byte, secretLen, secretLen)
	if _, e := rand.Read(randomness); e != nil {
		logger.Println("rand.Read returned error", e)
		return nil, fmt.Errorf(`internal error`)
	}

	index := [32]byte{}
	copy(index[:], randomness[:32])
	xorKeyBytes(keyBytes, randomness[32:])

	authenticated.Store(index, *keyBytes)
	keyBytes.Destroy()
	keyBytes = nil

	time.AfterFunc(tokenExpiration, func() {
		authenticated.Delete(index)
	})

	bs, e = json.Marshal(Token{
		Secret:  base64.StdEncoding.EncodeToString(randomness),
		Expires: time.Now().Add(tokenExpiration).Format(time.RFC3339),
	})
	if e != nil {
		authenticated.Delete(index)
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
	secret, e := base64.StdEncoding.DecodeString(tok.Secret)
	if e != nil {
		return nil, fmt.Errorf(`invalid token`)
	}
	if len(secret) != secretLen {
		return nil, fmt.Errorf(`invalid token`)
	}
	index := [32]byte{}
	copy(index[:], secret[:32])
	loaded, ok := authenticated.Load(index)
	if !ok {
		return nil, fmt.Errorf(`invalid token`)
	}
	keyByteArray := loaded.(auth.KeyBytes)
	keyBytes := &keyByteArray
	xorKeyBytes(keyBytes, secret[32:])

	randomness := make([]byte, secretLen, secretLen)
	if _, e := rand.Read(randomness); e != nil {
		logger.Println("rand.Read returned error", e)
		return nil, fmt.Errorf(`internal error`)
	}

	index = [32]byte{}
	copy(index[:], randomness[:32])
	xorKeyBytes(keyBytes, randomness[32:])

	authenticated.Store(index, *keyBytes)
	keyBytes.Destroy()
	keyBytes = nil

	time.AfterFunc(tokenExpiration, func() {
		authenticated.Delete(index)
	})

	bs, e := json.Marshal(Token{
		Secret:  base64.StdEncoding.EncodeToString(randomness),
		Expires: time.Now().Add(tokenExpiration).Format(time.RFC3339),
	})
	if e != nil {
		authenticated.Delete(index)
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
	secret, e := base64.StdEncoding.DecodeString(tok.Secret)
	if e != nil {
		return nil, fmt.Errorf(`invalid token`)
	}
	if len(secret) != secretLen {
		return nil, fmt.Errorf(`invalid token`)
	}
	index := [32]byte{}
	copy(index[:], secret[:32])
	loaded, ok := authenticated.Load(index)
	if !ok {
		return nil, fmt.Errorf(`invalid token`)
	}
	keyByteArray := loaded.(auth.KeyBytes)
	keyBytes := &keyByteArray
	xorKeyBytes(keyBytes, secret[32:])
	return auth.BytesToKey(keyBytes), nil
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

func xorKeyBytes(bs *auth.KeyBytes, mask []byte) {
	if len(mask) != auth.KeyBytesLen {
		panic("precondition violation: len(mask) != auth.KeyBytesLen")
	}
	i := 0
	for ; i < auth.KeyBytesLen-(auth.KeyBytesLen%8); i += 8 {
		bs[i+0] ^= mask[i+0]
		bs[i+1] ^= mask[i+1]
		bs[i+2] ^= mask[i+2]
		bs[i+3] ^= mask[i+3]
		bs[i+4] ^= mask[i+4]
		bs[i+5] ^= mask[i+5]
		bs[i+6] ^= mask[i+6]
		bs[i+7] ^= mask[i+7]
	}
	for ; i < auth.KeyBytesLen; i++ {
		bs[i] ^= mask[i]
	}
}
