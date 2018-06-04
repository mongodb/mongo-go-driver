// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package command

import (
	"context"
	"runtime"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/address"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/result"
	"github.com/mongodb/mongo-go-driver/core/version"
	"github.com/mongodb/mongo-go-driver/core/wiremessage"
)

// Handshake represents a generic MongoDB Handshake. It calls isMaster and
// buildInfo.
//
// The isMaster and buildInfo commands are used to build a server description.
type Handshake struct {
	Client      *bson.Document
	Compressors []string

	ismstr result.IsMaster
	err    error
}

// Encode will encode the handshake commands into a wire message containing isMaster
func (h *Handshake) Encode() (wiremessage.WireMessage, error) {
	var wm wiremessage.WireMessage
	ismstr, err := (&IsMaster{Client: h.Client, Compressors: h.Compressors}).Encode()
	if err != nil {
		return wm, err
	}

	wm = ismstr
	return wm, nil
}

// Decode will decode the wire messages.
// Errors during decoding are deferred until either the Result or Err methods
// are called.
func (h *Handshake) Decode(wm wiremessage.WireMessage) *Handshake {
	h.ismstr, h.err = (&IsMaster{}).Decode(wm).Result()
	if h.err != nil {
		return h
	}
	return h
}

// Result returns the result of decoded wire messages.
func (h *Handshake) Result(addr address.Address) (description.Server, error) {
	if h.err != nil {
		return description.Server{}, h.err
	}
	return description.NewServer(addr, h.ismstr), nil
}

// Err returns the error set on this Handshake.
func (h *Handshake) Err() error { return h.err }

// Handshake implements the connection.Handshaker interface. It is identical
// to the RoundTrip methods on other types in this package. It will execute
// the isMaster command.
func (h *Handshake) Handshake(ctx context.Context, addr address.Address, rw wiremessage.ReadWriter) (description.Server, error) {
	wm, err := h.Encode()
	if err != nil {
		return description.Server{}, err
	}

	err = rw.WriteWireMessage(ctx, wm)
	if err != nil {
		return description.Server{}, err
	}

	wm, err = rw.ReadWireMessage(ctx)
	if err != nil {
		return description.Server{}, err
	}
	return h.Decode(wm).Result(addr)
}

// ClientDoc creates a client information document for use in an isMaster
// command.
func ClientDoc(app string) *bson.Document {
	doc := bson.NewDocument(
		bson.EC.SubDocumentFromElements(
			"driver",
			bson.EC.String("name", "mongo-go-driver"),
			bson.EC.String("version", version.Driver),
		),
		bson.EC.SubDocumentFromElements(
			"os",
			bson.EC.String("type", runtime.GOOS),
			bson.EC.String("architecture", runtime.GOARCH),
		),
		bson.EC.String("platform", runtime.Version()))

	if app != "" {
		doc.Append(bson.EC.SubDocumentFromElements(
			"application",
			bson.EC.String("name", app),
		))
	}

	return doc
}
