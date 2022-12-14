package utils

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"os"
	"path"
	"sync"

	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/auth"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

const TOKEN_FILE = "token.json"

type MCChain struct {
	key       *ecdsa.PrivateKey
	chainData string
}

var (
	tokens     = map[string]oauth2.TokenSource{}
	token_lock = &sync.Mutex{}

	chains     = map[string]*MCChain{}
	chain_lock = &sync.Mutex{}
)

// GetTokenSource returns the token source for this username
func GetTokenSource(name string) oauth2.TokenSource {
	token_lock.Lock()
	defer token_lock.Unlock()

	if token, ok := tokens[name]; ok {
		return token
	}

	token, err := read_token(name)
	if err != nil {
		logrus.Errorf("failed to read token for %s", name)
	}

	tokens[name] = auth.RefreshTokenSource(token)
	new_token, err := tokens[name].Token()
	if err != nil {
		panic(err)
	}
	if !token.Valid() {
		logrus.Infof("Refreshed token for %s", name)
		write_token(name, new_token)
	}

	return tokens[name]
}

// GetChain gets a chain for this user
func GetChain(name string) (key *ecdsa.PrivateKey, chainData string, err error) {
	chain_lock.Lock()
	defer chain_lock.Unlock()
	if chain, ok := chains[name]; ok {
		return chain.key, chain.chainData, nil
	}
	key, chainData, err = minecraft.CreateChain(context.Background(), GetTokenSource(name))
	if err != nil {
		return nil, "", err
	}
	chains[name] = &MCChain{
		key:       key,
		chainData: chainData,
	}
	return key, chainData, nil
}

// write_token writes the token for this user to a json file
func write_token(name string, token *oauth2.Token) {
	os.Mkdir("tokens", 0o775)
	fname := path.Join("tokens", name+".json")
	buf, err := json.Marshal(token)
	if err != nil {
		panic(err)
	}
	os.WriteFile(fname, buf, 0o755)
}

// read_token reads the token of this user from a json file
// asks for interactive login if not found
func read_token(name string) (*oauth2.Token, error) {
	fname := path.Join("tokens", name+".json")
	if _, err := os.Stat(fname); err == nil {
		f, err := os.Open(fname)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		var token oauth2.Token
		if err := json.NewDecoder(f).Decode(&token); err != nil {
			return nil, err
		}
		return &token, nil
	} else {
		logrus.Infof("Login for %s", name)
		_token, err := auth.RequestLiveToken()
		if err != nil {
			return nil, err
		}
		write_token(name, _token)
		return _token, nil
	}
}
