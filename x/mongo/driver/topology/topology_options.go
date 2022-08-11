// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package topology

import (
	"crypto/tls"
	"encoding/pem"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo/description"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
	"go.mongodb.org/mongo-driver/x/mongo/driver/auth"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
	"go.mongodb.org/mongo-driver/x/mongo/driver/ocsp"
	"go.mongodb.org/mongo-driver/x/mongo/driver/operation"
	"go.mongodb.org/mongo-driver/x/mongo/driver/session"
)

// Option is a configuration option for a topology.
type Option func(*config) error

type config struct {
	mode                   MonitorMode
	replicaSetName         string
	seedList               []string
	serverOpts             []ServerOption
	cs                     connstring.ConnString // This must not be used for any logic in topology.Topology.
	uri                    string
	serverSelectionTimeout time.Duration
	serverMonitor          *event.ServerMonitor
	srvMaxHosts            int
	srvServiceName         string
	loadBalanced           bool
}

func newConfig(opts ...Option) (*config, error) {
	cfg := &config{
		seedList:               []string{"localhost:27017"},
		serverSelectionTimeout: 30 * time.Second,
	}

	for _, opt := range opts {
		if opt == nil {
			continue
		}
		err := opt(cfg)
		if err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

// convertToDriverAPIOptions converts a options.ServerAPIOptions instance to a driver.ServerAPIOptions.
func convertToDriverAPIOptions(s *options.ServerAPIOptions) *driver.ServerAPIOptions {
	driverOpts := driver.NewServerAPIOptions(string(s.ServerAPIVersion))
	if s.Strict != nil {
		driverOpts.SetStrict(*s.Strict)
	}
	if s.DeprecationErrors != nil {
		driverOpts.SetDeprecationErrors(*s.DeprecationErrors)
	}
	return driverOpts
}

type client interface {
	GetClusterClock() *session.ClusterClock
	SetServerAPI(*driver.ServerAPIOptions)
}

func newConfig_(co *options.ClientOptions, client client) (*config, error) {
	var defaultMaxPoolSize uint64 = 100
	var serverAPI *driver.ServerAPIOptions = nil

	var clusterClock *session.ClusterClock = nil
	if client != nil {
		clusterClock = client.GetClusterClock()
	}

	var defaultOptions int
	// Set default options
	if co.MaxPoolSize == nil {
		defaultOptions++
		co.SetMaxPoolSize(defaultMaxPoolSize)
	}
	if err := co.Validate(); err != nil {
		return nil, err
	}

	var connOpts []ConnectionOption
	var serverOpts []ServerOption
	var topologyOpts []Option

	// TODO(GODRIVER-814): Add tests for topology, server, and connection related options.

	// ServerAPIOptions need to be handled early as other client and server options below reference
	// c.serverAPI and serverOpts.serverAPI.
	if co.ServerAPIOptions != nil {
		serverAPI := convertToDriverAPIOptions(co.ServerAPIOptions)
		client.SetServerAPI(serverAPI)
		serverOpts = append(serverOpts, WithServerAPI(func(*driver.ServerAPIOptions) *driver.ServerAPIOptions {
			return serverAPI
		}))
	}

	// Pass down URI, SRV service name, and SRV max hosts so topology can poll SRV records correctly.
	topologyOpts = append(topologyOpts,
		WithURI(func(uri string) string { return co.GetURI() }),
		WithSRVServiceName(func(srvName string) string {
			if co.SRVServiceName != nil {
				return *co.SRVServiceName
			}
			return ""
		}),
		WithSRVMaxHosts(func(srvMaxHosts int) int {
			if co.SRVMaxHosts != nil {
				return *co.SRVMaxHosts
			}
			return 0
		}),
	)

	// AppName
	var appName string
	if co.AppName != nil {
		appName = *co.AppName

		serverOpts = append(serverOpts, WithServerAppName(func(string) string {
			return appName
		}))
	}
	// Compressors & ZlibLevel
	var comps []string
	if len(co.Compressors) > 0 {
		comps = co.Compressors

		connOpts = append(connOpts, WithCompressors(
			func(compressors []string) []string {
				return append(compressors, comps...)
			},
		))

		for _, comp := range comps {
			switch comp {
			case "zlib":
				connOpts = append(connOpts, WithZlibLevel(func(level *int) *int {
					return co.ZlibLevel
				}))
			case "zstd":
				connOpts = append(connOpts, WithZstdLevel(func(level *int) *int {
					return co.ZstdLevel
				}))
			}
		}

		serverOpts = append(serverOpts, WithCompressionOptions(
			func(opts ...string) []string { return append(opts, comps...) },
		))
	}

	var loadBalanced bool
	if co.LoadBalanced != nil {
		loadBalanced = *co.LoadBalanced
	}

	// Handshaker
	var handshaker = func(driver.Handshaker) driver.Handshaker {
		return operation.NewHello().AppName(appName).Compressors(comps).ClusterClock(clusterClock).
			ServerAPI(serverAPI).LoadBalanced(loadBalanced)
	}
	// Auth & Database & Password & Username
	if co.Auth != nil {
		cred := &auth.Cred{
			Username:    co.Auth.Username,
			Password:    co.Auth.Password,
			PasswordSet: co.Auth.PasswordSet,
			Props:       co.Auth.AuthMechanismProperties,
			Source:      co.Auth.AuthSource,
		}
		mechanism := co.Auth.AuthMechanism

		if len(cred.Source) == 0 {
			switch strings.ToUpper(mechanism) {
			case auth.MongoDBX509, auth.GSSAPI, auth.PLAIN:
				cred.Source = "$external"
			default:
				cred.Source = "admin"
			}
		}

		authenticator, err := auth.CreateAuthenticator(mechanism, cred)
		if err != nil {
			return nil, err
		}

		handshakeOpts := &auth.HandshakeOptions{
			AppName:       appName,
			Authenticator: authenticator,
			Compressors:   comps,
			ServerAPI:     serverAPI,
			LoadBalanced:  loadBalanced,
			ClusterClock:  clusterClock,
		}

		if mechanism == "" {
			// Required for SASL mechanism negotiation during handshake
			handshakeOpts.DBUser = cred.Source + "." + cred.Username
		}
		if co.AuthenticateToAnything != nil && *co.AuthenticateToAnything {
			// Authenticate arbiters
			handshakeOpts.PerformAuthentication = func(serv description.Server) bool {
				return true
			}
		}

		handshaker = func(driver.Handshaker) driver.Handshaker {
			return auth.Handshaker(nil, handshakeOpts)
		}
	}
	connOpts = append(connOpts, WithHandshaker(handshaker))
	// ConnectTimeout
	if co.ConnectTimeout != nil {
		serverOpts = append(serverOpts, WithHeartbeatTimeout(
			func(time.Duration) time.Duration { return *co.ConnectTimeout },
		))
		connOpts = append(connOpts, WithConnectTimeout(
			func(time.Duration) time.Duration { return *co.ConnectTimeout },
		))
	}
	// Dialer
	if co.Dialer != nil {
		connOpts = append(connOpts, WithDialer(
			func(Dialer) Dialer { return co.Dialer },
		))
	}
	// Direct
	if co.Direct != nil && *co.Direct {
		topologyOpts = append(topologyOpts, WithMode(
			func(MonitorMode) MonitorMode { return SingleMode },
		))
	}
	// HeartbeatInterval
	if co.HeartbeatInterval != nil {
		serverOpts = append(serverOpts, WithHeartbeatInterval(
			func(time.Duration) time.Duration { return *co.HeartbeatInterval },
		))
	}
	// Hosts
	hosts := []string{"localhost:27017"} // default host
	if len(co.Hosts) > 0 {
		hosts = co.Hosts
	}
	topologyOpts = append(topologyOpts, WithSeedList(
		func(...string) []string { return hosts },
	))
	// MaxConIdleTime
	if co.MaxConnIdleTime != nil {
		connOpts = append(connOpts, WithIdleTimeout(
			func(time.Duration) time.Duration { return *co.MaxConnIdleTime },
		))
	}
	// MaxPoolSize
	if co.MaxPoolSize != nil {
		serverOpts = append(
			serverOpts,
			WithMaxConnections(func(uint64) uint64 { return *co.MaxPoolSize }),
		)
	}
	// MinPoolSize
	if co.MinPoolSize != nil {
		serverOpts = append(
			serverOpts,
			WithMinConnections(func(uint64) uint64 { return *co.MinPoolSize }),
		)
	}
	// MaxConnecting
	if co.MaxConnecting != nil {
		serverOpts = append(
			serverOpts,
			WithMaxConnecting(func(uint64) uint64 { return *co.MaxConnecting }),
		)
	}
	// PoolMonitor
	if co.PoolMonitor != nil {
		serverOpts = append(
			serverOpts,
			WithConnectionPoolMonitor(func(*event.PoolMonitor) *event.PoolMonitor { return co.PoolMonitor }),
		)
	}
	// Monitor
	if co.Monitor != nil {
		connOpts = append(connOpts, WithMonitor(
			func(*event.CommandMonitor) *event.CommandMonitor { return co.Monitor },
		))
	}
	// ServerMonitor
	if co.ServerMonitor != nil {
		serverOpts = append(
			serverOpts,
			WithServerMonitor(func(*event.ServerMonitor) *event.ServerMonitor { return co.ServerMonitor }),
		)

		topologyOpts = append(
			topologyOpts,
			WithTopologyServerMonitor(func(*event.ServerMonitor) *event.ServerMonitor { return co.ServerMonitor }),
		)
	}
	// ReplicaSet
	if co.ReplicaSet != nil {
		topologyOpts = append(topologyOpts, WithReplicaSetName(
			func(string) string { return *co.ReplicaSet },
		))
	}
	// ServerSelectionTimeout
	if co.ServerSelectionTimeout != nil {
		topologyOpts = append(topologyOpts, WithServerSelectionTimeout(
			func(time.Duration) time.Duration { return *co.ServerSelectionTimeout },
		))
	}
	// SocketTimeout
	if co.SocketTimeout != nil {
		connOpts = append(
			connOpts,
			WithReadTimeout(func(time.Duration) time.Duration { return *co.SocketTimeout }),
			WithWriteTimeout(func(time.Duration) time.Duration { return *co.SocketTimeout }),
		)
	}
	// TLSConfig
	if co.TLSConfig != nil {
		connOpts = append(connOpts, WithTLSConfig(
			func(*tls.Config) *tls.Config {
				return co.TLSConfig
			},
		))
	}

	// OCSP cache
	ocspCache := ocsp.NewCache()
	connOpts = append(
		connOpts,
		WithOCSPCache(func(ocsp.Cache) ocsp.Cache { return ocspCache }),
	)

	// Disable communication with external OCSP responders.
	if co.DisableOCSPEndpointCheck != nil {
		connOpts = append(
			connOpts,
			WithDisableOCSPEndpointCheck(func(bool) bool { return *co.DisableOCSPEndpointCheck }),
		)
	}

	// LoadBalanced
	if co.LoadBalanced != nil {
		topologyOpts = append(
			topologyOpts,
			WithLoadBalanced(func(bool) bool { return *co.LoadBalanced }),
		)
		serverOpts = append(
			serverOpts,
			WithServerLoadBalanced(func(bool) bool { return *co.LoadBalanced }),
		)
		connOpts = append(
			connOpts,
			WithConnectionLoadBalanced(func(bool) bool { return *co.LoadBalanced }),
		)
	}

	serverOpts = append(
		serverOpts,
		WithClock(func(*session.ClusterClock) *session.ClusterClock { return clusterClock }),
		WithConnectionOptions(func(...ConnectionOption) []ConnectionOption { return connOpts }))
	topologyOpts = append(topologyOpts, WithServerOptions(
		func(...ServerOption) []ServerOption { return serverOpts },
	))

	// Deployment
	if co.Deployment != nil {
		// topology options: WithSeedlist, WithURI, WithSRVServiceName, WithSRVMaxHosts, and WithServerOptions
		// server options: WithClock and WithConnectionOptions + default maxPoolSize
		if len(serverOpts) > 2+defaultOptions || len(topologyOpts) > 5 {
			return nil, errors.New("cannot specify topology or server options with a deployment")
		}
	}
	return newConfig(topologyOpts...)
}

func NewConfig(co *options.ClientOptions) (*config, error) {
	return newConfig_(co, nil)
}

func NewConfigWithClient(co *options.ClientOptions, client client) (*config, error) {
	return newConfig_(co, client)
}

// WithMode configures the topology's monitor mode.
func WithMode(fn func(MonitorMode) MonitorMode) Option {
	return func(cfg *config) error {
		cfg.mode = fn(cfg.mode)
		return nil
	}
}

// WithReplicaSetName configures the topology's default replica set name.
func WithReplicaSetName(fn func(string) string) Option {
	return func(cfg *config) error {
		cfg.replicaSetName = fn(cfg.replicaSetName)
		return nil
	}
}

// WithSeedList configures a topology's seed list.
func WithSeedList(fn func(...string) []string) Option {
	return func(cfg *config) error {
		cfg.seedList = fn(cfg.seedList...)
		return nil
	}
}

// WithServerOptions configures a topology's server options for when a new server
// needs to be created.
func WithServerOptions(fn func(...ServerOption) []ServerOption) Option {
	return func(cfg *config) error {
		cfg.serverOpts = fn(cfg.serverOpts...)
		return nil
	}
}

// WithServerSelectionTimeout configures a topology's server selection timeout.
// A server selection timeout of 0 means there is no timeout for server selection.
func WithServerSelectionTimeout(fn func(time.Duration) time.Duration) Option {
	return func(cfg *config) error {
		cfg.serverSelectionTimeout = fn(cfg.serverSelectionTimeout)
		return nil
	}
}

// WithTopologyServerMonitor configures the monitor for all SDAM events
func WithTopologyServerMonitor(fn func(*event.ServerMonitor) *event.ServerMonitor) Option {
	return func(cfg *config) error {
		cfg.serverMonitor = fn(cfg.serverMonitor)
		return nil
	}
}

// WithURI specifies the URI that was used to create the topology.
func WithURI(fn func(string) string) Option {
	return func(cfg *config) error {
		cfg.uri = fn(cfg.uri)
		return nil
	}
}

// WithLoadBalanced specifies whether or not the cluster is behind a load balancer.
func WithLoadBalanced(fn func(bool) bool) Option {
	return func(cfg *config) error {
		cfg.loadBalanced = fn(cfg.loadBalanced)
		return nil
	}
}

// WithSRVMaxHosts specifies the SRV host limit that was used to create the topology.
func WithSRVMaxHosts(fn func(int) int) Option {
	return func(cfg *config) error {
		cfg.srvMaxHosts = fn(cfg.srvMaxHosts)
		return nil
	}
}

// WithSRVServiceName specifies the SRV service name that was used to create the topology.
func WithSRVServiceName(fn func(string) string) Option {
	return func(cfg *config) error {
		cfg.srvServiceName = fn(cfg.srvServiceName)
		return nil
	}
}

func loadCert(data []byte) ([]byte, error) {
	var certBlock *pem.Block

	for certBlock == nil {
		if len(data) == 0 {
			return nil, errors.New(".pem file must have both a CERTIFICATE and an RSA PRIVATE KEY section")
		}

		block, rest := pem.Decode(data)
		if block == nil {
			return nil, errors.New("invalid .pem file")
		}

		switch block.Type {
		case "CERTIFICATE":
			if certBlock != nil {
				return nil, errors.New("multiple CERTIFICATE sections in .pem file")
			}

			certBlock = block
		}

		data = rest
	}

	return certBlock.Bytes, nil
}
