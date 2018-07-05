package fs

import (
	"auth"
	"config"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

var logger = log.New(config.LogWriter, `auth/fs`, config.LogFlags)

const tokenExpiration = (15 * time.Minute)

var randomness = &atomic.Value{}

func init() {
	rotateEntropy()
	go func() {
		for {
			time.Sleep(time.Minute)
			rotateEntropy()
		}
	}()
}

func rotateEntropy() {
	s := make([]byte, 1024, 1024)
	if _, e := rand.Read(s); e != nil {
		panic(e)
	}
	randomness.Store(s)
}

type Folder string

var (
	_ auth.Authenticator = Folder("")
)

type Credentials struct {
	FilePath   []string `json:"filepath"`
	Passphrase string   `json:"passphrase"`
}

type Token struct {
	Address string `json:"address"`
	Secret  string `json:"secret"`
	Expires string `json:"expires"`
}

const maxKeyFileSize = 1024 * 1024 // 1MB

var authenticated = &sync.Map{}

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
	timestamp, e := time.Now().MarshalBinary()
	if e != nil {
		logger.Println("error marshalling timestamp", e)
		return nil, fmt.Errorf(`internal error (has been logged)`)
	}
	hash := sha512.New()
	if _, e := hash.Write(timestamp); e != nil {
		panic(e) // should never happen
	}
	if _, e := hash.Write([]byte(path)); e != nil {
		panic(e) // should never happen
	}
	if _, e := hash.Write([]byte(creds.Passphrase)); e != nil {
		panic(e) // should never happen
	}
	if _, e := hash.Write(randomness.Load().([]byte)); e != nil {
		panic(e) // should never happen
	}
	secret := base64.StdEncoding.EncodeToString(hash.Sum(nil))
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
		return nil, fmt.Errorf(`internal error (has been logged)`)
	}
	return bs, nil
}

func (f Folder) ExchangeToken(token json.RawMessage) (*auth.Key, error) {
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
