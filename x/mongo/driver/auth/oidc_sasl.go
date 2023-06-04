// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package auth

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo/address"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
	"go.mongodb.org/mongo-driver/x/mongo/driver/operation"
)

type oidcSaslConversation struct {
	SpeculativeConversation
	client oidcSaslClient
	source string
}

type oidcSaslClient interface {
	SaslClientCloser
	allowAddr(address.Address) bool
	auth(context.Context, *Config) error
}

func newOidcSaslConversation(client oidcSaslClient, source string, speculative bool) *oidcSaslConversation {
	return &oidcSaslConversation{
		SpeculativeConversation: newSaslConversation(client, source, speculative),
		client:                  client,
		source:                  source,
	}
}

func (osc *oidcSaslConversation) GetHandshakeInformation(ctx context.Context, hello *operation.Hello, addr address.Address, c driver.Connection) (driver.HandshakeInformation, error) {
	if !osc.client.allowAddr(addr) {
		return driver.HandshakeInformation{}, fmt.Errorf("OIDC host is not allowed: %s", addr.String())
	}

	return osc.SpeculativeConversation.GetHandshakeInformation(ctx, hello, addr, c)
}

func (osc *oidcSaslConversation) Finish(ctx context.Context, cfg *Config, firstResponse bsoncore.Document) error {
	if firstResponse == nil {
		osc.client.Close(cfg.Description.Addr)
		return nil
	}
	return osc.SpeculativeConversation.Finish(ctx, cfg, firstResponse)
}
