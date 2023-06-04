// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/uuid"
	"go.mongodb.org/mongo-driver/mongo/address"
	"go.mongodb.org/mongo-driver/mongo/auth"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

const (
	// MongoDBOIDC is the mechanism name for OIDC.
	MongoDBOIDC = "MONGODB-OIDC"

	oidcSource = "$external"

	oidcCallbackTimeout = 5 * time.Minute
)

// OidcCache stores OIDC authentications with key of username@address
var OidcCache sync.Map // cache map of map[string]*CacheEntry

// Auth contains authentication information.
type Auth struct {
	serverInfo   *auth.IDPServerInfo
	accessToken  *string
	refreshToken *string
	expiry       time.Time
}

// CacheEntry is the structure for an entry in OidcCache.
type CacheEntry struct {
	Auth         *Auth
	generationID uuid.UUID
	cond         sync.Cond

	reauthMutex sync.Mutex
}

func newOidcAuthenticator(cred *Cred) (Authenticator, error) {
	if cred.Source != oidcSource {
		return nil, fmt.Errorf("source in OIDC authentication must be '%s'", oidcSource)
	}
	if cred.Password != "" {
		return nil, fmt.Errorf("password is not allowed in OIDC authentication")
	}
	oidcauth := &OidcAuthenticator{
		Source: oidcSource,
	}
	if provider := cred.Props["PROVIDER_NAME"]; provider == "" {
		if cred.OidcOnRequest == nil {
			return nil, fmt.Errorf("missing OIDC request callback")
		}
		var hosts []string
		if allowed := cred.Props["ALLOWED_HOSTS"]; allowed != "" {
			err := json.Unmarshal([]byte(cred.Props["ALLOWED_HOSTS"]), &hosts)
			if err != nil {
				return nil, err
			}
		}
		oidcauth.Client = &oidcClient{
			username:     cred.Username,
			onRequest:    cred.OidcOnRequest,
			onRefresh:    cred.OidcOnRefresh,
			allowedHosts: hosts,
			reauth:       make(map[address.Address]*reauth),
		}
	} else {
		if cred.Username != "" {
			return nil, fmt.Errorf("username is not allowed in OIDC automatic authentication")
		}
		if cred.OidcOnRequest != nil || cred.OidcOnRefresh != nil {
			return nil, fmt.Errorf("callback is not allowed in OIDC automatic authentication")
		}
		switch provider {
		case "aws":
			oidcauth.Client = &aswOidcClient{}
		default:
			return nil, fmt.Errorf("%s is not supported in OIDC automatic authentication", provider)
		}
	}
	return oidcauth, nil
}

// OidcAuthenticator uses the OIDC algorithm over SASL to authenticate a connection.
type OidcAuthenticator struct {
	Source string
	Client oidcSaslClient
}

// CreateSpeculativeConversation creates a speculative conversation for OIDC authentication.
func (a *OidcAuthenticator) CreateSpeculativeConversation() SpeculativeConversation {
	return newOidcSaslConversation(a.Client, a.Source, true)
}

// Auth authenticates the connection.
func (a *OidcAuthenticator) Auth(ctx context.Context, cfg *Config) error {
	return a.Client.auth(ctx, cfg)
}

type aswOidcClient struct{}

func (*aswOidcClient) GetMechanism() string {
	return MongoDBOIDC
}

func (*aswOidcClient) Start(address.Address) ([]byte, error) {
	const env = "AWS_WEB_IDENTITY_TOKEN_FILE"
	builder := bsoncore.NewDocumentBuilder()
	var err error
	if path, ok := os.LookupEnv(env); !ok {
		err = fmt.Errorf("%s is not set", env)
	} else if res, err := ioutil.ReadFile(path); err == nil {
		builder.AppendString("jwt", string(res))
	}
	return builder.Build(), err
}

func (*aswOidcClient) Next(address.Address, []byte) ([]byte, error) {
	return nil, fmt.Errorf("unexpected step in OIDC automatic authentication")
}

func (*aswOidcClient) Completed() bool {
	return true
}

func (*aswOidcClient) Close(address.Address) {}

func (*aswOidcClient) allowAddr(address.Address) bool {
	return true
}

func (oc *aswOidcClient) auth(ctx context.Context, cfg *Config) error {
	return ConductSaslConversation(ctx, cfg, oidcSource, oc)
}

type reauthStep int

