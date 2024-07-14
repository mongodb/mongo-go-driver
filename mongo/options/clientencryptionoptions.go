// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

import (
	"crypto/tls"
	"fmt"
	"net/http"

	"go.mongodb.org/mongo-driver/internal/httputil"
)

// ClientEncryptionOptions represents all possible arguments used to configure a ClientEncryption instance.
type ClientEncryptionOptions struct {
	KeyVaultNamespace string
	KmsProviders      map[string]map[string]interface{}
	TLSConfig         map[string]*tls.Config
	HTTPClient        *http.Client
}

// ClientEncryptionOptionsBuilder contains options to configure client
// encryption operations. Each option can be set through setter functions. See
// documentation for each setter function for an explanation of the option.
type ClientEncryptionOptionsBuilder struct {
	Opts []func(*ClientEncryptionOptions) error
}

// ClientEncryption creates a new ClientEncryptionOptions instance.
func ClientEncryption() *ClientEncryptionOptionsBuilder {
	return &ClientEncryptionOptionsBuilder{
		Opts: []func(*ClientEncryptionOptions) error{
			func(arg *ClientEncryptionOptions) error {
				arg.HTTPClient = httputil.DefaultHTTPClient
				return nil
			},
		},
	}
}

// ArgsSetters returns a list of ClientEncryptionArgs setter functions.
func (c *ClientEncryptionOptionsBuilder) ArgsSetters() []func(*ClientEncryptionOptions) error {
	return c.Opts
}

// SetKeyVaultNamespace specifies the namespace of the key vault collection. This is required.
func (c *ClientEncryptionOptionsBuilder) SetKeyVaultNamespace(ns string) *ClientEncryptionOptionsBuilder {
	c.Opts = append(c.Opts, func(args *ClientEncryptionOptions) error {
		args.KeyVaultNamespace = ns
		return nil
	})
	return c
}

// SetKmsProviders specifies options for KMS providers. This is required.
func (c *ClientEncryptionOptionsBuilder) SetKmsProviders(providers map[string]map[string]interface{}) *ClientEncryptionOptionsBuilder {
	c.Opts = append(c.Opts, func(args *ClientEncryptionOptions) error {
		args.KmsProviders = providers
		return nil
	})
	return c
}

// SetTLSConfig specifies tls.Config instances for each KMS provider to use to configure TLS on all connections created
// to the KMS provider.
//
// This should only be used to set custom TLS configurations. By default, the connection will use an empty tls.Config{} with MinVersion set to tls.VersionTLS12.
func (c *ClientEncryptionOptionsBuilder) SetTLSConfig(tlsOpts map[string]*tls.Config) *ClientEncryptionOptionsBuilder {
	c.Opts = append(c.Opts, func(args *ClientEncryptionOptions) error {
		tlsConfigs := make(map[string]*tls.Config)
		for provider, config := range tlsOpts {
			// use TLS min version 1.2 to enforce more secure hash algorithms and advanced cipher suites
			if config.MinVersion == 0 {
				config.MinVersion = tls.VersionTLS12
			}
			tlsConfigs[provider] = config
		}
		args.TLSConfig = tlsConfigs
		return nil
	})
	return c
}

// BuildTLSConfig specifies tls.Config options for each KMS provider to use to configure TLS on all connections created
// to the KMS provider. The input map should contain a mapping from each KMS provider to a document containing the necessary
// options, as follows:
//
//	{
//			"kmip": {
//				"tlsCertificateKeyFile": "foo.pem",
//				"tlsCAFile": "fooCA.pem"
//			}
//	}
//
// Currently, the following TLS options are supported:
//
// 1. "tlsCertificateKeyFile" (or "sslClientCertificateKeyFile"): The "tlsCertificateKeyFile" option specifies a path to
// the client certificate and private key, which must be concatenated into one file.
//
// 2. "tlsCertificateKeyFilePassword" (or "sslClientCertificateKeyPassword"): Specify the password to decrypt the client
// private key file (e.g. "tlsCertificateKeyFilePassword=password").
//
// 3. "tlsCaFile" (or "sslCertificateAuthorityFile"): Specify the path to a single or bundle of certificate authorities
// to be considered trusted when making a TLS connection (e.g. "tlsCaFile=/path/to/caFile").
//
// This should only be used to set custom TLS options. By default, the connection will use an empty tls.Config{} with MinVersion set to tls.VersionTLS12.
func BuildTLSConfig(tlsOpts map[string]interface{}) (*tls.Config, error) {
	// use TLS min version 1.2 to enforce more secure hash algorithms and advanced cipher suites
	cfg := &tls.Config{MinVersion: tls.VersionTLS12}

	for name := range tlsOpts {
		var err error
		switch name {
		case "tlsCertificateKeyFile", "sslClientCertificateKeyFile":
			clientCertPath, ok := tlsOpts[name].(string)
			if !ok {
				return nil, fmt.Errorf("expected %q value to be of type string, got %T", name, tlsOpts[name])
			}
			// apply custom key file password if found, otherwise use empty string
			if keyPwd, found := tlsOpts["tlsCertificateKeyFilePassword"].(string); found {
				_, err = addClientCertFromConcatenatedFile(cfg, clientCertPath, keyPwd)
			} else if keyPwd, found := tlsOpts["sslClientCertificateKeyPassword"].(string); found {
				_, err = addClientCertFromConcatenatedFile(cfg, clientCertPath, keyPwd)
			} else {
				_, err = addClientCertFromConcatenatedFile(cfg, clientCertPath, "")
			}
		case "tlsCertificateKeyFilePassword", "sslClientCertificateKeyPassword":
			continue
		case "tlsCAFile", "sslCertificateAuthorityFile":
			caPath, ok := tlsOpts[name].(string)
			if !ok {
				return nil, fmt.Errorf("expected %q value to be of type string, got %T", name, tlsOpts[name])
			}
			err = addCACertFromFile(cfg, caPath)
		default:
			return nil, fmt.Errorf("unrecognized TLS option %v", name)
		}

		if err != nil {
			return nil, err
		}
	}

	return cfg, nil
}
