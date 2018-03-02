// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package auth

import (
	"context"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/private/conn"
	"github.com/mongodb/mongo-go-driver/mongo/private/msg"
)

// MongoDBX509 is the mechanism name for MongoDBX509.
const MongoDBX509 = "MONGODB-X509"

func newMongoDBX509Authenticator(cred *Cred) (Authenticator, error) {
	return &MongoDBX509Authenticator{User: cred.Username}, nil
}

// MongoDBX509Authenticator uses X.509 certificates over TLS to authenticate a connection.
type MongoDBX509Authenticator struct {
	User string
}

// Auth implements the Authenticator interface.
func (a *MongoDBX509Authenticator) Auth(ctx context.Context, c conn.Connection) error {
	authRequestDoc := bson.NewDocument(
		bson.EC.Int32("authenticate", 1),
		bson.EC.String("mechanism", MongoDBX509),
	)

	if !c.Model().Version.AtLeast(3, 4) {
		authRequestDoc.Append(bson.EC.String("user", a.User))
	}

	authRequest := msg.NewCommand(
		msg.NextRequestID(),
		"$external",
		true,
		authRequestDoc,
	)

	_, err := conn.ExecuteCommand(ctx, c, authRequest)
	if err != nil {
		return err
	}

	return nil
}
