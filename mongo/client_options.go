// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"time"

	"github.com/mongodb/mongo-go-driver/core/connection"
	"github.com/mongodb/mongo-go-driver/core/connstring"
	"github.com/mongodb/mongo-go-driver/core/topology"
)

type option func(*Client) error

// ClientOptions is used as a namespace for mongo.Client option constructors.
type ClientOptions struct {
	next *ClientOptions
	opt  option
}

// ClientOpt is a convenience variable provided for access to ClientOptions methods.
var ClientOpt = &ClientOptions{}

// AppName specifies the client application name. This value is used by MongoDB when it logs
// connection information and profile information, such as slow queries.
func (co *ClientOptions) AppName(s string) *ClientOptions {
	var fn option = func(c *Client) error {
		if c.connString.AppName == "" {
			c.connString.AppName = s
		}
		return nil
	}
	return &ClientOptions{next: co, opt: fn}
}

// AuthMechanism indicates the mechanism to use for authentication.
//
// Supported values include "SCRAM-SHA-1", "MONGODB-CR", "PLAIN", "GSSAPI", and "MONGODB-X509".
func (co *ClientOptions) AuthMechanism(s string) *ClientOptions {
	var fn option = func(c *Client) error {
		if len(c.connString.AuthMechanism) == 0 {
			c.connString.AuthMechanism = s
		}
		return nil
	}
	return &ClientOptions{next: co, opt: fn}
}

// AuthMechanismProperties specifies additional configuration options which may be used by certain
// authentication mechanisms.
func (co *ClientOptions) AuthMechanismProperties(m map[string]string) *ClientOptions {
	var fn option = func(c *Client) error {
		if c.connString.AuthMechanismProperties == nil {
			c.connString.AuthMechanismProperties = m
		}
		return nil
	}
	return &ClientOptions{next: co, opt: fn}
}

// AuthSource specifies the database to authenticate against.
func (co *ClientOptions) AuthSource(s string) *ClientOptions {
	var fn option = func(c *Client) error {
		if len(c.connString.AuthSource) == 0 {
			c.connString.AuthSource = s
		}
		return nil
	}
	return &ClientOptions{next: co, opt: fn}
}

// ConnectTimeout specifies the timeout for an initial connection to a server.
// If a custom Dialer is used, this method won't be set and the user is
// responsible for setting the ConnectTimeout for connections on the dialer
// themselves.
func (co *ClientOptions) ConnectTimeout(d time.Duration) *ClientOptions {
	var fn option = func(c *Client) error {
		if !c.connString.ConnectTimeoutSet {
			c.connString.ConnectTimeout = d
			c.connString.ConnectTimeoutSet = true
		}
		return nil
	}
	return &ClientOptions{next: co, opt: fn}
}

// Dialer specifies a custom dialer used to dial new connections to a server.
func (co *ClientOptions) Dialer(d Dialer) *ClientOptions {
	var fn option = func(c *Client) error {
		c.topologyOptions = append(
			c.topologyOptions,
			topology.WithServerOptions(func(opts ...topology.ServerOption) []topology.ServerOption {
				return append(
					opts,
					topology.WithConnectionOptions(func(opts ...connection.Option) []connection.Option {
						return append(
							opts,
							connection.WithDialer(func(connection.Dialer) connection.Dialer {
								return d
							}),
						)
					}),
				)
			}),
		)
		return nil
	}
	return &ClientOptions{next: co, opt: fn}
}

// HeartbeatInterval specifies the interval to wait between server monitoring checks.
func (co *ClientOptions) HeartbeatInterval(d time.Duration) *ClientOptions {
	var fn option = func(c *Client) error {
		if !c.connString.HeartbeatIntervalSet {
			c.connString.HeartbeatInterval = d
			c.connString.HeartbeatIntervalSet = true
		}
		return nil
	}
	return &ClientOptions{next: co, opt: fn}
}

// Hosts specifies the initial list of addresses from which to discover the rest of the cluster.
func (co *ClientOptions) Hosts(s []string) *ClientOptions {
	var fn option = func(c *Client) error {
		if c.connString.Hosts == nil {
			c.connString.Hosts = s
		}
		return nil
	}
	return &ClientOptions{next: co, opt: fn}
}

// Journal specifies the "j" field of the default write concern to set on the Client.
func (co *ClientOptions) Journal(b bool) *ClientOptions {
	var fn option = func(c *Client) error {
		if !c.connString.JSet {
			c.connString.J = b
			c.connString.JSet = true
		}
		return nil
	}
	return &ClientOptions{next: co, opt: fn}
}

