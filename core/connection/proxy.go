// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package connection

import (
	"context"
	"io"

	"github.com/mongodb/mongo-go-driver/core/wiremessage"
)

// Proxy implements a MongoDB proxy. It will use the given pool to connect to a
// MongoDB server and proxy the traffic between connections it is given and the
// server. It will pass each of the WireMessages it reads from the handled
// connection to a wiremessage.RequestTransformer. If an error is returned from
// the transformer and the wire message is nil, the connection will be closed.
// If the wire message is not nil, it will be used as an error and sent back to
// the client. If there is not an error the returned message will be passed onto
// the server.
//
// TODO(GODRIVER-268): Implement this.
type Proxy struct {
	Transformer wiremessage.RequestTransformer
	Pool        Pool
	// TODO: We should have some sort of logger here for error messages or
	// perhaps an error handler.
}

// HandleConnection implements the Handler interface.
func (p *Proxy) HandleConnection(c Connection) {
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
		// We can probably handle this case better. It seems like a proxy can
		// operate in a mode where it's both a proxy and a server, e.g. it can
		// sometimes pass the wiremessage back to be sent to the server,
		// sometimes modify the wiremessage and then return it to be sent to the
		// server, and sometimes just handle the wiremessage itself and return
		// its own response. The current structure here handles that, but it
		// isn't clear and the signaling is annoying. Using an error to indicate
		// that the Transformer actually wants to handle the request is
		// annoying.
		//
		// Instead of doing this we should add a new type, ProxyServer, that can
		// do the basic proxying that this type can do, but can also fully
		// handle a wiremessage. Since we've already read a wiremessage, the
		// handler would get both a wiremessage and a connection.
		if p.Transformer != nil {
			wm, respTrnsfmr, err = p.Transformer.TransformRequestWireMessage(wm)
			if err != nil {
				if wm == nil {
					// If we just got an error it makes sense to close the
					// connection.
					return
				}
				// Send the wm back to the client
				err = c.WriteWireMessage(context.TODO(), wm)
				if err != nil {
					//  ¯\_(ツ)_/¯
					// Should we close here? The connection might be broken now.
				}
			}
			if wm == nil {
				// Do not send to the server
				continue
			}
		}

		if serverConn == nil {
			serverConn, _, err = p.Pool.Get(context.TODO())
			if err != nil {
				// Can't proxy
				return
			}
		}

		// TODO: Check to see if this is OP_MSG with moreToCome set.
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
