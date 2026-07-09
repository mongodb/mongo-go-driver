// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

import (
	"go.mongodb.org/mongo-driver/v2/bson"
)

// These constants specify valid values for QueryType
// QueryType is used for Queryable Encryption.
const (
	QueryTypeEquality string = "equality"
)

// RangeOptions specifies index options for a Queryable Encryption field supporting "range" queries.
//
// See corresponding setter methods for documentation.
type RangeOptions struct {
	Min        *bson.RawValue
	Max        *bson.RawValue
	Sparsity   *int64
	TrimFactor *int32
	Precision  *int32
}

// RangeOptionsBuilder contains options to configure RangeOptions for queryable
// encryption. Each option can be set through setter functions. See
// documentation for each setter function for an explanation of the option.
type RangeOptionsBuilder struct {
	Opts []func(*RangeOptions) error
}

// Range creates a new RangeOptions instance.
func Range() *RangeOptionsBuilder {
	return &RangeOptionsBuilder{}
}

// List returns a list of RangeOptions setter functions.
func (ro *RangeOptionsBuilder) List() []func(*RangeOptions) error {
	return ro.Opts
}

// SetMin sets the range index minimum value.
func (ro *RangeOptionsBuilder) SetMin(min bson.RawValue) *RangeOptionsBuilder {
	ro.Opts = append(ro.Opts, func(opts *RangeOptions) error {
		opts.Min = &min

		return nil
	})

	return ro
}

// SetMax sets the range index maximum value.
func (ro *RangeOptionsBuilder) SetMax(max bson.RawValue) *RangeOptionsBuilder {
	ro.Opts = append(ro.Opts, func(opts *RangeOptions) error {
		opts.Max = &max

		return nil
	})

	return ro
}

// SetSparsity sets the range index sparsity.
func (ro *RangeOptionsBuilder) SetSparsity(sparsity int64) *RangeOptionsBuilder {
	ro.Opts = append(ro.Opts, func(opts *RangeOptions) error {
		opts.Sparsity = &sparsity

		return nil
	})

	return ro
}

// SetTrimFactor sets the range index trim factor.
func (ro *RangeOptionsBuilder) SetTrimFactor(trimFactor int32) *RangeOptionsBuilder {
	ro.Opts = append(ro.Opts, func(opts *RangeOptions) error {
		opts.TrimFactor = &trimFactor

		return nil
	})

	return ro
}

// SetPrecision sets the range index precision.
func (ro *RangeOptionsBuilder) SetPrecision(precision int32) *RangeOptionsBuilder {
	ro.Opts = append(ro.Opts, func(opts *RangeOptions) error {
		opts.Precision = &precision

		return nil
	})

	return ro
}

