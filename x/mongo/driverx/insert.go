package driverx

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver/session"
	"go.mongodb.org/mongo-driver/x/network/command"
	"go.mongodb.org/mongo-driver/x/network/commandx"
	"go.mongodb.org/mongo-driver/x/network/description"
	"go.mongodb.org/mongo-driver/x/network/result"
)

type InsertOperation struct {
	documents                []bsoncore.Document
	ordered                  *bool
	writeConcern             *writeconcern.WriteConcern
	bypassDocumentValidation *bool
	retry                    *bool

	serverSelector description.ServerSelector
	client         *session.Client
	clock          *session.ClusterClock
}

func Insert(documents ...bsoncore.Document) InsertOperation {
	return InsertOperation{documents: documents}
}

func (io InsertOperation) Documents(documents ...bsoncore.Document) InsertOperation {
	io.documents = documents
	return io
}

func (io InsertOperation) Ordered(ordered bool) InsertOperation {
	io.ordered = &ordered
	return io
}

func (io InsertOperation) WriteConcern(wc *writeconcern.WriteConcern) InsertOperation {
	io.writeConcern = wc
	return io
}

func (io InsertOperation) BypassDocumentValidation(b bool) InsertOperation {
	io.bypassDocumentValidation = &b
	return io
}

func (io InsertOperation) RetryWrite(retry bool) InsertOperation {
	io.retry = &retry
	return io
}

func (io InsertOperation) ServerSelector(ss description.ServerSelector) InsertOperation {
	io.serverSelector = ss
	return io
}

func (io InsertOperation) Session(client *session.Client) InsertOperation {
	io.client = client
	return io
}

func (io InsertOperation) ClusterClock(clock *session.ClusterClock) InsertOperation {
	io.clock = clock
	return io
}

func (io InsertOperation) Execute(ctx context.Context, ns Namespace, d Deployment) (result.Insert, error) {
	selector := io.serverSelector
	if selector == nil {
		selector = description.CompositeSelector([]description.ServerSelector{
			description.ReadPrefSelector(readpref.Primary()),
			description.LatencySelector(15 * time.Millisecond),
		})
	}

	srvr, err := d.SelectServer(ctx, selector)
	if err != nil {
		return result.Insert{}, err
	}

	deploymentKind := d.Description().Kind

	conn, err := srvr.Connection(ctx)
	if err != nil {
		return result.Insert{}, err
	}

	desc := conn.Description()

	if !retrySupported(d.Description(), desc, io.client, io.writeConcern) || (io.retry != nil && *io.retry == false) {
		defer conn.Close()
		if io.client != nil {
			io.client.RetryWrite = false // explicitly set to false to prevent encoding transaction number
		}
		return io.execute(ctx, deploymentKind, conn, nil)
	}

	// io.client must not be nil or retrySupported would have returned false
	io.client.RetryWrite = true
	io.client.IncrementTxnNumber()

	res, originalErr := io.execute(ctx, deploymentKind, conn, nil)

	// Retry if appropriate
	if cerr, ok := originalErr.(command.Error); ok && cerr.Retryable() ||
		res.WriteConcernError != nil && commandx.IsWriteConcernErrorRetryable(res.WriteConcernError) {
		conn.Close()
		srvr, err = d.SelectServer(ctx, selector)

		// Return original error if server selection fails.
		if err != nil {
			return res, originalErr
		}

		conn, err = srvr.Connection(ctx)
		// Return original error if connection retrieval fails or new server does not support retryable writes.
		if err != nil || conn == nil || !retrySupported(d.Description(), conn.Description(), io.client, io.writeConcern) {
			return res, originalErr
		}
		defer conn.Close()

		return io.execute(ctx, deploymentKind, conn, cerr)
	}
	conn.Close()

	return res, originalErr
}

func (io InsertOperation) execute(ctx context.Context, kind description.TopologyKind, conn Connection, oldErr error) (result.Insert, error) {
	return result.Insert{}, nil
}