const (
	jwtReauth reauthStep = iota + 1
	principalReauth
)

type reauth struct {
	generationID uuid.UUID
	reauthStep   reauthStep
	reauthLocked *sync.Mutex
}

type oidcClient struct {
	username     string
	onRequest    auth.OidcOnRequest
	onRefresh    auth.OidcOnRefresh
	allowedHosts []string

	reauth map[address.Address]*reauth
}

func (*oidcClient) GetMechanism() string {
	return MongoDBOIDC
}

func (oc *oidcClient) Start(addr address.Address) ([]byte, error) {
	var hasKey bool
	key := fmt.Sprintf("%s@%s", oc.username, addr)
	entry := func() *CacheEntry {
		var v interface{}
		v, hasKey = OidcCache.LoadOrStore(key, &CacheEntry{
			Auth: &Auth{},
			cond: sync.Cond{L: &sync.Mutex{}},
		})
		return v.(*CacheEntry)
	}()
	entry.cond.L.Lock()
	defer entry.cond.L.Unlock()
	if hasKey {
		timeout := false
		ctx, cancel := context.WithTimeout(context.Background(), oidcCallbackTimeout)
		go func(ctx context.Context) {
			if <-ctx.Done(); ctx.Err() == context.DeadlineExceeded {
				timeout = true
				entry.cond.Broadcast()
			}
		}(ctx)
		for entry.Auth.serverInfo == nil {
			entry.cond.Wait()
			if timeout {
				break
			}
		}
		cancel()
	}
	if _, ok := oc.reauth[addr]; !ok {
		oc.reauth[addr] = &reauth{}
	}
	builder := bsoncore.NewDocumentBuilder()
	if entry.Auth.accessToken == nil {
		entry.Auth.serverInfo = nil
		if oc.username != "" {
			builder.AppendString("n", oc.username)
		}
		return builder.Build(), nil
	} else if entry.Auth.expiry.Add(-5 * time.Minute).After(time.Now()) {
		oc.reauth[addr].generationID = entry.generationID
		builder.AppendString("jwt", *entry.Auth.accessToken)
		return builder.Build(), nil
	} else if oc.onRefresh == nil || entry.Auth.refreshToken == nil {
		entry.Auth.serverInfo = nil
		if oc.username != "" {
			builder.AppendString("n", oc.username)
		}
		return builder.Build(), nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), oidcCallbackTimeout)
	defer cancel()
	respCh := make(chan auth.IDPServerResp)
	errCh := make(chan error)
	go func() {
		resp, err := oc.onRefresh(ctx, oc.username, *entry.Auth.serverInfo, auth.IDPRefreshInfo{RefreshToken: entry.Auth.refreshToken})
		if err != nil {
			errCh <- err
		} else {
			respCh <- resp
		}
	}()
	var err error
	select {
	case <-ctx.Done():
		err = ctx.Err()
	case cbErr := <-errCh:
		err = cbErr
		entry.Auth.serverInfo = nil
		entry.Auth.accessToken = nil
		entry.Auth.refreshToken = nil
		entry.Auth.expiry = time.Time{}
	case resp := <-respCh:
		entry.generationID, _ = uuid.New()
		oc.reauth[addr].generationID = entry.generationID
		entry.Auth.accessToken = &resp.AccessToken
		entry.Auth.refreshToken = resp.RefreshToken
		if resp.ExpiresInSeconds != nil {
			entry.Auth.expiry = time.Now().Add(time.Duration(*resp.ExpiresInSeconds) * time.Second)
		} else {
			entry.Auth.expiry = time.Time{}
		}
		builder.AppendString("jwt", resp.AccessToken)
	}
	return builder.Build(), err
}

