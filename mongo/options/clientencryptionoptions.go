// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

import (
	"crypto/tls"
	"errors"
	"fmt"
)

// ClientEncryptionOptions represents all possible options used to configure a ClientEncryption instance.
type ClientEncryptionOptions struct {
	KeyVaultNamespace string
	KmsProviders      map[string]map[string]interface{}
	TLSConfig         map[string]*tls.Config
}

// ClientEncryption creates a new ClientEncryptionOptions instance.
func ClientEncryption() *ClientEncryptionOptions {
	return &ClientEncryptionOptions{}
}

// SetKeyVaultNamespace specifies the namespace of the key vault collection. This is required.
func (c *ClientEncryptionOptions) SetKeyVaultNamespace(ns string) *ClientEncryptionOptions {
	c.KeyVaultNamespace = ns
	return c
}

// SetKmsProviders specifies options for KMS providers. This is required.
func (c *ClientEncryptionOptions) SetKmsProviders(providers map[string]map[string]interface{}) *ClientEncryptionOptions {
	c.KmsProviders = providers
	return c
}

// SetTLSOptions specifies tls.Config instances for each KMS provider to use to configure TLS on all connections created
// to the cluster. The input map should contain a mapping from each KMS provider to a document containing the necessary 
// options, as follows:
//
// {
//		"kmip": {
//			"tlsCertificateKeyFile": "foo.pem",
// 			"tlsCAFile": "fooCA.pem"
//		}
// }
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
// This should only be used to set custom TLS options. By default, the connection will use an empty tls.Config{}.
func (c *ClientEncryptionOptions) SetTLSOptions(tlsOpts map[string]map[string]interface{}) (*ClientEncryptionOptions, error) {
	tlsConfig := make(map[string]*tls.Config)
	tlsConfig, err := applyTLSOptions(tlsOpts, tlsConfig)
	c.TLSConfig = tlsConfig
	return c, err
}

// MergeClientEncryptionOptions combines the argued ClientEncryptionOptions in a last-one wins fashion.
func MergeClientEncryptionOptions(opts ...*ClientEncryptionOptions) *ClientEncryptionOptions {
	ceo := ClientEncryption()
	for _, opt := range opts {
		if opt == nil {
			continue
		}

		if opt.KeyVaultNamespace != "" {
			ceo.KeyVaultNamespace = opt.KeyVaultNamespace
		}
		if opt.KmsProviders != nil {
			ceo.KmsProviders = opt.KmsProviders
		}
		if opt.TLSConfig != nil {
			ceo.TLSConfig = opt.TLSConfig
		}
	}

	return ceo
}

// Used by both ClientEncryptionOptions.SetTLSOptions() and AutoEncryptionOptions.SetTLSOptions()
func applyTLSOptions(tlsOpts map[string]map[string]interface{}, tlsConfigs map[string]*tls.Config) (map[string]*tls.Config, error) {
	for provider, opts := range tlsOpts {
		// use TLS min version 1.2 to enforce more secure hash algorithms and advanced cipher suites
		cfg := &tls.Config{MinVersion: tls.VersionTLS12}
		
		for cert := range opts {
			var err error
			switch cert {
			case "tlsCertificateKeyFile", "sslClientCertificateKeyFile":
				clientCertPath, ok := opts[cert].(string)
				if !ok {
					return tlsConfigs, fmt.Errorf("expected %q value to be of type string, got %T", cert, opts[cert])
				}
				// apply custom key file password if found, otherwise use empty string
				if keyPwd, found := opts["tlsCertificateKeyFilePassword"].(string); found {
					_, err = addClientCertFromConcatenatedFile(cfg, clientCertPath, keyPwd)
				} else if keyPwd, found := opts["sslClientCertificateKeyPassword"].(string); found {
					_, err = addClientCertFromConcatenatedFile(cfg, clientCertPath, keyPwd)
				} else {
					_, err = addClientCertFromConcatenatedFile(cfg, clientCertPath, "")
				}
			case "tlsCertificateKeyFilePassword", "sslClientCertificateKeyPassword":
				continue
			case "tlsCAFile", "sslCertificateAuthorityFile":
				caPath, ok := opts[cert].(string)
				if !ok {
					return tlsConfigs, fmt.Errorf("expected %q value to be of type string, got %T", cert, opts[cert])
				}
				err = addCACertFromFile(cfg, caPath)
			default:
				return tlsConfigs, errors.New(fmt.Sprintf("unrecognized TLS option %v for %v.", cert, provider))
			}
			
			if err != nil {
				return tlsConfigs, err
			}
		}

		tlsConfigs[provider] = cfg
	}
	return tlsConfigs, nil
}
