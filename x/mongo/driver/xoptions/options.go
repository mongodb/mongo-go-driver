// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package xoptions

import (
	"fmt"

	"go.mongodb.org/mongo-driver/v2/internal/optionsutil"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
)

// SetInternalClientOptions sets internal options for ClientOptions.
func SetInternalClientOptions(opts *options.ClientOptions, key string, option any) error {
	typeErrFunc := func(t string) error {
		return fmt.Errorf("unexpected type for %q: %T is not %s", key, option, t)
	}
	switch key {
	case "crypt":
		c, ok := option.(driver.Crypt)
		if !ok {
			return typeErrFunc("driver.Crypt")
		}
		opts.Crypt = c
	case "deployment":
		d, ok := option.(driver.Deployment)
		if !ok {
			return typeErrFunc("driver.Deployment")
		}
		opts.Deployment = d
	case "authenticateToAnything":
		b, ok := option.(bool)
		if !ok {
			return typeErrFunc("bool")
		}
		opts.Custom = optionsutil.WithValue(opts.Custom, key, b)
	default:
		return fmt.Errorf("unsupported option: %s", key)
	}
	return nil
}

// SetInternalAggregateOptions sets internal options for AggregateOptions.
func SetInternalAggregateOptions(a *options.AggregateOptionsBuilder, key string, option any) error {
	typeErrFunc := func(t string) error {
		return fmt.Errorf("unexpected type for %q: %T is not %s", key, option, t)
	}
	switch key {
	case "rawData":
		b, ok := option.(bool)
		if !ok {
			return typeErrFunc("bool")
		}
		a.Opts = append(a.Opts, func(opts *options.AggregateOptions) error {
			opts.Internal = optionsutil.WithValue(opts.Internal, key, b)
			return nil
		})
	default:
		return fmt.Errorf("unsupported option: %s", key)
	}
	return nil
}

// SetInternalCountOptions sets internal options for CountOptions.
func SetInternalCountOptions(a *options.CountOptionsBuilder, key string, option any) error {
	typeErrFunc := func(t string) error {
		return fmt.Errorf("unexpected type for %q: %T is not %s", key, option, t)
	}
	switch key {
	case "rawData":
		b, ok := option.(bool)
		if !ok {
			return typeErrFunc("bool")
		}
		a.Opts = append(a.Opts, func(opts *options.CountOptions) error {
			opts.Internal = optionsutil.WithValue(opts.Internal, key, b)
			return nil
		})
	default:
		return fmt.Errorf("unsupported option: %s", key)
	}
	return nil
}

// SetInternalDeleteOneOptions sets internal options for DeleteOneOptions.
func SetInternalDeleteOneOptions(a *options.DeleteOneOptionsBuilder, key string, option any) error {
	typeErrFunc := func(t string) error {
		return fmt.Errorf("unexpected type for %q: %T is not %s", key, option, t)
	}
	switch key {
	case "rawData":
		b, ok := option.(bool)
		if !ok {
			return typeErrFunc("bool")
		}
		a.Opts = append(a.Opts, func(opts *options.DeleteOneOptions) error {
			opts.Internal = optionsutil.WithValue(opts.Internal, key, b)
			return nil
		})
	default:
		return fmt.Errorf("unsupported option: %s", key)
	}
	return nil
}

// SetInternalDeleteManyOptions sets internal options for DeleteManyOptions.
func SetInternalDeleteManyOptions(a *options.DeleteManyOptionsBuilder, key string, option any) error {
	typeErrFunc := func(t string) error {
		return fmt.Errorf("unexpected type for %q: %T is not %s", key, option, t)
	}
	switch key {
	case "rawData":
		b, ok := option.(bool)
		if !ok {
			return typeErrFunc("bool")
		}
		a.Opts = append(a.Opts, func(opts *options.DeleteManyOptions) error {
			opts.Internal = optionsutil.WithValue(opts.Internal, key, b)
			return nil
		})
	default:
		return fmt.Errorf("unsupported option: %s", key)
	}
	return nil
}

// SetInternalDistinctOptions sets internal options for DistinctOptions.
func SetInternalDistinctOptions(a *options.DistinctOptionsBuilder, key string, option any) error {
	typeErrFunc := func(t string) error {
		return fmt.Errorf("unexpected type for %q: %T is not %s", key, option, t)
	}
	switch key {
	case "rawData":
		b, ok := option.(bool)
		if !ok {
			return typeErrFunc("bool")
		}
		a.Opts = append(a.Opts, func(opts *options.DistinctOptions) error {
			opts.Internal = optionsutil.WithValue(opts.Internal, key, b)
			return nil
		})
	default:
		return fmt.Errorf("unsupported option: %s", key)
	}
	return nil
}

// SetInternalEstimatedDocumentCountOptions sets internal options for EstimatedDocumentCountOptions.
func SetInternalEstimatedDocumentCountOptions(a *options.EstimatedDocumentCountOptionsBuilder, key string, option any) error {
	typeErrFunc := func(t string) error {
		return fmt.Errorf("unexpected type for %q: %T is not %s", key, option, t)
	}
	switch key {
	case "rawData":
		b, ok := option.(bool)
		if !ok {
			return typeErrFunc("bool")
		}
		a.Opts = append(a.Opts, func(opts *options.EstimatedDocumentCountOptions) error {
			opts.Internal = optionsutil.WithValue(opts.Internal, key, b)
			return nil
		})
	default:
		return fmt.Errorf("unsupported option: %s", key)
	}
	return nil
}

// SetInternalInsertManyOptions sets internal options for InsertManyOptions.
func SetInternalInsertManyOptions(a *options.InsertManyOptionsBuilder, key string, option any) error {
	typeErrFunc := func(t string) error {
		return fmt.Errorf("unexpected type for %q: %T is not %s", key, option, t)
	}
	switch key {
	case "rawData":
		b, ok := option.(bool)
		if !ok {
			return typeErrFunc("bool")
		}
		a.Opts = append(a.Opts, func(opts *options.InsertManyOptions) error {
			opts.Internal = optionsutil.WithValue(opts.Internal, key, b)
			return nil
		})
	default:
		return fmt.Errorf("unsupported option: %s", key)
	}
	return nil
}

// SetInternalInsertOneOptions sets internal options for InsertOneOptions.
func SetInternalInsertOneOptions(a *options.InsertOneOptionsBuilder, key string, option any) error {
	typeErrFunc := func(t string) error {
		return fmt.Errorf("unexpected type for %q: %T is not %s", key, option, t)
	}
	switch key {
	case "rawData":
		b, ok := option.(bool)
		if !ok {
			return typeErrFunc("bool")
		}
		a.Opts = append(a.Opts, func(opts *options.InsertOneOptions) error {
			opts.Internal = optionsutil.WithValue(opts.Internal, key, b)
			return nil
		})
	default:
		return fmt.Errorf("unsupported option: %s", key)
	}
	return nil
}