func (oc *oidcClient) Next(addr address.Address, challenge []byte) ([]byte, error) {
	var b struct {
		Issuer        string   `bson:"issuer"`
		ClientID      string   `bson:"clientId"`
		RequestScopes []string `bson:"requestScopes,omitempty"`
	}
	_ = bson.Unmarshal(bsoncore.Document(challenge), &b)
	if _, ok := oc.reauth[addr]; !ok {
		oc.reauth[addr] = &reauth{}
	}
	key := fmt.Sprintf("%s@%s", oc.username, addr)
	entry := func() *CacheEntry {
		v, _ := OidcCache.LoadOrStore(key, &CacheEntry{
			Auth: &Auth{},
			cond: sync.Cond{L: &sync.Mutex{}},
		})
		return v.(*CacheEntry)
	}()
	entry.cond.L.Lock()
	defer func() {
		entry.cond.L.Unlock()
		entry.cond.Broadcast()
	}()
	entry.Auth.serverInfo = &auth.IDPServerInfo{
		Issuer:        b.Issuer,
		ClientID:      b.ClientID,
		RequestScopes: b.RequestScopes,
	}
	ctx, cancel := context.WithTimeout(context.Background(), oidcCallbackTimeout)
	defer cancel()
	respCh := make(chan auth.IDPServerResp)
	errCh := make(chan error)
	go func() {
		resp, err := oc.onRequest(ctx, oc.username, *entry.Auth.serverInfo)
		if err != nil {
			errCh <- err
		} else {
			respCh <- resp
		}
	}()
	builder := bsoncore.NewDocumentBuilder()
	var err error
	select {
	case <-ctx.Done():
		err = ctx.Err()
	case cbErr := <-errCh:
		err = cbErr
	case resp := <-respCh:
		entry.generationID, _ = uuid.New()
		oc.reauth[addr].generationID = entry.generationID
		entry.Auth.accessToken = &resp.AccessToken
		entry.Auth.refreshToken = resp.RefreshToken
		if resp.ExpiresInSeconds != nil {
			entry.Auth.expiry = time.Now().Add(time.Duration(*resp.ExpiresInSeconds) * time.Second)
		}
		builder.AppendString("jwt", resp.AccessToken)
	}
	return builder.Build(), err
}

func (*oidcClient) Completed() bool {
	return true
}

func (oc *oidcClient) Close(addr address.Address) {
	key := fmt.Sprintf("%s@%s", oc.username, addr)
	v, ok := OidcCache.Load(key)
	if !ok {
		return
	}
	entry := v.(*CacheEntry)
	entry.cond.L.Lock()
	defer entry.cond.L.Unlock()
	// finish uncompleted conversation.
	if entry.Auth.serverInfo == nil {
		entry.Auth.serverInfo = &auth.IDPServerInfo{}
		entry.cond.Signal()
	}
}

func (oc *oidcClient) allowAddr(addr address.Address) bool {
	if addr.Network() == "unix" {
		return false
	}
	host, _, err := net.SplitHostPort(string(addr))
	if err != nil && !strings.Contains(err.Error(), "missing port in address") {
		return false
	}
	isAllowed := false
	for _, allowed := range oc.allowedHosts {
		if allowed == host || (strings.HasPrefix(allowed, ".") && strings.HasSuffix(host, allowed)) {
			isAllowed = true
			break
		}
	}
	return isAllowed
}

func (oc *oidcClient) auth(ctx context.Context, cfg *Config) error {
	var r *reauth
	if v, ok := oc.reauth[cfg.Description.Addr]; ok {
		r = v
	} else {
		r = &reauth{}
		oc.reauth[cfg.Description.Addr] = r
	}
	key := fmt.Sprintf("%s@%s", oc.username, cfg.Description.Addr)
	v, ok := OidcCache.Load(key)
	if !ok {
		return nil
	}
	entry := v.(*CacheEntry)

	// Do not lock in recursive calls.
	if r.reauthLocked == nil {
		r.reauthLocked = &entry.reauthMutex
		r.reauthLocked.Lock()
		r.reauthStep = 0
		defer func() {
			r.reauthStep = 0
			r.reauthLocked.Unlock()
			r.reauthLocked = nil
		}()
	}

	if r.reauthStep == jwtReauth {
		oc.Close(cfg.Description.Addr)
	}

	// Trigger the reauthentication procedure if the generation IDs are identical.
	// Otherwise, start with the current token.
	func() {
		entry.cond.L.Lock()
		defer entry.cond.L.Unlock()

		if r.generationID == entry.generationID {
			switch r.reauthStep {
			case principalReauth:
				entry.Auth.accessToken = nil
				fallthrough
			case jwtReauth:
				entry.Auth.refreshToken = nil
				fallthrough
			default:
				entry.Auth.expiry = time.Time{}
			}
		}
	}()
	r.reauthStep++
	if r.reauthStep > principalReauth {
		return newError(nil, MongoDBOIDC)
	}

	return ConductSaslConversation(ctx, cfg, oidcSource, oc)
}