// LocalThreshold specifies how far to distribute queries, beyond the server with the fastest
// round-trip time. If a server's roundtrip time is more than LocalThreshold slower than the
// the fastest, the driver will not send queries to that server.
func (co *ClientOptions) LocalThreshold(d time.Duration) *ClientOptions {
	var fn option = func(c *Client) error {
		if !c.connString.LocalThresholdSet {
			c.connString.LocalThreshold = d
			c.connString.LocalThresholdSet = true
		}
		return nil
	}
	return &ClientOptions{next: co, opt: fn}
}

// MaxConnIdleTime specifies the maximum number of milliseconds that a connection can remain idle
// in a connection pool before being removed and closed.
func (co *ClientOptions) MaxConnIdleTime(d time.Duration) *ClientOptions {
	var fn option = func(c *Client) error {
		if !c.connString.MaxConnIdleTimeSet {
			c.connString.MaxConnIdleTime = d
			c.connString.MaxConnIdleTimeSet = true
		}
		return nil
	}
	return &ClientOptions{next: co, opt: fn}
}

// MaxConnsPerHost specifies the max size of a server's connection pool.
func (co *ClientOptions) MaxConnsPerHost(u uint16) *ClientOptions {
	var fn option = func(c *Client) error {
		if !c.connString.MaxConnsPerHostSet {
			c.connString.MaxConnsPerHost = u
			c.connString.MaxConnsPerHostSet = true
		}
		return nil
	}
	return &ClientOptions{next: co, opt: fn}
}

// MaxIdleConnsPerHost specifies the number of connections in a server's connection pool that can
// be idle at any given time.
func (co *ClientOptions) MaxIdleConnsPerHost(u uint16) *ClientOptions {
	var fn option = func(c *Client) error {
		if !c.connString.MaxIdleConnsPerHostSet {
			c.connString.MaxIdleConnsPerHost = u
			c.connString.MaxIdleConnsPerHostSet = true
		}
		return nil
	}
	return &ClientOptions{next: co, opt: fn}
}

// Password specifies the password used for authentication.
func (co *ClientOptions) Password(s string) *ClientOptions {
	var fn option = func(c *Client) error {
		if !c.connString.PasswordSet {
			c.connString.Password = s
			c.connString.PasswordSet = true
		}
		return nil
	}
	return &ClientOptions{next: co, opt: fn}
}

// ReadConcernLevel specifies the read concern level of the default read concern to set on the
// client.
func (co *ClientOptions) ReadConcernLevel(s string) *ClientOptions {
	var fn option = func(c *Client) error {
		if c.connString.ReadConcernLevel == "" {
			c.connString.ReadConcernLevel = s
		}
		return nil
	}
	return &ClientOptions{next: co, opt: fn}
}

// ReadPreference specifies the read preference mode of the default read preference to set on the
// client.
func (co *ClientOptions) ReadPreference(s string) *ClientOptions {
	var fn option = func(c *Client) error {
		if c.connString.ReadPreference == "" {
			c.connString.ReadPreference = s
		}
		return nil
	}
	return &ClientOptions{next: co, opt: fn}
}

// ReadPreferenceTagSets specifies the read preference tagsets of the default read preference to
// set on the client.
func (co *ClientOptions) ReadPreferenceTagSets(m []map[string]string) *ClientOptions {
	var fn option = func(c *Client) error {
		if c.connString.ReadPreferenceTagSets == nil {
			c.connString.ReadPreferenceTagSets = m
		}
		return nil
	}
	return &ClientOptions{next: co, opt: fn}
}

//MaxStaleness sets the "maxStaleness" field of the read pref to set on the client.
func (co *ClientOptions) MaxStaleness(d time.Duration) *ClientOptions {
	var fn option = func(c *Client) error {
		if !c.connString.MaxStalenessSet {
			c.connString.MaxStaleness = d
			c.connString.MaxStalenessSet = true
		}
		return nil
	}
	return &ClientOptions{next: co, opt: fn}
}

// ReplicaSet specifies the name of the replica set of the cluster.
func (co *ClientOptions) ReplicaSet(s string) *ClientOptions {
	var fn option = func(c *Client) error {
		if c.connString.ReplicaSet == "" {
			c.connString.ReplicaSet = s
		}
		return nil
	}
	return &ClientOptions{next: co, opt: fn}
}

// ServerSelectionTimeout specifies a timeout in milliseconds to block for server selection.
func (co *ClientOptions) ServerSelectionTimeout(d time.Duration) *ClientOptions {
	var fn option = func(c *Client) error {
		if !c.connString.ServerSelectionTimeoutSet {
			c.connString.ServerSelectionTimeout = d
			c.connString.ServerSelectionTimeoutSet = true
		}
		return nil
	}
	return &ClientOptions{next: co, opt: fn}
}

