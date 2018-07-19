package connection

import (
	"context"
	"io"

	"github.com/mongodb/mongo-go-driver/core/wiremessage"
)

// Hijacker is the interface implemented by types that want control a connection
// for a period of time.
type Hijacker interface {
	HijackConnection(wiremessage.WireMessage, Connection) error
}

// HijackerFunc is an adapter to enable generic functions to become a Hijacker.
type HijackerFunc func(wiremessage.WireMessage, Connection) error

// HijackConnection implements Hijacker.
func (hf HijackerFunc) HijackConnection(wm wiremessage.WireMessage, c Connection) error {
	return hf(wm, c)
}

// TransformerRetriever is the interface implemented by types that can return a
// transformer for a given wiremessage.WireMessage.
type TransformerRetriever interface {
	RetrieveTransformer(wiremessage.WireMessage) (interface{}, error)
}

// TransformerRetrieverFunc is an adapter to enable generic functions to become
// a TransformerRetriever.
type TransformerRetrieverFunc func(wiremessage.WireMessage) (interface{}, error)

// RetrieveTransformer implements TransformerRetriever.
func (trf TransformerRetrieverFunc) RetrieveTransformer(wm wiremessage.WireMessage) (interface{}, error) {
	return trf(wm)
}

// ProxyServer is a combination of a Proxy and a Server. It can be used to build
// a proxy that can handle specific commands on its own.
type ProxyServer struct {
	Retriever TransformerRetriever
	Pool      Pool
}

// HandleConnection implements the Handler interface.
func (ps *ProxyServer) HandleConnection(c Connection) {
	var serverConn Connection
	for {
		wm, err := c.ReadWireMessage(context.TODO())
		if err != nil {
			//  ¯\_(ツ)_/¯
			if cerr, ok := err.(Error); ok {
				if cerr.Wrapped == io.EOF {
					return
				}
			}
			panic(err)
		}

		var respTrnsfmr wiremessage.ResponseTransformer
		transformer, err := ps.Retriever.RetrieveTransformer(wm)
		if err != nil {
			panic(err)
		}
		switch t := transformer.(type) {
		case wiremessage.RequestTransformer:
			wm, respTrnsfmr, err = t.TransformRequestWireMessage(wm)
			if err != nil {
				if wm == nil {
					// close the connection
					return
				}
				err = c.WriteWireMessage(context.TODO(), wm) // send wm back to the client
				if err != nil {
					//  ¯\_(ツ)_/¯
					panic(err)
				}
			}
			if wm == nil {
				// Do not send to the server
				continue
			}
		case Hijacker:
			err = t.HijackConnection(wm, c)
			if err != nil {
				panic(err)
			}
			continue // restart at the read stage.
		default:
			//  ¯\_(ツ)_/¯
		}

		if serverConn == nil {
			serverConn, _, err = ps.Pool.Get(context.TODO())
			if err != nil {
				// Can't proxy
				return
			}
		}

		// TODO: Check to see if this OP_MSG with moreToCome set.
		err = serverConn.WriteWireMessage(context.TODO(), wm)
		if err != nil {
			// connection to server is broken, we could retry, but for now just
			// shut everything down.
			return
		}

		// TODO: We'll need to deal with exhaust mode eventually, but for now we
		// should assume that there is at most 1 response.
		wm, err = serverConn.ReadWireMessage(context.TODO())
		if err != nil {
			// Can't read response, shut it all down.
			return
		}

		if respTrnsfmr != nil {
			wm, err = respTrnsfmr.TransformResponseWireMessage(wm)
			if err != nil {
				// Not sure what to do here yet....
				return
			}
		}

		err = c.WriteWireMessage(context.TODO(), wm)
		if err != nil {
			// Can't write to the client/driver, shut it down.
			return
		}
	}
}
