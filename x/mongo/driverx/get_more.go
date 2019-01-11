package driverx

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver/session"
	"go.mongodb.org/mongo-driver/x/network/description"
)

type GetMoreResult struct {
	ID    int64             // cursor ID
	Batch bsoncore.Document // firstBatch or nextBatch
}

type GetMoreOperation struct {
	id        int64     `drivergen:"ID"`
	ns        Namespace `drivergen:"Namespace"`
	maxTimeMS *int64
	batchSize *int64

	// TODO(GODRIVER-617): Not sure how we should handle setting this or if it even matters. We
	// don't use this for getMore commands.
	tkind  description.TopologyKind `drivergen:"TopologyKind"`
	clock  *session.ClusterClock    `drivergen:"ClusterClock"`
	client *session.Client          `drivergen:"Session"`
	server Server

	result GetMoreResult `drivergen:"-"`
}

func (gmo *GetMoreOperation) Result() GetMoreResult { return gmo.result }

func (gmo *GetMoreOperation) processResponse(response bsoncore.Document, _ Server) error { return nil }

func (gmo *GetMoreOperation) command(dst []byte, _ description.SelectedServer) ([]byte, error) {
	dst = bsoncore.AppendInt64Element(dst, "getMore", gmo.id)
	dst = bsoncore.AppendStringElement(dst, "collection", gmo.ns.Collection)

	if gmo.maxTimeMS != nil {
		dst = bsoncore.AppendInt64Element(dst, "maxTimeMS", *gmo.maxTimeMS)
	}
	if gmo.batchSize != nil {
		dst = bsoncore.AppendInt64Element(dst, "batchSize", *gmo.batchSize)
	}
	return dst, nil
}

func (gmo *GetMoreOperation) Execute(ctx context.Context) error {
	if gmo.server == nil {
		return errors.New("GetMoreOperation must have a Server set before Execute can be called.")
	}

	if gmo.ns.Collection == "" || gmo.ns.DB == "" {
		return errors.New("Collection and DB must be of non-zero length")
	}
	return OperationContext{
		CommandFn: gmo.command,
		Database:  gmo.ns.DB,

		Server:       gmo.server,
		TopologyKind: gmo.tkind,

		ProcessResponseFn: gmo.processResponse,

		Client: gmo.client,
		Clock:  gmo.clock,
	}.Execute(ctx)
}
