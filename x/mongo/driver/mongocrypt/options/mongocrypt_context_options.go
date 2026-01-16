// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

import (
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
)

// DataKeyOptions specifies options for creating a new data key.
type DataKeyOptions struct {
	KeyAltNames []string
	KeyMaterial []byte
	MasterKey   bsoncore.Document
}

// QueryType describes the type of query the result of Encrypt is used for.
type QueryType int

// These constants specify valid values for QueryType
const (
	QueryTypeEquality QueryType = 1
)

// ExplicitEncryptionOptions specifies options for configuring an explicit encryption context.
type ExplicitEncryptionOptions struct {
	KeyID            *bson.Binary
	KeyAltName       *string
	Algorithm        string
	QueryType        string
	ContentionFactor *int64
	RangeOptions     *ExplicitRangeOptions
	TextOptions      *ExplicitTextOptions
}

// ExplicitRangeOptions specifies options for the range index.
type ExplicitRangeOptions struct {
	Min        *bsoncore.Value
	Max        *bsoncore.Value
	Sparsity   *int64
	TrimFactor *int32
	Precision  *int32
}

// ExplicitTextOptions specifies options for the text query.
type ExplicitTextOptions struct {
	Substring          *SubstringOptions
	Prefix             *PrefixOptions
	Suffix             *SuffixOptions
	CaseSensitive      bool
	DiacriticSensitive bool
}

// SubstringOptions specifies options to support substring queries.
type SubstringOptions struct {
	StrMaxLength      int32
	StrMinQueryLength int32
	StrMaxQueryLength int32
}

// PrefixOptions specifies options to support prefix queries.
type PrefixOptions struct {
	StrMinQueryLength int32
	StrMaxQueryLength int32
}

// SuffixOptions specifies options to support suffix queries.
type SuffixOptions struct {
	StrMinQueryLength int32
	StrMaxQueryLength int32
}

// ExplicitEncryption creates a new ExplicitEncryptionOptions instance.
func ExplicitEncryption() *ExplicitEncryptionOptions {
	return &ExplicitEncryptionOptions{}
}

// SetKeyID sets the key identifier.
func (eeo *ExplicitEncryptionOptions) SetKeyID(keyID bson.Binary) *ExplicitEncryptionOptions {
	eeo.KeyID = &keyID
	return eeo
}

// SetKeyAltName sets the key alternative name.
func (eeo *ExplicitEncryptionOptions) SetKeyAltName(keyAltName string) *ExplicitEncryptionOptions {
	eeo.KeyAltName = &keyAltName
	return eeo
}

// SetAlgorithm specifies an encryption algorithm.
func (eeo *ExplicitEncryptionOptions) SetAlgorithm(algorithm string) *ExplicitEncryptionOptions {
	eeo.Algorithm = algorithm
	return eeo
}

// SetQueryType specifies the query type.
func (eeo *ExplicitEncryptionOptions) SetQueryType(queryType string) *ExplicitEncryptionOptions {
	eeo.QueryType = queryType
	return eeo
}

// SetContentionFactor specifies the contention factor.
func (eeo *ExplicitEncryptionOptions) SetContentionFactor(contentionFactor int64) *ExplicitEncryptionOptions {
	eeo.ContentionFactor = &contentionFactor
	return eeo
}

// SetRangeOptions specifies the range options.
func (eeo *ExplicitEncryptionOptions) SetRangeOptions(ro ExplicitRangeOptions) *ExplicitEncryptionOptions {
	eeo.RangeOptions = &ro
	return eeo
}

// SetTextOptions specifies the text options.
func (eeo *ExplicitEncryptionOptions) SetTextOptions(to ExplicitTextOptions) *ExplicitEncryptionOptions {
	eeo.TextOptions = &to
	return eeo
}

// RewrapManyDataKeyOptions represents all possible options used to decrypt and encrypt all matching data keys with a
// possibly new masterKey.
type RewrapManyDataKeyOptions struct {
	// Provider identifies the new KMS provider. If omitted, encrypting uses the current KMS provider.
	Provider *string

	// MasterKey identifies the new masterKey. If omitted, rewraps with the current masterKey.
	MasterKey bsoncore.Document
}
