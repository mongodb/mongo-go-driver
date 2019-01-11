package topologyx

import (
	"testing"

	"go.mongodb.org/mongo-driver/x/network/address"
	"go.mongodb.org/mongo-driver/x/network/command"
)

func TestProcessErr(t *testing.T) {
	s, err := NewServer(address.Address("localhost:27017"))
	if err != nil {
		t.Fatal(err)
	}
	s.connectionstate = connected
	sub, err := s.Subscribe()
	if err != nil {
		t.Fatal(err)
	}
	<-sub.C
	c := &Connection{s: s}
	err = command.Error{Code: 11000, Message: "E11000 duplicate key error collection: test.foo index: _id_ dup key: { : 12345.0 }"}
	c.processErr(err)
	select {
	case <-sub.C:
		t.Error("Received updated topology description but shouldn't have.")
	default:
	}
}
