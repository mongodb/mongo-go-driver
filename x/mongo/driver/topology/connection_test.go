// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package topology

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/x/network/address"
	"go.mongodb.org/mongo-driver/x/network/connection"
	"go.mongodb.org/mongo-driver/x/network/description"
	"go.mongodb.org/mongo-driver/x/network/wiremessage"
)

type netErr struct {
}

func (n netErr) Error() string {
	return "error"
}

func (n netErr) Timeout() bool {
	return false
}

func (n netErr) Temporary() bool {
	return false
}

type connect struct {
	err *connection.Error
}

func (c connect) WriteWireMessage(ctx context.Context, wm wiremessage.WireMessage) error {
	return *c.err
}
func (c connect) ReadWireMessage(ctx context.Context) (wiremessage.WireMessage, error) {
	return nil, *c.err
}
func (c connect) Close() error {
	return nil
}
func (c connect) Alive() bool {
	return true
}
func (c connect) Expired() bool {
	return false
}
func (c connect) ID() string {
	return ""
}

// Test case for sconn processErr
func TestConnectionProcessErrSpec(t *testing.T) {
	ctx := context.Background()
	s, err := NewServer(address.Address("localhost"), nil)
	require.NoError(t, err)

	desc := s.Description()
	require.Nil(t, desc.LastError)

	s.connectionstate = connected

	innerErr := netErr{}
	connectErr := connection.Error{ConnectionID: "blah", Wrapped: innerErr}
	c := connect{&connectErr}
	sc := sconn{c, s, 1}
	err = sc.WriteWireMessage(ctx, nil)
	require.NotNil(t, err)
	desc = s.Description()
	require.NotNil(t, desc.LastError)
	require.Equal(t, desc.Kind, (description.ServerKind)(description.Unknown))
}
