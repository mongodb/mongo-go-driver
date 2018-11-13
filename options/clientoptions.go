// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

import (
	"context"
	"net"
	"time"

	"github.com/mongodb/mongo-go-driver/bson/bsoncodec"
	"github.com/mongodb/mongo-go-driver/core/connection"
	"github.com/mongodb/mongo-go-driver/core/connstring"
	"github.com/mongodb/mongo-go-driver/core/event"
	"github.com/mongodb/mongo-go-driver/core/topology"
	"github.com/mongodb/mongo-go-driver/mongo/readconcern"
	"github.com/mongodb/mongo-go-driver/mongo/readpref"
	"github.com/mongodb/mongo-go-driver/mongo/writeconcern"
)

// ContextDialer makes new network connections
type ContextDialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// SSLOpt holds client SSL options.
//
// Enabled indicates whether SSL should be enabled.
//
// ClientCertificateKeyFile specifies the file containing the client certificate and private key
// used for authentication.
//
// ClientCertificateKeyPassword provides a callback that returns a password used for decrypting the
// private key of a PEM file (if one is provided).
//
// Insecure indicates whether to skip the verification of the server certificate and hostname.
//
// CaFile specifies the file containing the certificate authority used for SSL connections.
type SSLOpt struct {
	Enabled                      bool
	ClientCertificateKeyFile     string
	ClientCertificateKeyPassword func() string
	Insecure                     bool
	CaFile                       string
}

// Credential holds auth options.
//
// AuthMechanism indicates the mechanism to use for authentication.
// Supported values include "SCRAM-SHA-256", "SCRAM-SHA-1", "MONGODB-CR", "PLAIN", "GSSAPI", and "MONGODB-X509".
//
// AuthMechanismProperties specifies additional configuration options which may be used by certain
// authentication mechanisms.
//
// AuthSource specifies the database to authenticate against.
//
// Username specifies the username that will be authenticated.
//
// Password specifies the password used for authentication.
type Credential struct {
	AuthMechanism           string
	AuthMechanismProperties map[string]string
	AuthSource              string
	Username                string
	Password                string
}

// ClientOptions represents all possbile options to configure a client.
type ClientOptions struct {
	TopologyOptions []topology.Option
	ConnString      connstring.ConnString
	RetryWrites     *bool
	ReadPreference  *readpref.ReadPref
	ReadConcern     *readconcern.ReadConcern
	WriteConcern    *writeconcern.WriteConcern
	Registry        *bsoncodec.Registry
}

// Client creates a new ClientOptions instance.
func Client() *ClientOptions {
	return &ClientOptions{
		TopologyOptions: make([]topology.Option, 0),
	}
}

// SetAppName specifies the client application name. This value is used by MongoDB when it logs
// connection information and profile information, such as slow queries.
func (c *ClientOptions) SetAppName(s string) *ClientOptions {
	c.ConnString.AppName = s

	return c
}

// SetAuth sets the authentication options.
func (c *ClientOptions) SetAuth(auth Credential) *ClientOptions {
	c.ConnString.AuthMechanism = auth.AuthMechanism
	c.ConnString.AuthMechanismProperties = auth.AuthMechanismProperties
	c.ConnString.AuthSource = auth.AuthSource
	c.ConnString.Username = auth.Username
	c.ConnString.Password = auth.Password

	return c
}

// SetConnectTimeout specifies the timeout for an initial connection to a server.
// If a custom Dialer is used, this method won't be set and the user is
// responsible for setting the ConnectTimeout for connections on the dialer
// themselves.
func (c *ClientOptions) SetConnectTimeout(d time.Duration) *ClientOptions {
	c.ConnString.ConnectTimeout = d
	c.ConnString.ConnectTimeoutSet = true

	return c
}

