// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package topology

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
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

func newConfigPrototype(
	co *options.ClientOptions,
	opts ...Option,
) (*config, error) {

	fmt.Println("1.a")

	var clusterClock *session.ClusterClock = nil
	var defaultMaxPoolSize uint64 = 100
	var serverAPI *driver.ServerAPIOptions = nil

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
		// convert passed in options to driver form for client.
		// ! c.serverAPI = convertToDriverAPIOptions(opts.ServerAPIOptions)

		serverOpts = append(serverOpts, WithServerAPI(func(*driver.ServerAPIOptions) *driver.ServerAPIOptions {
			return convertToDriverAPIOptions(co.ServerAPIOptions)
		}))
	}

	// ClusterClock
	// ! c.clock = new(session.ClusterClock)

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
			ClusterClock:  clusterClock,
			ServerAPI:     serverAPI,
			LoadBalanced:  loadBalanced,
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
		fmt.Printf("hosts from topology options: %v", co.Hosts)
		hosts = co.Hosts
	}
	topologyOpts = append(topologyOpts, WithSeedList(
		func(...string) []string { return hosts },
	))
	// LocalThreshold
	// ! c.localThreshold = defaultLocalThreshold
	//if opts.LocalThreshold != nil {
	//	c.localThreshold = *opts.LocalThreshold
	//}
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
		// ! c.monitor = co.Monitor
		connOpts = append(connOpts, WithMonitor(
			func(*event.CommandMonitor) *event.CommandMonitor { return co.Monitor },
		))
	}
	// ServerMonitor
	if co.ServerMonitor != nil {
		// ! c.serverMonitor = opts.ServerMonitor
		serverOpts = append(
			serverOpts,
			WithServerMonitor(func(*event.ServerMonitor) *event.ServerMonitor { return co.ServerMonitor }),
		)

		topologyOpts = append(
			topologyOpts,
			WithTopologyServerMonitor(func(*event.ServerMonitor) *event.ServerMonitor { return co.ServerMonitor }),
		)
	}
	// ReadConcern
	// ! c.readConcern = readconcern.New()
	//if co.ReadConcern != nil {
	//	c.readConcern = opts.ReadConcern
	//}
	// ReadPreference
	//c.readPreference = readpref.Primary()
	//if opts.ReadPreference != nil {
	//	c.readPreference = opts.ReadPreference
	//}
	// Registry
	//c.registry = bson.DefaultRegistry
	//if opts.Registry != nil {
	//	c.registry = opts.Registry
	//}
	// ReplicaSet
	if co.ReplicaSet != nil {
		topologyOpts = append(topologyOpts, WithReplicaSetName(
			func(string) string { return *co.ReplicaSet },
		))
	}
	// RetryWrites
	//c.retryWrites = true // retry writes on by default
	//if opts.RetryWrites != nil {
	//	c.retryWrites = *opts.RetryWrites
	//}
	//c.retryReads = true
	//if opts.RetryReads != nil {
	//	c.retryReads = *opts.RetryReads
	//}
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
	// Timeout
	//c.timeout = opts.Timeout
	// TLSConfig
	if co.TLSConfig != nil {
		connOpts = append(connOpts, WithTLSConfig(
			func(*tls.Config) *tls.Config {
				return co.TLSConfig
			},
		))
	}
	// WriteConcern
	//if opts.WriteConcern != nil {
	//	c.writeConcern = opts.WriteConcern
	//}
	// AutoEncryptionOptions
	//if opts.AutoEncryptionOptions != nil {
	//	if err := c.configureAutoEncryption(opts); err != nil {
	//		return err
	//	}
	//} else {
	//	c.cryptFLE = opts.Crypt
	//}

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
	fmt.Println(1)

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
		WithConnectionOptions(func(...ConnectionOption) []ConnectionOption { return connOpts }),
	)
	topologyOpts = append(topologyOpts, WithServerOptions(
		func(...ServerOption) []ServerOption { return serverOpts },
	))
	// ! c.topologyOptions = topologyOpts

	// Deployment
	if co.Deployment != nil {
		// topology options: WithSeedlist, WithURI, WithSRVServiceName, WithSRVMaxHosts, and WithServerOptions
		// server options: WithClock and WithConnectionOptions + default maxPoolSize
		if len(serverOpts) > 2+defaultOptions || len(topologyOpts) > 5 {
			return nil, errors.New("cannot specify topology or server options with a deployment")
		}
		// ! c.deployment = opts.Deployment
	}
	return newConfig(topologyOpts...)
}

func X(uri string) Option {
	return func(c *config) error {

		opts := new(options.ClientOptions)
		opts = opts.ApplyURI(uri)

		fmt.Printf("options: %+v\n", opts)
		return nil
	}
}

