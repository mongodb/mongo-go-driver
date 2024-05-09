// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

import (
	"go.mongodb.org/mongo-driver/bson"
)

// These constants specify valid values for QueryType
// QueryType is used for Queryable Encryption.
const (
	QueryTypeEquality string = "equality"
)

// RangeArgs specifies index options for a Queryable Encryption field supporting
// "rangePreview" queries. Beta: The Range algorithm is experimental only. It is
// not intended for public use. It is subject to breaking changes.
type RangeArgs struct {
	Min       *bson.RawValue
	Max       *bson.RawValue
	Sparsity  int64
	Precision *int32
}

// RangeOptions contains options to configure RangeArgs for queryeable
// encryption. Each option can be set through setter functions. See
// documentation for each setter function for an explanation of the option.
type RangeOptions struct {
	Opts []func(*RangeArgs) error
}

// Range creates a new RangeOptions instance.
func Range() *RangeOptions {
	return &RangeOptions{}
}

// ArgsSetters returns a list of RangeArgs setter functions.
func (ro *RangeOptions) ArgsSetters() []func(*RangeArgs) error {
	return ro.Opts
}

// EncryptArgs represents options to explicitly encrypt a value.
type EncryptArgs struct {
	KeyID            *bson.Binary
	KeyAltName       *string
	Algorithm        string
	QueryType        string
	ContentionFactor *int64
	RangeOptions     *RangeOptions
}

// EncryptOptions contains options to configure EncryptArgs for queryeable
// encryption. Each option can be set through setter functions. See
// documentation for each setter function for an explanation of the option.
type EncryptOptions struct {
	Opts []func(*EncryptArgs) error
}

// ArgsSetters returns a list of EncryptArgs setter functions.
func (e *EncryptOptions) ArgsSetters() []func(*EncryptArgs) error {
	return e.Opts
}

// Encrypt creates a new EncryptOptions instance.
func Encrypt() *EncryptOptions {
	return &EncryptOptions{}
}

// SetKeyID specifies an _id of a data key. This should be a UUID (a primitive.Binary with subtype 4).
func (e *EncryptOptions) SetKeyID(keyID bson.Binary) *EncryptOptions {
	e.Opts = append(e.Opts, func(args *EncryptArgs) error {
		args.KeyID = &keyID

		return nil
	})
	return e
}

// SetKeyAltName identifies a key vault document by 'keyAltName'.
func (e *EncryptOptions) SetKeyAltName(keyAltName string) *EncryptOptions {
	e.Opts = append(e.Opts, func(args *EncryptArgs) error {
		args.KeyAltName = &keyAltName

		return nil
	})

	return e
}

// SetAlgorithm specifies an algorithm to use for encryption. This should be one of the following:
// - AEAD_AES_256_CBC_HMAC_SHA_512-Deterministic
// - AEAD_AES_256_CBC_HMAC_SHA_512-Random
// - Indexed
// - Unindexed
// This is required.
// Indexed and Unindexed are used for Queryable Encryption.
func (e *EncryptOptions) SetAlgorithm(algorithm string) *EncryptOptions {
	e.Opts = append(e.Opts, func(args *EncryptArgs) error {
		args.Algorithm = algorithm

		return nil
	})

	return e
}

// SetQueryType specifies the intended query type. It is only valid to set if algorithm is "Indexed".
// This should be one of the following:
// - equality
// QueryType is used for Queryable Encryption.
func (e *EncryptOptions) SetQueryType(queryType string) *EncryptOptions {
	e.Opts = append(e.Opts, func(args *EncryptArgs) error {
		args.QueryType = queryType

		return nil
	})

	return e
}

// SetContentionFactor specifies the contention factor. It is only valid to set if algorithm is "Indexed".
// ContentionFactor is used for Queryable Encryption.
func (e *EncryptOptions) SetContentionFactor(contentionFactor int64) *EncryptOptions {
	e.Opts = append(e.Opts, func(args *EncryptArgs) error {
		args.ContentionFactor = &contentionFactor

		return nil
	})

	return e
}

// SetRangeOptions specifies the options to use for explicit encryption with range. It is only valid to set if algorithm is "rangePreview".
// Beta: The Range algorithm is experimental only. It is not intended for public use. It is subject to breaking changes.
func (e *EncryptOptions) SetRangeOptions(ro *RangeOptions) *EncryptOptions {
	e.Opts = append(e.Opts, func(args *EncryptArgs) error {
		args.RangeOptions = ro

		return nil
	})

	return e
}

// SetMin sets the range index minimum value.
// Beta: The Range algorithm is experimental only. It is not intended for public use. It is subject to breaking changes.
func (ro *RangeOptions) SetMin(min bson.RawValue) *RangeOptions {
	ro.Opts = append(ro.Opts, func(args *RangeArgs) error {
		args.Min = &min

		return nil
	})

	return ro
}

// SetMax sets the range index maximum value.
// Beta: The Range algorithm is experimental only. It is not intended for public use. It is subject to breaking changes.
func (ro *RangeOptions) SetMax(max bson.RawValue) *RangeOptions {
	ro.Opts = append(ro.Opts, func(args *RangeArgs) error {
		args.Max = &max

		return nil
	})

	return ro
}

// SetSparsity sets the range index sparsity.
// Beta: The Range algorithm is experimental only. It is not intended for public use. It is subject to breaking changes.
func (ro *RangeOptions) SetSparsity(sparsity int64) *RangeOptions {
	ro.Opts = append(ro.Opts, func(args *RangeArgs) error {
		args.Sparsity = sparsity

		return nil
	})

	return ro
}

// SetPrecision sets the range index precision.
// Beta: The Range algorithm is experimental only. It is not intended for public use. It is subject to breaking changes.
func (ro *RangeOptions) SetPrecision(precision int32) *RangeOptions {
	ro.Opts = append(ro.Opts, func(args *RangeArgs) error {
		args.Precision = &precision

		return nil
	})

	return ro
}
