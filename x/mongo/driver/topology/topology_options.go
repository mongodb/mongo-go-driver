// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package topology

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/internal/logger"
	"go.mongodb.org/mongo-driver/internal/mongoutil"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
	"go.mongodb.org/mongo-driver/x/mongo/driver/auth"
	"go.mongodb.org/mongo-driver/x/mongo/driver/ocsp"
	"go.mongodb.org/mongo-driver/x/mongo/driver/operation"
	"go.mongodb.org/mongo-driver/x/mongo/driver/session"
)

const defaultServerSelectionTimeout = 30 * time.Second
const defaultConnectionTimeout = 30 * time.Second

// Config is used to construct a topology.
type Config struct {
	Mode                   MonitorMode
	ReplicaSetName         string
	SeedList               []string
	ServerOpts             []ServerOption
	URI                    string
	ConnectTimeout         time.Duration
	Timeout                *time.Duration
	ServerSelectionTimeout time.Duration
	ServerMonitor          *event.ServerMonitor
	SRVMaxHosts            int
	SRVServiceName         string
	LoadBalanced           bool
	logger                 *logger.Logger
}

// ConvertToDriverAPIOptions converts a options.ServerAPIOptions instance to a driver.ServerAPIOptions.
func ConvertToDriverAPIOptions(opts mongoutil.MongoOptions[options.ServerAPIArgs]) *driver.ServerAPIOptions {
	args, _ := mongoutil.NewArgsFromOptions[options.ServerAPIArgs](opts)

	driverOpts := driver.NewServerAPIOptions(string(args.ServerAPIVersion))
	if args.Strict != nil {
		driverOpts.SetStrict(*args.Strict)
	}
	if args.DeprecationErrors != nil {
		driverOpts.SetDeprecationErrors(*args.DeprecationErrors)
	}
	return driverOpts
}

func newLogger(opts mongoutil.MongoOptions[options.LoggerArgs]) (*logger.Logger, error) {
	if opts == nil {
		opts = options.Logger()
	}

	args, err := mongoutil.NewArgsFromOptions[options.LoggerArgs](opts)
	if err != nil {
		return nil, fmt.Errorf("failed to construct arguments from options: %w", err)
	}

	componentLevels := make(map[logger.Component]logger.Level)
	for component, level := range args.ComponentLevels {
		componentLevels[logger.Component(component)] = logger.Level(level)
	}

	log, err := logger.New(args.Sink, args.MaxDocumentLength, componentLevels)
	if err != nil {
		return nil, fmt.Errorf("error creating logger: %w", err)
	}

	return log, nil
}

// NewConfig behaves like NewConfigFromArgs by extracting arguments from the
// ClientOptions setters.
func NewConfig(opts *options.ClientOptions, clock *session.ClusterClock) (*Config, error) {
	args, err := mongoutil.NewArgsFromOptions[options.ClientArgs](opts)
	if err != nil {
		return nil, fmt.Errorf("failed to construct arguments from options: %w", err)
	}

	return NewConfigFromArgs(args, clock)
}