// StringOptions specifies index options for a Queryable Encryption field
// supporting prefix, suffix, and substring queries.
//
// See corresponding setter methods for documentation.
type StringOptions struct {
	// Substring specifies options to support substring queries.
	Substring *SubstringOptions

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

// StringOptionsBuilder contains options to configure StringOptions for
// queryable encryption. Each option can be set through setter functions. See
// documentation for each setter function for an explanation of the option.
type StringOptionsBuilder struct {
	Opts []func(*StringOptions) error
}

// String creates a new StringOptions instance.
func String() *StringOptionsBuilder {
	return &StringOptionsBuilder{}
}

// List returns a list of StringOptions setter functions.
func (so *StringOptionsBuilder) List() []func(*StringOptions) error {
	return so.Opts
}

// SetSubstring sets the string index substring value.
func (so *StringOptionsBuilder) SetSubstring(substring SubstringOptions) *StringOptionsBuilder {
	so.Opts = append(so.Opts, func(opts *StringOptions) error {
		opts.Substring = &substring

		return nil
	})

	return so
}

// SetPrefix sets the string index prefix value.
func (so *StringOptionsBuilder) SetPrefix(prefix PrefixOptions) *StringOptionsBuilder {
	so.Opts = append(so.Opts, func(opts *StringOptions) error {
		opts.Prefix = &prefix

		return nil
	})

	return so
}

// SetSuffix sets the string index suffix value.
func (so *StringOptionsBuilder) SetSuffix(suffix SuffixOptions) *StringOptionsBuilder {
	so.Opts = append(so.Opts, func(opts *StringOptions) error {
		opts.Suffix = &suffix

		return nil
	})

	return so
}

// SetCaseSensitive sets the string index caseSensitive value.
func (so *StringOptionsBuilder) SetCaseSensitive(caseSensitive bool) *StringOptionsBuilder {
	so.Opts = append(so.Opts, func(opts *StringOptions) error {
		opts.CaseSensitive = caseSensitive

		return nil
	})

	return so
}

// SetDiacriticSensitive sets the string index diacriticSensitive value.
func (so *StringOptionsBuilder) SetDiacriticSensitive(diacriticSensitive bool) *StringOptionsBuilder {
	so.Opts = append(so.Opts, func(opts *StringOptions) error {
		opts.DiacriticSensitive = diacriticSensitive

		return nil
	})

	return so
}

// EncryptOptions represents arguments to explicitly encrypt a value.
//
// See corresponding setter methods for documentation.
type EncryptOptions struct {
	KeyID            *bson.Binary
	KeyAltName       *string
	Algorithm        string
	QueryType        string
	ContentionFactor *int64
	RangeOptions     *RangeOptionsBuilder
	StringOptions    *StringOptionsBuilder
}

// EncryptOptionsBuilder contains options to configure Encryptopts for
// queryeable encryption. Each option can be set through setter functions. See
// documentation for each setter function for an explanation of the option.
type EncryptOptionsBuilder struct {
	Opts []func(*EncryptOptions) error
}

// List returns a list of EncryptOptions setter functions.
func (e *EncryptOptionsBuilder) List() []func(*EncryptOptions) error {
	return e.Opts
}

// Encrypt creates a new EncryptOptions instance.
func Encrypt() *EncryptOptionsBuilder {
	return &EncryptOptionsBuilder{}
}

// SetKeyID specifies an _id of a data key. This should be a UUID (a bson.Binary with subtype 4).
func (e *EncryptOptionsBuilder) SetKeyID(keyID bson.Binary) *EncryptOptionsBuilder {
	e.Opts = append(e.Opts, func(opts *EncryptOptions) error {
		opts.KeyID = &keyID

		return nil
	})
	return e
}

// SetKeyAltName identifies a key vault document by 'keyAltName'.
func (e *EncryptOptionsBuilder) SetKeyAltName(keyAltName string) *EncryptOptionsBuilder {
	e.Opts = append(e.Opts, func(opts *EncryptOptions) error {
		opts.KeyAltName = &keyAltName

		return nil
	})

	return e
}

// SetAlgorithm specifies an algorithm to use for encryption. This should be one
// of the following:
//
//   - AEAD_AES_256_CBC_HMAC_SHA_512-Deterministic
//   - AEAD_AES_256_CBC_HMAC_SHA_512-Random
//   - Indexed
//   - Unindexed
//   - Range
//   - String
//
// This is required.
//
// Indexed, Range, and String are used for Queryable Encryption.
func (e *EncryptOptionsBuilder) SetAlgorithm(algorithm string) *EncryptOptionsBuilder {
	e.Opts = append(e.Opts, func(opts *EncryptOptions) error {
		opts.Algorithm = algorithm

		return nil
	})

	return e
}

// SetQueryType specifies the intended query type. It is only valid to set when
// algorithm is "Indexed", "Range", or "String". This should be one of the
// following:
//
//   - equality
//   - range
//   - prefix / prefixPreview (used for the $encStrStartsWith operator)
//   - suffix / suffixPreview (used for the $encStrEndsWith operator)
//   - substring / substringPreview (used for the $encStrContains operator)
//
// QueryType is used for Queryable Encryption.
//
// Beta: "prefixPreview", "suffixPreview", and "substringPreview" are preview
// features and should only be used for experimental workloads. They are not
// intended for public use and are subject to breaking changes.
func (e *EncryptOptionsBuilder) SetQueryType(queryType string) *EncryptOptionsBuilder {
	e.Opts = append(e.Opts, func(opts *EncryptOptions) error {
		opts.QueryType = queryType

		return nil
	})

	return e
}

// SetContentionFactor specifies the contention factor. It is only valid to set if algorithm is "Indexed".
// ContentionFactor is used for Queryable Encryption.
func (e *EncryptOptionsBuilder) SetContentionFactor(contentionFactor int64) *EncryptOptionsBuilder {
	e.Opts = append(e.Opts, func(opts *EncryptOptions) error {
		opts.ContentionFactor = &contentionFactor

		return nil
	})

	return e
}

// SetRangeOptions specifies the options to use for explicit encryption with range. It is only valid to set if algorithm is "range".
func (e *EncryptOptionsBuilder) SetRangeOptions(ro *RangeOptionsBuilder) *EncryptOptionsBuilder {
	e.Opts = append(e.Opts, func(opts *EncryptOptions) error {
		opts.RangeOptions = ro

		return nil
	})

	return e
}

// SetStringOptions specifies the options to use for string queries.
func (e *EncryptOptionsBuilder) SetStringOptions(so *StringOptionsBuilder) *EncryptOptionsBuilder {
	e.Opts = append(e.Opts, func(opts *EncryptOptions) error {
		opts.StringOptions = so

		return nil
	})

	return e
}