func WithClientOptions(opts ...options.ClientOptions) Option {
	return func(c *config) error {
		return nil
	}
}

// WithConnString configures the topology using the connection string.
func WithConnString(fn func(connstring.ConnString) connstring.ConnString) Option {
	return func(c *config) error {
		cs := fn(c.cs)
		c.cs = cs

		if cs.ServerSelectionTimeoutSet {
			c.serverSelectionTimeout = cs.ServerSelectionTimeout
		}

		var connOpts []ConnectionOption

		if cs.AppName != "" {
			c.serverOpts = append(c.serverOpts, WithServerAppName(func(string) string { return cs.AppName }))
		}

		if cs.Connect == connstring.SingleConnect || (cs.DirectConnectionSet && cs.DirectConnection) {
			c.mode = SingleMode
		}

		c.seedList = cs.Hosts

		if cs.ConnectTimeout > 0 {
			c.serverOpts = append(c.serverOpts, WithHeartbeatTimeout(func(time.Duration) time.Duration { return cs.ConnectTimeout }))
			connOpts = append(connOpts, WithConnectTimeout(func(time.Duration) time.Duration { return cs.ConnectTimeout }))
		}

		if cs.SocketTimeoutSet {
			connOpts = append(
				connOpts,
				WithReadTimeout(func(time.Duration) time.Duration { return cs.SocketTimeout }),
				WithWriteTimeout(func(time.Duration) time.Duration { return cs.SocketTimeout }),
			)
		}

		if cs.HeartbeatInterval > 0 {
			c.serverOpts = append(c.serverOpts, WithHeartbeatInterval(func(time.Duration) time.Duration { return cs.HeartbeatInterval }))
		}

		if cs.MaxConnIdleTime > 0 {
			connOpts = append(connOpts, WithIdleTimeout(func(time.Duration) time.Duration { return cs.MaxConnIdleTime }))
		}

		if cs.MaxPoolSizeSet {
			c.serverOpts = append(c.serverOpts, WithMaxConnections(func(uint64) uint64 { return cs.MaxPoolSize }))
		}

		if cs.MinPoolSizeSet {
			c.serverOpts = append(c.serverOpts, WithMinConnections(func(u uint64) uint64 { return cs.MinPoolSize }))
		}

		if cs.ReplicaSet != "" {
			c.replicaSetName = cs.ReplicaSet
		}

		var x509Username string
		if cs.SSL {
			tlsConfig := new(tls.Config)

			if cs.SSLCaFileSet {
				err := addCACertFromFile(tlsConfig, cs.SSLCaFile)
				if err != nil {
					return err
				}
			}

			if cs.SSLInsecure {
				tlsConfig.InsecureSkipVerify = true
			}

			if cs.SSLClientCertificateKeyFileSet {
				var keyPasswd string
				if cs.SSLClientCertificateKeyPasswordSet && cs.SSLClientCertificateKeyPassword != nil {
					keyPasswd = cs.SSLClientCertificateKeyPassword()
				}
				s, err := addClientCertFromFile(tlsConfig, cs.SSLClientCertificateKeyFile, keyPasswd)
				if err != nil {
					return err
				}

				// The Go x509 package gives the subject with the pairs in reverse order that we want.
				pairs := strings.Split(s, ",")
				b := bytes.NewBufferString("")

				for i := len(pairs) - 1; i >= 0; i-- {
					b.WriteString(pairs[i])

					if i > 0 {
						b.WriteString(",")
					}
				}

				x509Username = b.String()
			}

			connOpts = append(connOpts, WithTLSConfig(func(*tls.Config) *tls.Config { return tlsConfig }))
		}

		if cs.Username != "" || cs.AuthMechanism == auth.MongoDBX509 || cs.AuthMechanism == auth.GSSAPI {
			cred := &auth.Cred{
				Source:      "admin",
				Username:    cs.Username,
				Password:    cs.Password,
				PasswordSet: cs.PasswordSet,
				Props:       cs.AuthMechanismProperties,
			}

			if cs.AuthSource != "" {
				cred.Source = cs.AuthSource
			} else {
				switch cs.AuthMechanism {
				case auth.MongoDBX509:
					if cred.Username == "" {
						cred.Username = x509Username
					}
					fallthrough
				case auth.GSSAPI, auth.PLAIN:
					cred.Source = "$external"
				default:
					cred.Source = cs.Database
				}
			}

			authenticator, err := auth.CreateAuthenticator(cs.AuthMechanism, cred)
			if err != nil {
				return err
			}

			connOpts = append(connOpts, WithHandshaker(func(h Handshaker) Handshaker {
				options := &auth.HandshakeOptions{
					AppName:       cs.AppName,
					Authenticator: authenticator,
					Compressors:   cs.Compressors,
					LoadBalanced:  cs.LoadBalancedSet && cs.LoadBalanced,
				}
				if cs.AuthMechanism == "" {
					// Required for SASL mechanism negotiation during handshake
					options.DBUser = cred.Source + "." + cred.Username
				}
				return auth.Handshaker(h, options)
			}))
		} else {
			// We need to add a non-auth Handshaker to the connection options
			connOpts = append(connOpts, WithHandshaker(func(h driver.Handshaker) driver.Handshaker {
				return operation.NewHello().
					AppName(cs.AppName).
					Compressors(cs.Compressors).
					LoadBalanced(cs.LoadBalancedSet && cs.LoadBalanced)
			}))
		}

		if len(cs.Compressors) > 0 {
			connOpts = append(connOpts, WithCompressors(func(compressors []string) []string {
				return append(compressors, cs.Compressors...)
			}))

			for _, comp := range cs.Compressors {
				switch comp {
				case "zlib":
					connOpts = append(connOpts, WithZlibLevel(func(level *int) *int {
						return &cs.ZlibLevel
					}))
				case "zstd":
					connOpts = append(connOpts, WithZstdLevel(func(level *int) *int {
						return &cs.ZstdLevel
					}))
				}
			}

			c.serverOpts = append(c.serverOpts, WithCompressionOptions(func(opts ...string) []string {
				return append(opts, cs.Compressors...)
			}))
		}

		// LoadBalanced
		if cs.LoadBalancedSet {
			c.loadBalanced = cs.LoadBalanced
			c.serverOpts = append(c.serverOpts, WithServerLoadBalanced(func(bool) bool {
				return cs.LoadBalanced
			}))
			connOpts = append(connOpts, WithConnectionLoadBalanced(func(bool) bool {
				return cs.LoadBalanced
			}))
		}

		if len(connOpts) > 0 {
			c.serverOpts = append(c.serverOpts, WithConnectionOptions(func(opts ...ConnectionOption) []ConnectionOption {
				return append(opts, connOpts...)
			}))
		}

		return nil
	}
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

// addCACertFromFile adds a root CA certificate to the configuration given a path
// to the containing file.
func addCACertFromFile(cfg *tls.Config, file string) error {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	certBytes, err := loadCert(data)
	if err != nil {
		return err
	}

	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return err
	}

	if cfg.RootCAs == nil {
		cfg.RootCAs = x509.NewCertPool()
	}

	cfg.RootCAs.AddCert(cert)

	return nil
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

// addClientCertFromFile adds a client certificate to the configuration given a path to the
// containing file and returns the certificate's subject name.
func addClientCertFromFile(cfg *tls.Config, clientFile, keyPasswd string) (string, error) {
	data, err := ioutil.ReadFile(clientFile)
	if err != nil {
		return "", err
	}

	var currentBlock *pem.Block
	var certBlock, certDecodedBlock, keyBlock []byte

	remaining := data
	start := 0
	for {
		currentBlock, remaining = pem.Decode(remaining)
		if currentBlock == nil {
			break
		}

		if currentBlock.Type == "CERTIFICATE" {
			certBlock = data[start : len(data)-len(remaining)]
			certDecodedBlock = currentBlock.Bytes
			start += len(certBlock)
		} else if strings.HasSuffix(currentBlock.Type, "PRIVATE KEY") {
			if keyPasswd != "" && x509.IsEncryptedPEMBlock(currentBlock) {
				var encoded bytes.Buffer
				buf, err := x509.DecryptPEMBlock(currentBlock, []byte(keyPasswd))
				if err != nil {
					return "", err
				}

				pem.Encode(&encoded, &pem.Block{Type: currentBlock.Type, Bytes: buf})
				keyBlock = encoded.Bytes()
				start = len(data) - len(remaining)
			} else {
				keyBlock = data[start : len(data)-len(remaining)]
				start += len(keyBlock)
			}
		}
	}
	if len(certBlock) == 0 {
		return "", fmt.Errorf("failed to find CERTIFICATE")
	}
	if len(keyBlock) == 0 {
		return "", fmt.Errorf("failed to find PRIVATE KEY")
	}

	cert, err := tls.X509KeyPair(certBlock, keyBlock)
	if err != nil {
		return "", err
	}

	cfg.Certificates = append(cfg.Certificates, cert)

	// The documentation for the tls.X509KeyPair indicates that the Leaf certificate is not
	// retained.
	crt, err := x509.ParseCertificate(certDecodedBlock)
	if err != nil {
		return "", err
	}

	return crt.Subject.String(), nil
}
