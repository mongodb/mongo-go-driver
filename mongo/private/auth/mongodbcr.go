// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package auth

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"

	"io"

	"github.com/10gen/mongo-go-driver/mongo/model"
	"github.com/10gen/mongo-go-driver/mongo/private/conn"
	"github.com/10gen/mongo-go-driver/mongo/private/msg"
	"github.com/skriptble/wilson/bson"
)

// MONGODBCR is the mechanism name for MONGODB-CR.
const MONGODBCR = "MONGODB-CR"

func newMongoDBCRAuthenticator(cred *Cred) (Authenticator, error) {
	return &MongoDBCRAuthenticator{
		DB:       cred.Source,
		Username: cred.Username,
		Password: cred.Password,
	}, nil
}

// MongoDBCRAuthenticator uses the MONGODB-CR algorithm to authenticate a connection.
type MongoDBCRAuthenticator struct {
	DB       string
	Username string
	Password string
}

// Auth authenticates the connection.
func (a *MongoDBCRAuthenticator) Auth(ctx context.Context, c conn.Connection) error {

	// Arbiters cannot be authenticated
	if c.Model().Kind == model.RSArbiter {
		return nil
	}

	db := a.DB
	if db == "" {
		db = defaultAuthDB
	}

	getNonceRequest := msg.NewCommand(
		msg.NextRequestID(),
		db,
		true,
		bson.NewDocument(bson.C.Int32("getnonce", 1)),
	)
	var getNonceResult struct {
		Nonce string `bson:"nonce"`
	}

	rdr, err := conn.ExecuteCommand(ctx, c, getNonceRequest)
	if err != nil {
		return newError(err, MONGODBCR)
	}

	err = bson.NewDecoder(bytes.NewReader(rdr)).Decode(&getNonceResult)
	if err != nil {
		return err
	}

	authRequest := msg.NewCommand(
		msg.NextRequestID(),
		db,
		true,
		bson.NewDocument(
			bson.C.Int32("authenticate", 1),
			bson.C.String("user", a.Username),
			bson.C.String("nonce", getNonceResult.Nonce),
			bson.C.String("key", a.createKey(getNonceResult.Nonce))),
	)
	_, err = conn.ExecuteCommand(ctx, c, authRequest)
	if err != nil {
		return newError(err, MONGODBCR)
	}

	return nil
}

func (a *MongoDBCRAuthenticator) createKey(nonce string) string {
	h := md5.New()

	_, _ = io.WriteString(h, nonce)
	_, _ = io.WriteString(h, a.Username)
	_, _ = io.WriteString(h, mongoPasswordDigest(a.Username, a.Password))
	return fmt.Sprintf("%x", h.Sum(nil))
}
