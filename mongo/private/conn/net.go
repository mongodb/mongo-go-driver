// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package conn

import (
	"context"
	"net"
)

// Dialer dials a server according to the network and address.
type Dialer func(ctx context.Context, dialer *net.Dialer, network, address string) (net.Conn, error)

func dialWithoutTLS(ctx context.Context, dialer *net.Dialer, network, address string) (net.Conn, error) {
	conn, err := dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}

	return conn, nil
}
