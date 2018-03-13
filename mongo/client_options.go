// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"time"

	"github.com/mongodb/mongo-go-driver/mongo/private/options"
)

// ClientOptions is used as a namespace for mongo.Client option constructors.
type ClientOptions struct {
	next *ClientOptions
	opt  options.ClientOptioner
	err  error
}

// ClientOpt is a convenience variable provided for access to ClientOptions methods.
var ClientOpt = &ClientOptions{}

// AppName specifies the client application name. This value is used by MongoDB when it logs
// connection information and profile information, such as slow queries.
func (co *ClientOptions) AppName(s string) *ClientOptions {
	return &ClientOptions{next: co, opt: options.AppName(s), err: nil}
}

// AuthMechanism indicates the mechanism to use for authentication.
//
// Supported values include "SCRAM-SHA-1", "MONGODB-CR", "PLAIN", "GSSAPI", and "MONGODB-X509".
func (co *ClientOptions) AuthMechanism(s string) *ClientOptions {
	return &ClientOptions{next: co, opt: options.AuthMechanism(s), err: nil}
}

// AuthMechanismProperties specifies additional configuration options which may be used by certain
// authentication mechanisms.
func (co *ClientOptions) AuthMechanismProperties(m map[string]string) *ClientOptions {
	return &ClientOptions{next: co, opt: options.AuthMechanismProperties(m), err: nil}
}

// AuthSource specifies the database to authenticate against.
func (co *ClientOptions) AuthSource(s string) *ClientOptions {
	return &ClientOptions{next: co, opt: options.AuthSource(s), err: nil}
}

// ConnectTimeout specifies the timeout for an initial connection to a server.
func (co *ClientOptions) ConnectTimeout(d time.Duration) *ClientOptions {
	return &ClientOptions{next: co, opt: options.ConnectTimeout(d), err: nil}
}

// HeartbeatInterval specifies the interval to wait between server monitoring checks.
func (co *ClientOptions) HeartbeatInterval(d time.Duration) *ClientOptions {
	return &ClientOptions{next: co, opt: options.HeartbeatInterval(d), err: nil}
}

// Hosts specifies the initial list of addresses from which to discover the rest of the cluster.
func (co *ClientOptions) Hosts(s []string) *ClientOptions {
	return &ClientOptions{next: co, opt: options.Hosts(s), err: nil}
}

// Journal specifies the "j" field of the default write concern to set on the Client.
func (co *ClientOptions) Journal(b bool) *ClientOptions {
	return &ClientOptions{next: co, opt: options.Journal(b), err: nil}
}

// LocalThreshold specifies how far to distribute queries, beyond the server with the fastest
// round-trip time. If a server's roundtrip time is more than LocalThreshold slower than the
// the fastest, the driver will not send queries to that server.
func (co *ClientOptions) LocalThreshold(d time.Duration) *ClientOptions {
	return &ClientOptions{next: co, opt: options.LocalThreshold(d), err: nil}
}

// MaxConnIdleTime specifies the maximum number of milliseconds that a connection can remain idle
// in a connection pool before being removed and closed.
func (co *ClientOptions) MaxConnIdleTime(d time.Duration) *ClientOptions {
	return &ClientOptions{next: co, opt: options.MaxConnIdleTime(d), err: nil}
}

// MaxConnsPerHost specifies the max size of a server's connection pool.
func (co *ClientOptions) MaxConnsPerHost(u uint16) *ClientOptions {
	return &ClientOptions{next: co, opt: options.MaxConnsPerHost(u), err: nil}
}

// MaxIdleConnsPerHost specifies the number of connections in a server's connection pool that can
// be idle at any given time.
func (co *ClientOptions) MaxIdleConnsPerHost(u uint16) *ClientOptions {
	return &ClientOptions{next: co, opt: options.MaxIdleConnsPerHost(u), err: nil}
}