// Single specifies whether the driver should connect directly to the server instead of
// auto-discovering other servers in the cluster.
func (co *ClientOptions) Single(b bool) *ClientOptions {
	var fn option = func(c *Client) error {
		if !c.connString.ConnectSet {
			if b {
				c.connString.Connect = connstring.SingleConnect
			} else {
				c.connString.Connect = connstring.AutoConnect
			}

			c.connString.ConnectSet = true
		}
		return nil
	}
	return &ClientOptions{next: co, opt: fn}
}

// SocketTimeout specifies the time in milliseconds to attempt to send or receive on a socket
// before the attempt times out.
func (co *ClientOptions) SocketTimeout(d time.Duration) *ClientOptions {
	var fn option = func(c *Client) error {
		if !c.connString.SocketTimeoutSet {
			c.connString.SocketTimeout = d
			c.connString.SocketTimeoutSet = true
		}
		return nil
	}
	return &ClientOptions{next: co, opt: fn}
}

// SSL indicates whether SSL should be enabled.
func (co *ClientOptions) SSL(b bool) *ClientOptions {
	var fn option = func(c *Client) error {
		if !c.connString.SSLSet {
			c.connString.SSL = b
			c.connString.SSLSet = true
		}
		return nil
	}
	return &ClientOptions{next: co, opt: fn}
}

// SSLClientCertificateKeyFile specifies the file containing the client certificate and private key
// used for authentication.
func (co *ClientOptions) SSLClientCertificateKeyFile(s string) *ClientOptions {
	var fn option = func(c *Client) error {
		if !c.connString.SSLClientCertificateKeyFileSet {
			c.connString.SSLClientCertificateKeyFile = s
			c.connString.SSLClientCertificateKeyFileSet = true
		}
		return nil
	}
	return &ClientOptions{next: co, opt: fn}
}

// SSLClientCertificateKeyPassword provides a callback that returns a password used for decrypting the
// private key of a PEM file (if one is provided).
func (co *ClientOptions) SSLClientCertificateKeyPassword(s func() string) *ClientOptions {
	var fn option = func(c *Client) error {
		if !c.connString.SSLClientCertificateKeyPasswordSet {
			c.connString.SSLClientCertificateKeyPassword = s
			c.connString.SSLClientCertificateKeyPasswordSet = true
		}
		return nil
	}
	return &ClientOptions{next: co, opt: fn}
}

// SSLInsecure indicates whether to skip the verification of the server certificate and hostname.
func (co *ClientOptions) SSLInsecure(b bool) *ClientOptions {
	var fn option = func(c *Client) error {
		if !c.connString.SSLInsecureSet {
			c.connString.SSLInsecure = b
			c.connString.SSLInsecureSet = true
		}
		return nil
	}
	return &ClientOptions{next: co, opt: fn}
}

// SSLCaFile specifies the file containing the certificate authority used for SSL connections.
func (co *ClientOptions) SSLCaFile(s string) *ClientOptions {
	var fn option = func(c *Client) error {
		if !c.connString.SSLCaFileSet {
			c.connString.SSLCaFile = s
			c.connString.SSLCaFileSet = true
		}
		return nil
	}
	return &ClientOptions{next: co, opt: fn}
}

// WString sets the "w" field of the default write concern to set on the client.
func (co *ClientOptions) WString(s string) *ClientOptions {
	var fn option = func(c *Client) error {
		if !c.connString.WNumberSet && c.connString.WString == "" {
			c.connString.WString = s
			c.connString.WNumberSet = true
		}
		return nil
	}
	return &ClientOptions{next: co, opt: fn}
}

// WNumber sets the "w" field of the default write concern to set on the client.
func (co *ClientOptions) WNumber(i int) *ClientOptions {
	var fn option = func(c *Client) error {
		if !c.connString.WNumberSet && c.connString.WString == "" {
			c.connString.WNumber = i
			c.connString.WNumberSet = true
		}
		return nil
	}
	return &ClientOptions{next: co, opt: fn}
}

// Username specifies the username that will be authenticated.
func (co *ClientOptions) Username(s string) *ClientOptions {
	var fn option = func(c *Client) error {
		if c.connString.Username == "" {
			c.connString.Username = s
		}
		return nil
	}
	return &ClientOptions{next: co, opt: fn}
}

// WTimeout sets the "wtimeout" field of the default write concern to set on the client.
func (co *ClientOptions) WTimeout(d time.Duration) *ClientOptions {
	var fn option = func(c *Client) error {
		if !c.connString.WTimeoutSet && !c.connString.WTimeoutSetFromOption {
			c.connString.WTimeout = d
			c.connString.WTimeoutSetFromOption = true
		}
		return nil
	}
	return &ClientOptions{next: co, opt: fn}
}
