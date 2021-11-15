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

// AutoEncryptionOptions represents options used to configure auto encryption/decryption behavior for a mongo.Client
// instance.
//
// Automatic encryption is an enterprise only feature that only applies to operations on a collection. Automatic
// encryption is not supported for operations on a database or view, and operations that are not bypassed will result
// in error. Too bypass automatic encryption for all operations, set BypassAutoEncryption=true.
//
// Auto encryption requires the authenticated user to have the listCollections privilege action.
//
// If automatic encryption fails on an operation, use a MongoClient configured with bypassAutoEncryption=true and use
// ClientEncryption.encrypt() to manually encrypt values.
//
// Enabling Client Side Encryption reduces the maximum document and message size (using a maxBsonObjectSize of 2MiB and
// maxMessageSizeBytes of 6MB) and may have a negative performance impact.
type AutoEncryptionOptions struct {
	KeyVaultClientOptions *ClientOptions
	KeyVaultNamespace     string
	KmsProviders          map[string]map[string]interface{}
	SchemaMap             map[string]interface{}
	BypassAutoEncryption  *bool
	ExtraOptions          map[string]interface{}
	TLSConfig             map[string]tls.Config
}

// AutoEncryption creates a new AutoEncryptionOptions configured with default values.
func AutoEncryption() *AutoEncryptionOptions {
	return &AutoEncryptionOptions{}
}

// SetKeyVaultClientOptions specifies options for the client used to communicate with the key vault collection.
//
// If this is set, it is used to create an internal mongo.Client.
// Otherwise, if the target mongo.Client being configured has an unlimited connection pool size (i.e. maxPoolSize=0),
// it is reused to interact with the key vault collection.
// Otherwise, if the target mongo.Client has a limited connection pool size, a separate internal mongo.Client is used
// (and created if necessary). The internal mongo.Client may be shared during automatic encryption (if
// BypassAutomaticEncryption is false). The internal mongo.Client is configured with the same options as the target
// mongo.Client except minPoolSize is set to 0 and AutoEncryptionOptions is omitted.
func (a *AutoEncryptionOptions) SetKeyVaultClientOptions(opts *ClientOptions) *AutoEncryptionOptions {
	a.KeyVaultClientOptions = opts
	return a
}

// SetKeyVaultNamespace specifies the namespace of the key vault collection. This is required.
func (a *AutoEncryptionOptions) SetKeyVaultNamespace(ns string) *AutoEncryptionOptions {
	a.KeyVaultNamespace = ns
	return a
}

// SetKmsProviders specifies options for KMS providers. This is required.
func (a *AutoEncryptionOptions) SetKmsProviders(providers map[string]map[string]interface{}) *AutoEncryptionOptions {
	a.KmsProviders = providers
	return a
}

// SetSchemaMap specifies a map from namespace to local schema document. Schemas supplied in the schemaMap only apply
// to configuring automatic encryption for client side encryption. Other validation rules in the JSON schema will not
// be enforced by the driver and will result in an error.
//
// Supplying a schemaMap provides more security than relying on JSON Schemas obtained from the server. It protects
// against a malicious server advertising a false JSON Schema, which could trick the client into sending unencrypted
// data that should be encrypted.
func (a *AutoEncryptionOptions) SetSchemaMap(schemaMap map[string]interface{}) *AutoEncryptionOptions {
	a.SchemaMap = schemaMap
	return a
}

// SetBypassAutoEncryption specifies whether or not auto encryption should be done.
//
// If this is unset or false and target mongo.Client being configured has an unlimited connection pool size
// (i.e. maxPoolSize=0), it is reused in the process of auto encryption.
// Otherwise, if the target mongo.Client has a limited connection pool size, a separate internal mongo.Client is used
// (and created if necessary). The internal mongo.Client may be shared for key vault operations (if KeyVaultClient is
// unset). The internal mongo.Client is configured with the same options as the target mongo.Client except minPoolSize
// is set to 0 and AutoEncryptionOptions is omitted.
func (a *AutoEncryptionOptions) SetBypassAutoEncryption(bypass bool) *AutoEncryptionOptions {
	a.BypassAutoEncryption = &bypass
	return a
}

// SetExtraOptions specifies a map of options to configure the mongocryptd process.
func (a *AutoEncryptionOptions) SetExtraOptions(extraOpts map[string]interface{}) *AutoEncryptionOptions {
	a.ExtraOptions = extraOpts
	return a
}

// TODO: check to return nil if error
func (a *AutoEncryptionOptions) SetTLSConfig(tlsOpts map[string]map[string]interface{}) (*AutoEncryptionOptions, error) {
	for key, doc := range tlsOpts {
		if key != "tlsCertificateKeyFile" || key != "tlsCertificateKeyFilePassword" || key != "tlsCAFile" {
			return a, errors.New(fmt.Sprintf("Error setting TLS option %v for %v.", "TODO", key))
		}

		var cfg tls.Config
		if clientCertPath, found := doc["tlsCertificateKeyFile"].(string); found {
			if keyPwd, found := doc["tlsCertificateKeyFilePassword"].(string); found {
				_, err := addClientCertFromConcatenatedFile(&cfg, clientCertPath, keyPwd)
				if err != nil {
					return a, err
				}
			} else {
				_, err := addClientCertFromConcatenatedFile(&cfg, clientCertPath, "")
				if err != nil {
					return a, err
				}
			}	
		}
	
		if CApath, found := doc["tlsCAFile"].(string); found {
			err := addCACertFromFile(&cfg, CApath)
			if err != nil {
				return a , err
			}
		}
		
		a.TLSConfig[key] = cfg
	}
	return a, nil
}

// MergeAutoEncryptionOptions combines the argued AutoEncryptionOptions in a last-one wins fashion.
func MergeAutoEncryptionOptions(opts ...*AutoEncryptionOptions) *AutoEncryptionOptions {
	aeo := AutoEncryption()
	for _, opt := range opts {
		if opt == nil {
			continue
		}

		if opt.KeyVaultClientOptions != nil {
			aeo.KeyVaultClientOptions = opt.KeyVaultClientOptions
		}
		if opt.KeyVaultNamespace != "" {
			aeo.KeyVaultNamespace = opt.KeyVaultNamespace
		}
		if opt.KmsProviders != nil {
			aeo.KmsProviders = opt.KmsProviders
		}
		if opt.SchemaMap != nil {
			aeo.SchemaMap = opt.SchemaMap
		}
		if opt.BypassAutoEncryption != nil {
			aeo.BypassAutoEncryption = opt.BypassAutoEncryption
		}
		if opt.ExtraOptions != nil {
			aeo.ExtraOptions = opt.ExtraOptions
		}
		if opt.TLSConfig != nil {
			aeo.TLSConfig = opt.TLSConfig
		}
	}

	return aeo
}