// SetDialer specifies a custom dialer used to dial new connections to a server.
func (c *ClientOptions) SetDialer(d ContextDialer) *ClientOptions {
	c.TopologyOptions = append(
		c.TopologyOptions,
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

	return c
}

// SetMonitor specifies a command monitor used to see commands for a client.
func (c *ClientOptions) SetMonitor(m *event.CommandMonitor) *ClientOptions {
	c.TopologyOptions = append(
		c.TopologyOptions,
		topology.WithServerOptions(func(opts ...topology.ServerOption) []topology.ServerOption {
			return append(
				opts,
				topology.WithConnectionOptions(func(opts ...connection.Option) []connection.Option {
					return append(
						opts,
						connection.WithMonitor(func(*event.CommandMonitor) *event.CommandMonitor {
							return m
						}),
					)
				}),
			)
		}),
	)

	return c
}

// SetHeartbeatInterval specifies the interval to wait between server monitoring checks.
func (c *ClientOptions) SetHeartbeatInterval(d time.Duration) *ClientOptions {
	c.ConnString.HeartbeatInterval = d
	c.ConnString.HeartbeatIntervalSet = true

	return c
}

// SetHosts specifies the initial list of addresses from which to discover the rest of the cluster.
func (c *ClientOptions) SetHosts(s []string) *ClientOptions {
	c.ConnString.Hosts = s

	return c
}

// SetLocalThreshold specifies how far to distribute queries, beyond the server with the fastest
// round-trip time. If a server's roundtrip time is more than LocalThreshold slower than the
// the fastest, the driver will not send queries to that server.
func (c *ClientOptions) SetLocalThreshold(d time.Duration) *ClientOptions {
	c.ConnString.LocalThreshold = d
	c.ConnString.LocalThresholdSet = true

	return c
}

// SetMaxConnIdleTime specifies the maximum number of milliseconds that a connection can remain idle
// in a connection pool before being removed and closed.
func (c *ClientOptions) SetMaxConnIdleTime(d time.Duration) *ClientOptions {
	c.ConnString.MaxConnIdleTime = d
	c.ConnString.MaxConnIdleTimeSet = true

	return c
}

// SetMaxConnsPerHost specifies the max size of a server's connection pool.
func (c *ClientOptions) SetMaxConnsPerHost(u uint16) *ClientOptions {
	c.ConnString.MaxConnsPerHost = u
	c.ConnString.MaxConnsPerHostSet = true

	return c
}

// SetMaxIdleConnsPerHost specifies the number of connections in a server's connection pool that can
// be idle at any given time.
func (c *ClientOptions) SetMaxIdleConnsPerHost(u uint16) *ClientOptions {
	c.ConnString.MaxIdleConnsPerHost = u
	c.ConnString.MaxIdleConnsPerHostSet = true

	return c
}

// SetReadConcern specifies the read concern.
func (c *ClientOptions) SetReadConcern(rc *readconcern.ReadConcern) *ClientOptions {
	c.ReadConcern = rc

	return c
}

// SetReadPreference specifies the read preference.
func (c *ClientOptions) SetReadPreference(rp *readpref.ReadPref) *ClientOptions {
	c.ReadPreference = rp

	return c
}

// SetRegistry specifies the bsoncodec.Registry.
func (c *ClientOptions) SetRegistery(registry *bsoncodec.Registry) *ClientOptions {
	c.Registry = registry

	return c
}

// SetReplicaSet specifies the name of the replica set of the cluster.
func (c *ClientOptions) SetReplicaSet(s string) *ClientOptions {
	c.ConnString.ReplicaSet = s

	return c
}

// SetRetryWrites specifies whether the client has retryable writes enabled.
func (c *ClientOptions) SetRetryWrites(b bool) *ClientOptions {
	c.RetryWrites = &b

	return c
}

// SetServerSelectionTimeout specifies a timeout in milliseconds to block for server selection.
func (c *ClientOptions) SetServerSelectionTimeout(d time.Duration) *ClientOptions {
	c.ConnString.ServerSelectionTimeout = d
	c.ConnString.ServerSelectionTimeoutSet = true

	return c
}

// SetSingle specifies whether the driver should connect directly to the server instead of
// auto-discovering other servers in the cluster.
func (c *ClientOptions) SetSingle(b bool) *ClientOptions {
	if b {
		c.ConnString.Connect = connstring.SingleConnect
	} else {
		c.ConnString.Connect = connstring.AutoConnect
	}
	c.ConnString.ConnectSet = true

	return c
}

// SetSocketTimeout specifies the time in milliseconds to attempt to send or receive on a socket
// before the attempt times out.
func (c *ClientOptions) SetSocketTimeout(d time.Duration) *ClientOptions {
	c.ConnString.SocketTimeout = d
	c.ConnString.SocketTimeoutSet = true

	return c
}

// SetSSL sets SSL options.
func (c *ClientOptions) SetSSL(ssl *SSLOpt) *ClientOptions {
	c.ConnString.SSL = ssl.Enabled
	c.ConnString.SSLSet = true

	c.ConnString.SSLClientCertificateKeyFile = ssl.ClientCertificateKeyFile
	c.ConnString.SSLClientCertificateKeyFileSet = true

	c.ConnString.SSLClientCertificateKeyPassword = ssl.ClientCertificateKeyPassword
	c.ConnString.SSLClientCertificateKeyPasswordSet = true

	c.ConnString.SSLInsecure = ssl.Insecure
	c.ConnString.SSLInsecureSet = true

	c.ConnString.SSLCaFile = ssl.CaFile
	c.ConnString.SSLCaFileSet = true

	return c
}

// SetWriteConcern sets the write concern.
func (c *ClientOptions) SetWriteConcern(wc *writeconcern.WriteConcern) *ClientOptions {
	c.WriteConcern = wc

	return c
}

// MergeClientOptions combines the given connstring and *ClientOptions into a single *ClientOptions in a last one wins
// fashion. The given connstring will be used for the default options, which can be overwritten using the given
// *ClientOptions.
func MergeClientOptions(cs connstring.ConnString, opts ...*ClientOptions) *ClientOptions {
	c := Client()
	c.ConnString = cs

	for _, opt := range opts {
		if opt == nil {
			continue
		}
		c.TopologyOptions = append(c.TopologyOptions, opt.TopologyOptions...)

		if an := opt.ConnString.AppName; an != "" {
			c.ConnString.AppName = an
		}
		if am := opt.ConnString.AuthMechanism; len(am) != 0 {
			c.ConnString.AuthMechanism = am
		}
		if amp := opt.ConnString.AuthMechanismProperties; amp != nil {
			c.ConnString.AuthMechanismProperties = amp
		}
		if as := opt.ConnString.AuthSource; len(as) != 0 {
			c.ConnString.AuthSource = as
		}
		if u := opt.ConnString.Username; len(u) != 0 {
			c.ConnString.Username = u
		}
		if p := opt.ConnString.Password; len(p) != 0 {
			c.ConnString.Password = p
		}
		if opt.ConnString.ConnectTimeoutSet {
			c.ConnString.ConnectTimeoutSet = true
			c.ConnString.ConnectTimeout = opt.ConnString.ConnectTimeout
		}
		if opt.ConnString.HeartbeatIntervalSet {
			c.ConnString.HeartbeatIntervalSet = true
			c.ConnString.HeartbeatInterval = opt.ConnString.HeartbeatInterval
		}
		if h := opt.ConnString.Hosts; h != nil {
			c.ConnString.Hosts = h
		}
		if opt.ConnString.LocalThresholdSet {
			c.ConnString.LocalThresholdSet = true
			c.ConnString.LocalThreshold = opt.ConnString.LocalThreshold
		}
		if opt.ConnString.MaxConnIdleTimeSet {
			c.ConnString.MaxConnIdleTimeSet = true
			c.ConnString.MaxConnIdleTime = opt.ConnString.MaxConnIdleTime
		}
		if opt.ConnString.MaxConnsPerHostSet {
			c.ConnString.MaxConnsPerHostSet = true
			c.ConnString.MaxConnsPerHost = opt.ConnString.MaxConnsPerHost
		}
		if opt.ConnString.MaxIdleConnsPerHostSet {
			c.ConnString.MaxIdleConnsPerHostSet = true
			c.ConnString.MaxIdleConnsPerHost = opt.ConnString.MaxIdleConnsPerHost
		}
		if opt.ReadConcern != nil {
			c.ReadConcern = opt.ReadConcern
		}
		if opt.ReadPreference != nil {
			c.ReadPreference = opt.ReadPreference
		}
		if opt.Registry != nil {
			c.Registry = opt.Registry
		}
		if rs := opt.ConnString.ReplicaSet; rs != "" {
			c.ConnString.ReplicaSet = rs
		}
		if opt.RetryWrites != nil {
			c.RetryWrites = opt.RetryWrites
		}
		if opt.ConnString.ServerSelectionTimeoutSet {
			c.ConnString.ServerSelectionTimeoutSet = true
			c.ConnString.ServerSelectionTimeout = opt.ConnString.ServerSelectionTimeout
		}
		if opt.ConnString.ConnectSet {
			c.ConnString.ConnectSet = true
			c.ConnString.Connect = opt.ConnString.Connect
		}
		if opt.ConnString.SocketTimeoutSet {
			c.ConnString.SocketTimeoutSet = true
			c.ConnString.SocketTimeout = opt.ConnString.SocketTimeout
		}
		if opt.ConnString.SSLSet {
			c.ConnString.SSLSet = true
			c.ConnString.SSL = opt.ConnString.SSL
		}
		if opt.ConnString.SSLClientCertificateKeyFileSet {
			c.ConnString.SSLClientCertificateKeyFileSet = true
			c.ConnString.SSLClientCertificateKeyFile = opt.ConnString.SSLClientCertificateKeyFile
		}
		if opt.ConnString.SSLClientCertificateKeyPasswordSet {
			c.ConnString.SSLClientCertificateKeyPasswordSet = true
			c.ConnString.SSLClientCertificateKeyPassword = opt.ConnString.SSLClientCertificateKeyPassword
		}
		if opt.ConnString.SSLInsecureSet {
			c.ConnString.SSLInsecureSet = true
			c.ConnString.SSLInsecure = opt.ConnString.SSLInsecure
		}
		if opt.ConnString.SSLCaFileSet {
			c.ConnString.SSLCaFileSet = true
			c.ConnString.SSLCaFile = opt.ConnString.SSLCaFile
		}
		if opt.WriteConcern != nil {
			c.WriteConcern = opt.WriteConcern
		}
	}

	return c
}
