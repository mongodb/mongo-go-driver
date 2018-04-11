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
	"github.com/mongodb/mongo-go-driver/core/addr"
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
	Client *bson.Document

	ismstr  result.IsMaster
	bldinfo result.BuildInfo
	err     error
}

// Encode will encode the handshake commands into two wire messages. The wire
// messages are ordered with the isMaster command first and the buildInfo command
// second.
func (h *Handshake) Encode() ([2]wiremessage.WireMessage, error) {
	var wms [2]wiremessage.WireMessage
	ismstr, err := (&IsMaster{Client: h.Client}).Encode()
	if err != nil {
		return wms, err
	}
	buildinfo, err := (&BuildInfo{}).Encode()
	if err != nil {
		return wms, err
	}
	wms[0], wms[1] = ismstr, buildinfo
	return wms, nil
}

// Decode will decode the wire messages. The order of the wire messages are
// expected to be an isMaster response first and a buildInfo response second.
// Errors during decoding are deferred until either the Result or Err methods
// are called.
func (h *Handshake) Decode(wms [2]wiremessage.WireMessage) *Handshake {
	h.ismstr, h.err = (&IsMaster{}).Decode(wms[0]).Result()
	if h.err != nil {
		return h
	}
	h.bldinfo, h.err = (&BuildInfo{}).Decode(wms[1]).Result()
	if h.err != nil {
		return h
	}
	return h
}

// Result returns the result of decoded wire messages.
func (h *Handshake) Result(address addr.Addr) (description.Server, error) {
	if h.err != nil {
		return description.Server{}, h.err
	}
	return description.NewServer(address, h.ismstr, h.bldinfo), nil
}

// Err returns the error set on this Handshake.
func (h *Handshake) Err() error { return h.err }

// Handshake implements the connection.Handshaker interface. It is identical
// to the RoundTrip methods on other types in this package. It will execute
// the isMaster and buildInfo commands, using pipelining to enable a single
// roundtrip.
func (h *Handshake) Handshake(ctx context.Context, address addr.Addr, rw wiremessage.ReadWriter) (description.Server, error) {
	wms, err := h.Encode()
	if err != nil {
		return description.Server{}, err
	}

	err = rw.WriteWireMessage(ctx, wms[0])
	if err != nil {
		return description.Server{}, err
	}
	err = rw.WriteWireMessage(ctx, wms[1])
	if err != nil {
		return description.Server{}, err
	}

	wms[0], err = rw.ReadWireMessage(ctx)
	if err != nil {
		return description.Server{}, err
	}
	wms[1], err = rw.ReadWireMessage(ctx)
	if err != nil {
		return description.Server{}, err
	}
	return h.Decode(wms).Result(address)
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