// NewConfigFromArgs will translate data from client arguments into a topology
// config for building non-default deployments. Server and topology options are
// not honored if a custom deployment is used.
func NewConfigFromArgs(args *options.ClientArgs, clock *session.ClusterClock) (*Config, error) {
	var serverAPI *driver.ServerAPIOptions

	clientOpts := options.ClientOptions{
		Opts: []func(*options.ClientArgs) error{
			func(ca *options.ClientArgs) error {
				*ca = *args

				return nil
			},
		},
	}

	if err := clientOpts.Validate(); err != nil {
		return nil, err
	}

	var connOpts []ConnectionOption
	var serverOpts []ServerOption

	cfgp := &Config{
		Timeout: args.Timeout,
	}

	// Set the default "ServerSelectionTimeout" to 30 seconds.
	cfgp.ServerSelectionTimeout = defaultServerSelectionTimeout

	// Set the default "ConnectionTimeout" to 30 seconds.
	cfgp.ConnectTimeout = defaultConnectionTimeout

	// Set the default "SeedList" to localhost.
	cfgp.SeedList = []string{"localhost:27017"}

	// TODO(GODRIVER-814): Add tests for topology, server, and connection related options.

	// ServerAPIOptions need to be handled early as other client and server options below reference
	// c.serverAPI and serverOpts.serverAPI.
	if args.ServerAPIOptions != nil {
		serverAPI = ConvertToDriverAPIOptions(args.ServerAPIOptions)
		serverOpts = append(serverOpts, WithServerAPI(func(*driver.ServerAPIOptions) *driver.ServerAPIOptions {
			return serverAPI
		}))
	}

	cfgp.URI = args.GetURI()

	if args.SRVServiceName != nil {
		cfgp.SRVServiceName = *args.SRVServiceName
	}

	if args.SRVMaxHosts != nil {
		cfgp.SRVMaxHosts = *args.SRVMaxHosts
	}

	// AppName
	var appName string
	if args.AppName != nil {
		appName = *args.AppName

		serverOpts = append(serverOpts, WithServerAppName(func(string) string {
			return appName
		}))
	}
	// Compressors & ZlibLevel
	var comps []string
	if len(args.Compressors) > 0 {
		comps = args.Compressors

		connOpts = append(connOpts, WithCompressors(
			func(compressors []string) []string {
				return append(compressors, comps...)
			},
		))

		for _, comp := range comps {
			switch comp {
			case "zlib":
				connOpts = append(connOpts, WithZlibLevel(func(level *int) *int {
					return args.ZlibLevel
				}))
			case "zstd":
				connOpts = append(connOpts, WithZstdLevel(func(level *int) *int {
					return args.ZstdLevel
				}))
			}
		}

		serverOpts = append(serverOpts, WithCompressionOptions(
			func(opts ...string) []string { return append(opts, comps...) },
		))
	}

	var loadBalanced bool
	if args.LoadBalanced != nil {
		loadBalanced = *args.LoadBalanced
	}

	// Handshaker
	var handshaker = func(driver.Handshaker) driver.Handshaker {
		return operation.NewHello().AppName(appName).Compressors(comps).ClusterClock(clock).
			ServerAPI(serverAPI).LoadBalanced(loadBalanced)
	}
	// Auth & Database & Password & Username
	if args.Auth != nil {
		cred := &auth.Cred{
			Username:    args.Auth.Username,
			Password:    args.Auth.Password,
			PasswordSet: args.Auth.PasswordSet,
			Props:       args.Auth.AuthMechanismProperties,
			Source:      args.Auth.AuthSource,
		}
		mechanism := args.Auth.AuthMechanism

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
			ClusterClock:  clock,
			HTTPClient:    args.HTTPClient,
		}

		if mechanism == "" {
			// Required for SASL mechanism negotiation during handshake
			handshakeOpts.DBUser = cred.Source + "." + cred.Username
		}

		handshaker = func(driver.Handshaker) driver.Handshaker {
			return auth.Handshaker(nil, handshakeOpts)
		}
	}
	connOpts = append(connOpts, WithHandshaker(handshaker))

	// Dialer
	if args.Dialer != nil {
		connOpts = append(connOpts, WithDialer(
			func(Dialer) Dialer { return args.Dialer },
		))
	}
	// Direct
	if args.Direct != nil && *args.Direct {
		cfgp.Mode = SingleMode
	}

	// HeartbeatInterval
	if args.HeartbeatInterval != nil {
		serverOpts = append(serverOpts, WithHeartbeatInterval(
			func(time.Duration) time.Duration { return *args.HeartbeatInterval },
		))
	}
	// Hosts
	cfgp.SeedList = []string{"localhost:27017"} // default host
	if len(args.Hosts) > 0 {
		cfgp.SeedList = args.Hosts
	}

	// MaxConIdleTime
	if args.MaxConnIdleTime != nil {
		serverOpts = append(serverOpts, WithConnectionPoolMaxIdleTime(
			func(time.Duration) time.Duration { return *args.MaxConnIdleTime },
		))
	}
	// MaxPoolSize
	if args.MaxPoolSize != nil {
		serverOpts = append(
			serverOpts,
			WithMaxConnections(func(uint64) uint64 { return *args.MaxPoolSize }),
		)
	}
	// MinPoolSize
	if args.MinPoolSize != nil {
		serverOpts = append(
			serverOpts,
			WithMinConnections(func(uint64) uint64 { return *args.MinPoolSize }),
		)
	}
	// MaxConnecting
	if args.MaxConnecting != nil {
		serverOpts = append(
			serverOpts,
			WithMaxConnecting(func(uint64) uint64 { return *args.MaxConnecting }),
		)
	}
	// PoolMonitor
	if args.PoolMonitor != nil {
		serverOpts = append(
			serverOpts,
			WithConnectionPoolMonitor(func(*event.PoolMonitor) *event.PoolMonitor { return args.PoolMonitor }),
		)
	}
	// Monitor
	if args.Monitor != nil {
		connOpts = append(connOpts, WithMonitor(
			func(*event.CommandMonitor) *event.CommandMonitor { return args.Monitor },
		))
	}
	// ServerMonitor
	if args.ServerMonitor != nil {
		serverOpts = append(
			serverOpts,
			WithServerMonitor(func(*event.ServerMonitor) *event.ServerMonitor { return args.ServerMonitor }),
		)
		cfgp.ServerMonitor = args.ServerMonitor
	}
	// ReplicaSet
	if args.ReplicaSet != nil {
		cfgp.ReplicaSetName = *args.ReplicaSet
	}
	// ServerSelectionTimeout
	if args.ServerSelectionTimeout != nil {
		cfgp.ServerSelectionTimeout = *args.ServerSelectionTimeout
	}
	//ConnectionTimeout
	if args.ConnectTimeout != nil {
		cfgp.ConnectTimeout = *args.ConnectTimeout
	}
	// TLSConfig
	if args.TLSConfig != nil {
		connOpts = append(connOpts, WithTLSConfig(
			func(*tls.Config) *tls.Config {
				return args.TLSConfig
			},
		))
	}

	// HTTP Client
	if args.HTTPClient != nil {
		connOpts = append(connOpts, WithHTTPClient(
			func(*http.Client) *http.Client {
				return args.HTTPClient
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
	if args.DisableOCSPEndpointCheck != nil {
		connOpts = append(
			connOpts,
			WithDisableOCSPEndpointCheck(func(bool) bool { return *args.DisableOCSPEndpointCheck }),
		)
	}

	// LoadBalanced
	if args.LoadBalanced != nil {
		cfgp.LoadBalanced = *args.LoadBalanced

		serverOpts = append(
			serverOpts,
			WithServerLoadBalanced(func(bool) bool { return *args.LoadBalanced }),
		)
		connOpts = append(
			connOpts,
			WithConnectionLoadBalanced(func(bool) bool { return *args.LoadBalanced }),
		)
	}

	lgr, err := newLogger(args.LoggerOptions)
	if err != nil {
		return nil, err
	}

	serverOpts = append(
		serverOpts,
		withLogger(func() *logger.Logger { return lgr }),
		withServerMonitoringMode(args.ServerMonitoringMode),
	)

	cfgp.logger = lgr

	serverOpts = append(
		serverOpts,
		WithClock(func(*session.ClusterClock) *session.ClusterClock { return clock }),
		WithConnectionOptions(func(...ConnectionOption) []ConnectionOption { return connOpts }))

	cfgp.ServerOpts = serverOpts

	return cfgp, nil
}
