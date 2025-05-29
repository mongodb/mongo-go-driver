// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package topology

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/internal/logger"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/auth"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/ocsp"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/operation"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/session"
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

// ConvertToDriverAPIOptions converts a given ServerAPIOptions object from the
// options package to a ServerAPIOptions object from the driver package.
func ConvertToDriverAPIOptions(opts *options.ServerAPIOptions) *driver.ServerAPIOptions {
	driverOpts := driver.NewServerAPIOptions(string(opts.ServerAPIVersion))
	if opts.Strict != nil {
		driverOpts.SetStrict(*opts.Strict)
	}
	if opts.DeprecationErrors != nil {
		driverOpts.SetDeprecationErrors(*opts.DeprecationErrors)
	}
	return driverOpts
}

func newLogger(opts *options.LoggerOptions) (*logger.Logger, error) {
	if opts == nil {
		opts = options.Logger()
	}

	componentLevels := make(map[logger.Component]logger.Level)
	for component, level := range opts.ComponentLevels {
		componentLevels[logger.Component(component)] = logger.Level(level)
	}

	log, err := logger.New(opts.Sink, opts.MaxDocumentLength, componentLevels)
	if err != nil {
		return nil, fmt.Errorf("error creating logger: %w", err)
	}

	return log, nil
}

// convertOIDCArgs converts the internal *driver.OIDCArgs into the equivalent
// public type *options.OIDCArgs.
func convertOIDCArgs(args *driver.OIDCArgs) *options.OIDCArgs {
	if args == nil {
		return nil
	}
	return &options.OIDCArgs{
		Version:      args.Version,
		IDPInfo:      (*options.IDPInfo)(args.IDPInfo),
		RefreshToken: args.RefreshToken,
	}
}

// ConvertCreds takes an [options.Credential] and returns the equivalent
// [driver.Cred].
func ConvertCreds(cred *options.Credential) *driver.Cred {
	if cred == nil {
		return nil
	}

	var oidcMachineCallback auth.OIDCCallback
	if cred.OIDCMachineCallback != nil {
		oidcMachineCallback = func(ctx context.Context, args *driver.OIDCArgs) (*driver.OIDCCredential, error) {
			cred, err := cred.OIDCMachineCallback(ctx, convertOIDCArgs(args))
			return (*driver.OIDCCredential)(cred), err
		}
	}

	var oidcHumanCallback auth.OIDCCallback
	if cred.OIDCHumanCallback != nil {
		oidcHumanCallback = func(ctx context.Context, args *driver.OIDCArgs) (*driver.OIDCCredential, error) {
			cred, err := cred.OIDCHumanCallback(ctx, convertOIDCArgs(args))
			return (*driver.OIDCCredential)(cred), err
		}
	}

	return &auth.Cred{
		Source:              cred.AuthSource,
		Username:            cred.Username,
		Password:            cred.Password,
		PasswordSet:         cred.PasswordSet,
		Props:               cred.AuthMechanismProperties,
		OIDCMachineCallback: oidcMachineCallback,
		OIDCHumanCallback:   oidcHumanCallback,
	}
}

// NewConfig will translate data from client options into a topology config for
// building non-default deployments. Server and topology options are not honored
// if a custom deployment is used.
func NewConfig(opts *options.ClientOptions, clock *session.ClusterClock) (*Config, error) {
	var authenticator driver.Authenticator
	var err error
	if opts.Auth != nil {
		authenticator, err = auth.CreateAuthenticator(
			opts.Auth.AuthMechanism,
			ConvertCreds(opts.Auth),
			opts.HTTPClient,
		)
		if err != nil {
			return nil, fmt.Errorf("error creating authenticator: %w", err)
		}
	}
	return NewConfigFromOptionsWithAuthenticator(opts, clock, authenticator)
}

