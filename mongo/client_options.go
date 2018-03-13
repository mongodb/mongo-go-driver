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

// ClientOpt is a convenience variable provided for access to ClientOptions methods.
var ClientOpt ClientOptions

// ClientOptions is used as a namespace for mongo.Client option constructors.
type ClientOptions struct{}

// AppName specifies the client application name. This value is used by MongoDB when it logs
// connection information and profile information, such as slow queries.
func (ClientOptions) AppName(s string) options.AppName {
	return options.AppName(s)
}

// AuthMechanism indicates the mechanism to use for authentication.
//
// Supported values include "SCRAM-SHA-1", "MONGODB-CR", "PLAIN", "GSSAPI", and "MONGODB-X509".
func (ClientOptions) AuthMechanism(s string) options.AuthMechanism {
	return options.AuthMechanism(s)
}

// AuthMechanismProperties specifies additional configuration options which may be used by certain
// authentication mechanisms.
func (ClientOptions) AuthMechanismProperties(m map[string]string) options.AuthMechanismProperties {
	return options.AuthMechanismProperties(m)
}

// AuthSource specifies the database to authenticate against.
func (ClientOptions) AuthSource(s string) options.AuthSource {
	return options.AuthSource(s)
}

// ConnectTimeout specifies the timeout for an initial connection to a server.
func (ClientOptions) ConnectTimeout(d time.Duration) options.ConnectTimeout {
	return options.ConnectTimeout(d)
}

// HeartbeatInterval specifies the interval to wait between server monitoring checks.
func (ClientOptions) HeartbeatInterval(d time.Duration) options.HeartbeatInterval {
	return options.HeartbeatInterval(d)
}

// Hosts specifies the initial list of addresses from which to discover the rest of the cluster.
func (ClientOptions) Hosts(s ...string) options.Hosts {
	return options.Hosts(s)
}

// J specifies the "j" field of the default write concern to set on the Client
func (ClientOptions) J(b bool) options.J {
	return options.J(b)
}

// LocalThreshold specifies how far to distribute queries, beyond the server with the fastest
// round-trip time. If a server's roundtrip time is more than LocalThreshold slower than the
// the fastest, the driver will not send queries to that server.
func (ClientOptions) LocalThreshold(d time.Duration) options.LocalThreshold {
	return options.LocalThreshold(d)
}

// MaxConnIdleTime specifies the maximum number of milliseconds that a connection can remain idle
// in a connection pool before being removed and closed.
func (ClientOptions) MaxConnIdleTime(d time.Duration) options.MaxConnIdleTime {
	return options.MaxConnIdleTime(d)
}

// MaxConnsPerHost specifies the max size of a server's connection pool.
func (ClientOptions) MaxConnsPerHost(u uint16) options.MaxConnsPerHost {
	return options.MaxConnsPerHost(u)
}

// MaxIdleConnsPerHost specifies the number of connections in a server's connection pool that can
// be idle at any given time.
func (ClientOptions) MaxIdleConnsPerHost(u uint16) options.MaxIdleConnsPerHost {
	return options.MaxIdleConnsPerHost(u)
}

// Password specifies the password used for authentication.
func (ClientOptions) Password(s string) options.Password {
	return options.Password(s)
}

// ReadConcernLevel specifies the read concern level of the default read concern to set on the
// client.
func (ClientOptions) ReadConcernLevel(s string) options.ReadConcernLevel {
	return options.ReadConcernLevel(s)
}

// ReadPreference specifies the read preference mode of the default read preference to set on the
// client.
func (ClientOptions) ReadPreference(s string) options.ReadPreference {
	return options.ReadPreference(s)
}

// ReadPreferenceTagSets specifies the read preference tagsets of the default read preference to
// set on the client.
func (ClientOptions) ReadPreferenceTagSets(m ...map[string]string) options.ReadPreferenceTagSets {
	return options.ReadPreferenceTagSets(m)
}

// ReplicaSet specifies the name of the replica set of the cluster.
func (ClientOptions) ReplicaSet(s string) options.ReplicaSet {
	return options.ReplicaSet(s)
}

// ServerSelectionTimeout specifies a timeout in milliseconds to block for server selection.
func (ClientOptions) ServerSelectionTimeout(d time.Duration) options.ServerSelectionTimeout {
	return options.ServerSelectionTimeout(d)
}

// Single specifies whether the driver should connect directly to the server instead of
// auto-discovering other servers in the cluster
func (ClientOptions) Single(b bool) options.Single {
	return options.Single(b)
}

// SocketTimeout specifies the time in milliseconds to attempt to send or receive on a socket
// before the attempt times out.
func (ClientOptions) SocketTimeout(d time.Duration) options.SocketTimeout {
	return options.SocketTimeout(d)
}

// SSL indicates whether SSL should be enabled.
func (ClientOptions) SSL(b bool) options.SSL {
	return options.SSL(b)
}

// SSLClientCertificateKeyFile specifies the file containing the client certificate and private key
// used for authentication.
func (ClientOptions) SSLClientCertificateKeyFile(s string) options.SSLClientCertificateKeyFile {
	return options.SSLClientCertificateKeyFile(s)
}

// SSLInsecure indicates whether to skip the verification of the server certificate and hostname.
func (ClientOptions) SSLInsecure(b bool) options.SSLInsecure {
	return options.SSLInsecure(b)
}

// SSLCaFile specifies the file containing the certificate authority used for SSL connections.
func (ClientOptions) SSLCaFile(s string) options.SSLCaFile {
	return options.SSLCaFile(s)
}

// WString sets the "w" field of the default write concern to set on the client.
func (ClientOptions) WString(s string) options.WString {
	return options.WString(s)
}

// WNumber sets the "w" field of the default write concern to set on the client.
func (ClientOptions) WNumber(i int) options.WNumber {
	return options.WNumber(i)
}

// Username specifies the username that will be authenticated.
func (ClientOptions) Username(s string) options.Username {
	return options.Username(s)
}

// WTimeout sets the "wtimeout" field of the default write concern to set on the client.
func (ClientOptions) WTimeout(d time.Duration) options.WTimeout {
	return options.WTimeout(d)
}