// Password specifies the password used for authentication.
func (co *ClientOptions) Password(s string) *ClientOptions {
	return &ClientOptions{next: co, opt: options.Password(s), err: nil}
}

// ReadConcernLevel specifies the read concern level of the default read concern to set on the
// client.
func (co *ClientOptions) ReadConcernLevel(s string) *ClientOptions {
	return &ClientOptions{next: co, opt: options.ReadConcernLevel(s), err: nil}
}

// ReadPreference specifies the read preference mode of the default read preference to set on the
// client.
func (co *ClientOptions) ReadPreference(s string) *ClientOptions {
	return &ClientOptions{next: co, opt: options.ReadPreference(s), err: nil}
}

// ReadPreferenceTagSets specifies the read preference tagsets of the default read preference to
// set on the client.
func (co *ClientOptions) ReadPreferenceTagSets(m []map[string]string) *ClientOptions {
	return &ClientOptions{next: co, opt: options.ReadPreferenceTagSets(m), err: nil}
}

// ReplicaSet specifies the name of the replica set of the cluster.
func (co *ClientOptions) ReplicaSet(s string) *ClientOptions {
	return &ClientOptions{next: co, opt: options.ReplicaSet(s), err: nil}
}

// ServerSelectionTimeout specifies a timeout in milliseconds to block for server selection.
func (co *ClientOptions) ServerSelectionTimeout(d time.Duration) *ClientOptions {
	return &ClientOptions{next: co, opt: options.ServerSelectionTimeout(d), err: nil}
}

// Single specifies whether the driver should connect directly to the server instead of
// auto-discovering other servers in the cluster.
func (co *ClientOptions) Single(b bool) *ClientOptions {
	return &ClientOptions{next: co, opt: options.Single(b), err: nil}
}

// SocketTimeout specifies the time in milliseconds to attempt to send or receive on a socket
// before the attempt times out.
func (co *ClientOptions) SocketTimeout(d time.Duration) *ClientOptions {
	return &ClientOptions{next: co, opt: options.SocketTimeout(d), err: nil}
}

// SSL indicates whether SSL should be enabled.
func (co *ClientOptions) SSL(b bool) *ClientOptions {
	return &ClientOptions{next: co, opt: options.SSL(b), err: nil}
}

// SSLClientCertificateKeyFile specifies the file containing the client certificate and private key
// used for authentication.
func (co *ClientOptions) SSLClientCertificateKeyFile(s string) *ClientOptions {
	return &ClientOptions{next: co, opt: options.SSLClientCertificateKeyFile(s), err: nil}
}

// SSLInsecure indicates whether to skip the verification of the server certificate and hostname.
func (co *ClientOptions) SSLInsecure(b bool) *ClientOptions {
	return &ClientOptions{next: co, opt: options.SSLInsecure(b), err: nil}
}

// SSLCaFile specifies the file containing the certificate authority used for SSL connections.
func (co *ClientOptions) SSLCaFile(s string) *ClientOptions {
	return &ClientOptions{next: co, opt: options.SSLCaFile(s), err: nil}
}

// WString sets the "w" field of the default write concern to set on the client.
func (co *ClientOptions) WString(s string) *ClientOptions {
	return &ClientOptions{next: co, opt: options.WString(s), err: nil}
}

// WNumber sets the "w" field of the default write concern to set on the client.
func (co *ClientOptions) WNumber(i int) *ClientOptions {
	return &ClientOptions{next: co, opt: options.WNumber(i), err: nil}
}

// Username specifies the username that will be authenticated.
func (co *ClientOptions) Username(s string) *ClientOptions {
	return &ClientOptions{next: co, opt: options.Username(s), err: nil}
}

// WTimeout sets the "wtimeout" field of the default write concern to set on the client.
func (co *ClientOptions) WTimeout(d time.Duration) *ClientOptions {
	return &ClientOptions{next: co, opt: options.WTimeout(d), err: nil}
}
