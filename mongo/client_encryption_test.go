// Copyright (C) MongoDB, Inc. 2024-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

//go:build cse
// +build cse

package mongo

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/mongocrypt"
)

func TestClientEncryption_ErrClientDisconnected(t *testing.T) {
	t.Parallel()

	client, _ := Connect(options.Client().ApplyURI("mongodb://test"))
	crypt := driver.NewCrypt(&driver.CryptOptions{MongoCrypt: &mongocrypt.MongoCrypt{}})

	ce := &ClientEncryption{keyVaultClient: client, crypt: crypt}
	_ = ce.Close(context.Background())

	t.Run("CreateEncryptedCollection", func(t *testing.T) {
		t.Parallel()
		_, _, err := ce.CreateEncryptedCollection(context.Background(), nil, "", options.CreateCollection(), "", nil)
		assert.ErrorIs(t, err, ErrClientDisconnected)
	})
	t.Run("AddKeyAltName", func(t *testing.T) {
		t.Parallel()
		err := ce.AddKeyAltName(context.Background(), bson.Binary{}, "").err
		assert.ErrorIs(t, err, ErrClientDisconnected)
	})
	t.Run("CreateDataKey", func(t *testing.T) {
		t.Parallel()
		_, err := ce.CreateDataKey(context.Background(), "", options.DataKey())
		assert.ErrorIs(t, err, ErrClientDisconnected)
	})
	t.Run("Encrypt", func(t *testing.T) {
		t.Parallel()
		_, err := ce.Encrypt(context.Background(), bson.RawValue{}, options.Encrypt())
		assert.ErrorIs(t, err, ErrClientDisconnected)
	})
	t.Run("EncryptExpression", func(t *testing.T) {
		t.Parallel()
		err := ce.EncryptExpression(context.Background(), nil, nil, options.Encrypt())
		assert.ErrorIs(t, err, ErrClientDisconnected)
	})
	t.Run("Decrypt", func(t *testing.T) {
		t.Parallel()
		_, err := ce.Decrypt(context.Background(), bson.Binary{})
		assert.ErrorIs(t, err, ErrClientDisconnected)
	})
	t.Run("Close", func(t *testing.T) {
		t.Parallel()
		err := ce.Close(context.Background())
		assert.ErrorIs(t, err, ErrClientDisconnected)
	})
	t.Run("DeleteKey", func(t *testing.T) {
		t.Parallel()
		_, err := ce.DeleteKey(context.Background(), bson.Binary{})
		assert.ErrorIs(t, err, ErrClientDisconnected)
	})
	t.Run("GetKeyByAltName", func(t *testing.T) {
		t.Parallel()
		err := ce.GetKeyByAltName(context.Background(), "").err
		assert.ErrorIs(t, err, ErrClientDisconnected)
	})
	t.Run("GetKey", func(t *testing.T) {
		t.Parallel()
		err := ce.GetKey(context.Background(), bson.Binary{}).err
		assert.ErrorIs(t, err, ErrClientDisconnected)
	})
	t.Run("GetKeys", func(t *testing.T) {
		t.Parallel()
		_, err := ce.GetKeys(context.Background())
		assert.ErrorIs(t, err, ErrClientDisconnected)
	})
	t.Run("RemoveKeyAltName", func(t *testing.T) {
		t.Parallel()
		err := ce.RemoveKeyAltName(context.Background(), bson.Binary{}, "").err
		assert.ErrorIs(t, err, ErrClientDisconnected)
	})
	t.Run("RewrapManyDataKey", func(t *testing.T) {
		t.Parallel()
		_, err := ce.RewrapManyDataKey(context.Background(), nil, options.RewrapManyDataKey())
		assert.ErrorIs(t, err, ErrClientDisconnected)
	})
}
