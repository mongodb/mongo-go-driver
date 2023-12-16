package mnet

import (
	"go.mongodb.org/mongo-driver/mongo/address"
	"go.mongodb.org/mongo-driver/mongo/description"
)

// Describer represents a Connection that can be described.
type Describer interface {
	Description() description.Server
	ID() string
	ServerConnectionID() *int64
	DriverConnectionID() int64
	Address() address.Address
	Stale() bool
}

type defaultDescriber struct{}

// TODO: Add Logic
func (*defaultDescriber) Description() description.Server { return description.Server{} }
func (*defaultDescriber) ID() string                      { return "" }
func (*defaultDescriber) ServerConnectionID() *int64      { return nil }
func (*defaultDescriber) DriverConnectionID() int64       { return 0 }
func (*defaultDescriber) Address() address.Address        { return "" }
func (*defaultDescriber) Stale() bool                     { return false }
