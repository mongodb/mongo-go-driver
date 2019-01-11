// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package topologyx

import (
	"bytes"
	"context"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver/auth"
	"go.mongodb.org/mongo-driver/x/mongo/driverx"
	"go.mongodb.org/mongo-driver/x/network/address"
	"go.mongodb.org/mongo-driver/x/network/commandx"
	"go.mongodb.org/mongo-driver/x/network/compressor"
	"go.mongodb.org/mongo-driver/x/network/connstring"
	"go.mongodb.org/mongo-driver/x/network/description"
)

// Option is a configuration option for a topology.
type Option func(*config) error

type config struct {
	mode                   MonitorMode
	replicaSetName         string
	seedList               []string
	serverOpts             []ServerOption
	cs                     connstring.ConnString
	serverSelectionTimeout time.Duration
}

func newConfig(opts ...Option) (*config, error) {
	cfg := &config{
		seedList:               []string{"localhost:27017"},
		serverSelectionTimeout: 30 * time.Second,
	}

	for _, opt := range opts {
		err := opt(cfg)
		if err != nil {
			return nil, err
		}
	}

	return cfg, nil
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
			connOpts = append(connOpts, WithAppName(func(string) string { return cs.AppName }))
		}

		switch cs.Connect {
		case connstring.SingleConnect:
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
			c.serverOpts = append(c.serverOpts, WithMaxConnections(func(uint16) uint16 { return cs.MaxPoolSize }))
			c.serverOpts = append(c.serverOpts, WithMaxIdleConnections(func(uint16) uint16 { return cs.MaxPoolSize }))
		}

		if cs.ReplicaSet != "" {
			c.replicaSetName = cs.ReplicaSet
		}

		var x509Username string
		if cs.SSL {
			tlsConfig := NewTLSConfig()

			if cs.SSLCaFileSet {
				err := tlsConfig.AddCACertFromFile(cs.SSLCaFile)
				if err != nil {
					return err
				}
			}

			if cs.SSLInsecure {
				tlsConfig.SetInsecure(true)
			}

			if cs.SSLClientCertificateKeyFileSet {
				if cs.SSLClientCertificateKeyPasswordSet && cs.SSLClientCertificateKeyPassword != nil {
					tlsConfig.SetClientCertDecryptPassword(cs.SSLClientCertificateKeyPassword)
				}
				s, err := tlsConfig.AddClientCertFromFile(cs.SSLClientCertificateKeyFile)
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

			connOpts = append(connOpts, WithTLSConfig(func(*TLSConfig) *TLSConfig { return tlsConfig }))
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

			// authenticator, err := auth.CreateAuthenticator(cs.AuthMechanism, cred)
			// if err != nil {
			// 	return err
			// }
			//
			// TODO(GODRIVER-617): We need to move the auth handshaker into this package or else we'll have a cyclic dependency (or we need to move the Handshaker interface into the driverx pacakge).
			// connOpts = append(connOpts, WithHandshaker(func(h Handshaker) Handshaker {
			// 	options := &auth.HandshakeOptions{
			// 		AppName:       cs.AppName,
			// 		Authenticator: authenticator,
			// 		Compressors:   cs.Compressors,
			// 	}
			// 	if cs.AuthMechanism == "" {
			// 		// Required for SASL mechanism negotiation during handshake
			// 		options.DBUser = cred.Source + "." + cred.Username
			// 	}
			// 	return auth.Handshaker(h, options)
			// }))
		} else {
			// We need to add a non-auth Handshaker to the connection options
			connOpts = append(connOpts, WithHandshaker(func(h Handshaker) Handshaker {
				return HandshakerFunc(func(ctx context.Context, addr address.Address, conn driverx.Connection) (description.Server, error) {
					// TODO(GODRIVER-617): Make this command once and store it.
					idx, compressors := bsoncore.AppendArrayStart(nil)
					for i, comp := range cs.Compressors {
						compressors = bsoncore.AppendStringElement(compressors, strconv.Itoa(i), comp)
					}
					compressors, _ = bsoncore.AppendArrayEnd(compressors, idx)

					isMasterCmd := commandx.IsMaster{
						Client:      commandx.ClientDoc(cs.AppName),
						Compressors: bsoncore.Value{Type: bsontype.Array, Data: compressors},
					}
					wm, _ := isMasterCmd.MarshalWireMessage(nil, description.SelectedServer{})
					err := conn.WriteWireMessage(ctx, wm)
					if err != nil {
						return description.Server{}, err
					}
					wm, err = conn.ReadWireMessage(ctx, wm[:0])
					if err != nil {
						return description.Server{}, err
					}
					isMaster, err := commandx.IsMasterResult{}.ConvertToResult(wm)
					if err != nil {
						return description.Server{}, err
					}
					return description.NewServer(addr, isMaster), nil
				})
			}))
		}

		if len(cs.Compressors) > 0 {
			comp := make([]compressor.Compressor, 0, len(cs.Compressors))

			for _, c := range cs.Compressors {
				switch c {
				case "snappy":
					comp = append(comp, compressor.CreateSnappy())
				case "zlib":
					zlibComp, err := compressor.CreateZlib(cs.ZlibLevel)
					if err != nil {
						return err
					}

					comp = append(comp, zlibComp)
				}
			}

			connOpts = append(connOpts, WithCompressors(func(compressors []compressor.Compressor) []compressor.Compressor {
				return append(compressors, comp...)
			}))

			c.serverOpts = append(c.serverOpts, WithCompressionOptions(func(opts ...string) []string {
				return append(opts, cs.Compressors...)
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