// NewConfigFromOptionsWithAuthenticator will translate data from client options into a
// topology config for building non-default deployments. Server and topology
// options are not honored if a custom deployment is used. It uses a passed in
// authenticator to authenticate the connection.
func NewConfigFromOptionsWithAuthenticator(opts *options.ClientOptions, clock *session.ClusterClock, authenticator driver.Authenticator) (*Config, error) {
	var serverAPI *driver.ServerAPIOptions

	if err := opts.Validate(); err != nil {
		return nil, err
	}

	var connOpts []ConnectionOption
	var serverOpts []ServerOption

	cfgp := &Config{
		Timeout: opts.Timeout,
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
	if opts.ServerAPIOptions != nil {
		serverAPI = ConvertToDriverAPIOptions(opts.ServerAPIOptions)
		serverOpts = append(serverOpts, WithServerAPI(func(*driver.ServerAPIOptions) *driver.ServerAPIOptions {
			return serverAPI
		}))
	}

	cfgp.URI = opts.GetURI()

	if opts.SRVServiceName != nil {
		cfgp.SRVServiceName = *opts.SRVServiceName
	}

	if opts.SRVMaxHosts != nil {
		cfgp.SRVMaxHosts = *opts.SRVMaxHosts
	}

	// AppName
	var appName string
	if opts.AppName != nil {
		appName = *opts.AppName

		serverOpts = append(serverOpts, WithServerAppName(func(string) string {
			return appName
		}))
	}

	var outerLibraryName, outerLibraryVersion, outerLibraryPlatform string
	if opts.DriverInfo != nil {
		outerLibraryName = opts.DriverInfo.Name
		outerLibraryVersion = opts.DriverInfo.Version
		outerLibraryPlatform = opts.DriverInfo.Platform

		serverOpts = append(serverOpts, WithOuterLibraryName(func(string) string {
			return outerLibraryName
		}))

		serverOpts = append(serverOpts, WithOuterLibraryVersion(func(string) string {
			return outerLibraryVersion
		}))

		serverOpts = append(serverOpts, WithOuterLibraryPlatform(func(string) string {
			return outerLibraryPlatform
		}))
	}

	// Compressors & ZlibLevel
	var comps []string
	if len(opts.Compressors) > 0 {
		comps = opts.Compressors

		connOpts = append(connOpts, WithCompressors(
			func(compressors []string) []string {
				return append(compressors, comps...)
			},
		))

		for _, comp := range comps {
			switch comp {
			case "zlib":
				connOpts = append(connOpts, WithZlibLevel(func(*int) *int {
					return opts.ZlibLevel
				}))
			case "zstd":
				connOpts = append(connOpts, WithZstdLevel(func(*int) *int {
					return opts.ZstdLevel
				}))
			}
		}

		serverOpts = append(serverOpts, WithCompressionOptions(
			func(opts ...string) []string { return append(opts, comps...) },
		))
	}

	var loadBalanced bool
	if opts.LoadBalanced != nil {
		loadBalanced = *opts.LoadBalanced
	}

	// Handshaker
	var handshaker func(driver.Handshaker) driver.Handshaker
	if authenticator != nil {
		handshakeOpts := &auth.HandshakeOptions{
			AppName:              appName,
			Authenticator:        authenticator,
			Compressors:          comps,
			ServerAPI:            serverAPI,
			LoadBalanced:         loadBalanced,
			ClusterClock:         clock,
			OuterLibraryName:     outerLibraryName,
			OuterLibraryVersion:  outerLibraryVersion,
			OuterLibraryPlatform: outerLibraryPlatform,
		}

		if opts.Auth.AuthMechanism == "" {
			// Required for SASL mechanism negotiation during handshake
			handshakeOpts.DBUser = opts.Auth.AuthSource + "." + opts.Auth.Username
		}

		handshaker = func(driver.Handshaker) driver.Handshaker {
			return auth.Handshaker(nil, handshakeOpts)
		}

	} else {
		handshaker = func(driver.Handshaker) driver.Handshaker {
			return operation.NewHello().
				AppName(appName).
				Compressors(comps).
				ClusterClock(clock).
				ServerAPI(serverAPI).
				LoadBalanced(loadBalanced).
				OuterLibraryName(outerLibraryName).
				OuterLibraryVersion(outerLibraryVersion).
				OuterLibraryPlatform(outerLibraryPlatform)
		}
	}

	connOpts = append(connOpts, WithHandshaker(handshaker))

	// Dialer
	if opts.Dialer != nil {
		connOpts = append(connOpts, WithDialer(
			func(Dialer) Dialer { return opts.Dialer },
		))
	}
	// Direct
	if opts.Direct != nil && *opts.Direct {
		cfgp.Mode = SingleMode
	}

	// HeartbeatInterval
	if opts.HeartbeatInterval != nil {
		serverOpts = append(serverOpts, WithHeartbeatInterval(
			func(time.Duration) time.Duration { return *opts.HeartbeatInterval },
		))
	}
	// Hosts
	cfgp.SeedList = []string{"localhost:27017"} // default host
	if len(opts.Hosts) > 0 {
		cfgp.SeedList = opts.Hosts
	}

	// MaxConIdleTime
	if opts.MaxConnIdleTime != nil {
		serverOpts = append(serverOpts, WithConnectionPoolMaxIdleTime(
			func(time.Duration) time.Duration { return *opts.MaxConnIdleTime },
		))
	}
	// MaxPoolSize
	if opts.MaxPoolSize != nil {
		serverOpts = append(
			serverOpts,
			WithMaxConnections(func(uint64) uint64 { return *opts.MaxPoolSize }),
		)
	}
	// MinPoolSize
	if opts.MinPoolSize != nil {
		serverOpts = append(
			serverOpts,
			WithMinConnections(func(uint64) uint64 { return *opts.MinPoolSize }),
		)
	}
	// MaxConnecting
	if opts.MaxConnecting != nil {
		serverOpts = append(
			serverOpts,
			WithMaxConnecting(func(uint64) uint64 { return *opts.MaxConnecting }),
		)
	}
	// PoolMonitor
	if opts.PoolMonitor != nil {
		serverOpts = append(
			serverOpts,
			WithConnectionPoolMonitor(func(*event.PoolMonitor) *event.PoolMonitor { return opts.PoolMonitor }),
		)
	}
	// Monitor
	if opts.Monitor != nil {
		connOpts = append(connOpts, WithMonitor(
			func(*event.CommandMonitor) *event.CommandMonitor { return opts.Monitor },
		))
	}
	// ServerMonitor
	if opts.ServerMonitor != nil {
		serverOpts = append(
			serverOpts,
			WithServerMonitor(func(*event.ServerMonitor) *event.ServerMonitor { return opts.ServerMonitor }),
		)
		cfgp.ServerMonitor = opts.ServerMonitor
	}
	// ReplicaSet
	if opts.ReplicaSet != nil {
		cfgp.ReplicaSetName = *opts.ReplicaSet
	}
	// ServerSelectionTimeout
	if opts.ServerSelectionTimeout != nil {
		cfgp.ServerSelectionTimeout = *opts.ServerSelectionTimeout
	}
	// ConnectionTimeout
	if opts.ConnectTimeout != nil {
		cfgp.ConnectTimeout = *opts.ConnectTimeout
	}
	// TLSConfig
	if opts.TLSConfig != nil {
		connOpts = append(connOpts, WithTLSConfig(
			func(*tls.Config) *tls.Config {
				return opts.TLSConfig
			},
		))
	}

	// HTTP Client
	if opts.HTTPClient != nil {
		connOpts = append(connOpts, WithHTTPClient(
			func(*http.Client) *http.Client {
				return opts.HTTPClient
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
	if opts.DisableOCSPEndpointCheck != nil {
		connOpts = append(
			connOpts,
			WithDisableOCSPEndpointCheck(func(bool) bool { return *opts.DisableOCSPEndpointCheck }),
		)
	}

	// LoadBalanced
	if opts.LoadBalanced != nil {
		cfgp.LoadBalanced = *opts.LoadBalanced

		serverOpts = append(
			serverOpts,
			WithServerLoadBalanced(func(bool) bool { return *opts.LoadBalanced }),
		)
		connOpts = append(
			connOpts,
			WithConnectionLoadBalanced(func(bool) bool { return *opts.LoadBalanced }),
		)
	}

	lgr, err := newLogger(opts.LoggerOptions)
	if err != nil {
		return nil, err
	}

	serverOpts = append(
		serverOpts,
		withLogger(func() *logger.Logger { return lgr }),
		withServerMonitoringMode(opts.ServerMonitoringMode),
	)

	cfgp.logger = lgr

	serverOpts = append(
		serverOpts,
		WithClock(func(*session.ClusterClock) *session.ClusterClock { return clock }),
		WithConnectionOptions(func(...ConnectionOption) []ConnectionOption { return connOpts }))

	cfgp.ServerOpts = serverOpts

	return cfgp, nil
}
