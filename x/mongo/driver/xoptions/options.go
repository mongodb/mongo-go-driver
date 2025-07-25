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
//
// Deprecated: This function is for internal use only. It may be changed or removed in any release.
func SetInternalClientOptions(opts *options.ClientOptions, key string, option any) error {
	typeErrFunc := func(t string) error {
		return fmt.Errorf("unexpected type for %s: %T is not %s", key, option, t)
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
//
// Deprecated: This function is for internal use only. It may be changed or removed in any release.
func SetInternalAggregateOptions(a *options.AggregateOptionsBuilder, key string, option any) error {
	typeErrFunc := func(t string) error {
		return fmt.Errorf("unexpected type for %s: %T is not %s", key, option, t)
	}
	switch key {
	case "rawData":
		b, ok := option.(bool)
		if !ok {
			return typeErrFunc("bool")
		}
		a.Opts = append(a.Opts, func(opts *options.AggregateOptions) error {
			opts.CustomOptions = optionsutil.WithValue(opts.CustomOptions, key, b)
			return nil
		})
	default:
		return fmt.Errorf("unsupported option: %s", key)
	}
	return nil
}

// SetInternalBulkWriteOptions sets internal options for BulkWriteOptions.
//
// Deprecated: This function is for internal use only. It may be changed or removed in any release.
func SetInternalBulkWriteOptions(a *options.BulkWriteOptionsBuilder, key string, option any) error {
	typeErrFunc := func(t string) error {
		return fmt.Errorf("unexpected type for %s: %T is not %s", key, option, t)
	}
	switch key {
	case "rawData":
		b, ok := option.(bool)
		if !ok {
			return typeErrFunc("bool")
		}
		a.Opts = append(a.Opts, func(opts *options.BulkWriteOptions) error {
			opts.CustomOptions = optionsutil.WithValue(opts.CustomOptions, key, b)
			return nil
		})
	default:
		return fmt.Errorf("unsupported option: %s", key)
	}
	return nil
}

// SetInternalCountOptions sets internal options for CountOptions.
//
// Deprecated: This function is for internal use only. It may be changed or removed in any release.
func SetInternalCountOptions(a *options.CountOptionsBuilder, key string, option any) error {
	typeErrFunc := func(t string) error {
		return fmt.Errorf("unexpected type for %s: %T is not %s", key, option, t)
	}
	switch key {
	case "rawData":
		b, ok := option.(bool)
		if !ok {
			return typeErrFunc("bool")
		}
		a.Opts = append(a.Opts, func(opts *options.CountOptions) error {
			opts.CustomOptions = optionsutil.WithValue(opts.CustomOptions, key, b)
			return nil
		})
	default:
		return fmt.Errorf("unsupported option: %s", key)
	}
	return nil
}

// SetInternalDeleteOneOptions sets internal options for DeleteOneOptions.
//
// Deprecated: This function is for internal use only. It may be changed or removed in any release.
func SetInternalDeleteOneOptions(a *options.DeleteOneOptionsBuilder, key string, option any) error {
	typeErrFunc := func(t string) error {
		return fmt.Errorf("unexpected type for %s: %T is not %s", key, option, t)
	}
	switch key {
	case "rawData":
		b, ok := option.(bool)
		if !ok {
			return typeErrFunc("bool")
		}
		a.Opts = append(a.Opts, func(opts *options.DeleteOneOptions) error {
			opts.CustomOptions = optionsutil.WithValue(opts.CustomOptions, key, b)
			return nil
		})
	default:
		return fmt.Errorf("unsupported option: %s", key)
	}
	return nil
}

// SetInternalDeleteManyOptions sets internal options for DeleteManyOptions.
//
// Deprecated: This function is for internal use only. It may be changed or removed in any release.
func SetInternalDeleteManyOptions(a *options.DeleteManyOptionsBuilder, key string, option any) error {
	typeErrFunc := func(t string) error {
		return fmt.Errorf("unexpected type for %s: %T is not %s", key, option, t)
	}
	switch key {
	case "rawData":
		b, ok := option.(bool)
		if !ok {
			return typeErrFunc("bool")
		}
		a.Opts = append(a.Opts, func(opts *options.DeleteManyOptions) error {
			opts.CustomOptions = optionsutil.WithValue(opts.CustomOptions, key, b)
			return nil
		})
	default:
		return fmt.Errorf("unsupported option: %s", key)
	}
	return nil
}

// SetInternalDistinctOptions sets internal options for DistinctOptions.
//
// Deprecated: This function is for internal use only. It may be changed or removed in any release.
func SetInternalDistinctOptions(a *options.DistinctOptionsBuilder, key string, option any) error {
	typeErrFunc := func(t string) error {
		return fmt.Errorf("unexpected type for %s: %T is not %s", key, option, t)
	}
	switch key {
	case "rawData":
		b, ok := option.(bool)
		if !ok {
			return typeErrFunc("bool")
		}
		a.Opts = append(a.Opts, func(opts *options.DistinctOptions) error {
			opts.CustomOptions = optionsutil.WithValue(opts.CustomOptions, key, b)
			return nil
		})
	default:
		return fmt.Errorf("unsupported option: %s", key)
	}
	return nil
}

// SetInternalEstimatedDocumentCountOptions sets internal options for EstimatedDocumentCountOptions.
//
// Deprecated: This function is for internal use only. It may be changed or removed in any release.
func SetInternalEstimatedDocumentCountOptions(a *options.EstimatedDocumentCountOptionsBuilder, key string, option any) error {
	typeErrFunc := func(t string) error {
		return fmt.Errorf("unexpected type for %s: %T is not %s", key, option, t)
	}
	switch key {
	case "rawData":
		b, ok := option.(bool)
		if !ok {
			return typeErrFunc("bool")
		}
		a.Opts = append(a.Opts, func(opts *options.EstimatedDocumentCountOptions) error {
			opts.CustomOptions = optionsutil.WithValue(opts.CustomOptions, key, b)
			return nil
		})
	default:
		return fmt.Errorf("unsupported option: %s", key)
	}
	return nil
}

// SetInternalFindOptions sets internal options for FindOptions.
//
// Deprecated: This function is for internal use only. It may be changed or removed in any release.
func SetInternalFindOptions(a *options.FindOptionsBuilder, key string, option any) error {
	typeErrFunc := func(t string) error {
		return fmt.Errorf("unexpected type for %s: %T is not %s", key, option, t)
	}
	switch key {
	case "rawData":
		b, ok := option.(bool)
		if !ok {
			return typeErrFunc("bool")
		}
		a.Opts = append(a.Opts, func(opts *options.FindOptions) error {
			opts.CustomOptions = optionsutil.WithValue(opts.CustomOptions, key, b)
			return nil
		})
	default:
		return fmt.Errorf("unsupported option: %s", key)
	}
	return nil
}

// SetInternalFindOneOptions sets internal options for FindOneOptions.
//
// Deprecated: This function is for internal use only. It may be changed or removed in any release.
func SetInternalFindOneOptions(a *options.FindOneOptionsBuilder, key string, option any) error {
	typeErrFunc := func(t string) error {
		return fmt.Errorf("unexpected type for %s: %T is not %s", key, option, t)
	}
	switch key {
	case "rawData":
		b, ok := option.(bool)
		if !ok {
			return typeErrFunc("bool")
		}
		a.Opts = append(a.Opts, func(opts *options.FindOneOptions) error {
			opts.CustomOptions = optionsutil.WithValue(opts.CustomOptions, key, b)
			return nil
		})
	default:
		return fmt.Errorf("unsupported option: %s", key)
	}
	return nil
}

// SetInternalFindOneAndReplaceOptions sets internal options for FindOneAndReplaceOptions.
//
// Deprecated: This function is for internal use only. It may be changed or removed in any release.
func SetInternalFindOneAndReplaceOptions(a *options.FindOneAndReplaceOptionsBuilder, key string, option any) error {
	typeErrFunc := func(t string) error {
		return fmt.Errorf("unexpected type for %s: %T is not %s", key, option, t)
	}
	switch key {
	case "rawData":
		b, ok := option.(bool)
		if !ok {
			return typeErrFunc("bool")
		}
		a.Opts = append(a.Opts, func(opts *options.FindOneAndReplaceOptions) error {
			opts.CustomOptions = optionsutil.WithValue(opts.CustomOptions, key, b)
			return nil
		})
	default:
		return fmt.Errorf("unsupported option: %s", key)
	}
	return nil
}

// SetInternalFindOneAndUpdateOptions sets internal options for FindOneAndUpdateOptions.
//
// Deprecated: This function is for internal use only. It may be changed or removed in any release.
func SetInternalFindOneAndUpdateOptions(a *options.FindOneAndUpdateOptionsBuilder, key string, option any) error {
	typeErrFunc := func(t string) error {
		return fmt.Errorf("unexpected type for %s: %T is not %s", key, option, t)
	}
	switch key {
	case "rawData":
		b, ok := option.(bool)
		if !ok {
			return typeErrFunc("bool")
		}
		a.Opts = append(a.Opts, func(opts *options.FindOneAndUpdateOptions) error {
			opts.CustomOptions = optionsutil.WithValue(opts.CustomOptions, key, b)
			return nil
		})
	default:
		return fmt.Errorf("unsupported option: %s", key)
	}
	return nil
}

// SetInternalFindOneAndDeleteOptions sets internal options for FindOneAndDeleteOptions.
//
// Deprecated: This function is for internal use only. It may be changed or removed in any release.
func SetInternalFindOneAndDeleteOptions(a *options.FindOneAndDeleteOptionsBuilder, key string, option any) error {
	typeErrFunc := func(t string) error {
		return fmt.Errorf("unexpected type for %s: %T is not %s", key, option, t)
	}
	switch key {
	case "rawData":
		b, ok := option.(bool)
		if !ok {
			return typeErrFunc("bool")
		}
		a.Opts = append(a.Opts, func(opts *options.FindOneAndDeleteOptions) error {
			opts.CustomOptions = optionsutil.WithValue(opts.CustomOptions, key, b)
			return nil
		})
	default:
		return fmt.Errorf("unsupported option: %s", key)
	}
	return nil
}

// SetInternalInsertManyOptions sets internal options for InsertManyOptions.
//
// Deprecated: This function is for internal use only. It may be changed or removed in any release.
func SetInternalInsertManyOptions(a *options.InsertManyOptionsBuilder, key string, option any) error {
	typeErrFunc := func(t string) error {
		return fmt.Errorf("unexpected type for %s: %T is not %s", key, option, t)
	}
	switch key {
	case "rawData":
		b, ok := option.(bool)
		if !ok {
			return typeErrFunc("bool")
		}
		a.Opts = append(a.Opts, func(opts *options.InsertManyOptions) error {
			opts.CustomOptions = optionsutil.WithValue(opts.CustomOptions, key, b)
			return nil
		})
	default:
		return fmt.Errorf("unsupported option: %s", key)
	}
	return nil
}

// SetInternalInsertOneOptions sets internal options for InsertOneOptions.
//
// Deprecated: This function is for internal use only. It may be changed or removed in any release.
func SetInternalInsertOneOptions(a *options.InsertOneOptionsBuilder, key string, option any) error {
	typeErrFunc := func(t string) error {
		return fmt.Errorf("unexpected type for %s: %T is not %s", key, option, t)
	}
	switch key {
	case "rawData":
		b, ok := option.(bool)
		if !ok {
			return typeErrFunc("bool")
		}
		a.Opts = append(a.Opts, func(opts *options.InsertOneOptions) error {
			opts.CustomOptions = optionsutil.WithValue(opts.CustomOptions, key, b)
			return nil
		})
	default:
		return fmt.Errorf("unsupported option: %s", key)
	}
	return nil
}

// SetInternalReplaceOptions sets internal options for ReplaceOptions.
//
// Deprecated: This function is for internal use only. It may be changed or removed in any release.
func SetInternalReplaceOptions(a *options.ReplaceOptionsBuilder, key string, option any) error {
	typeErrFunc := func(t string) error {
		return fmt.Errorf("unexpected type for %s: %T is not %s", key, option, t)
	}
	switch key {
	case "rawData":
		b, ok := option.(bool)
		if !ok {
			return typeErrFunc("bool")
		}
		a.Opts = append(a.Opts, func(opts *options.ReplaceOptions) error {
			opts.CustomOptions = optionsutil.WithValue(opts.CustomOptions, key, b)
			return nil
		})
	default:
		return fmt.Errorf("unsupported option: %s", key)
	}
	return nil
}

// SetInternalUpdateManyOptions sets internal options for UpdateManyOptions.
//
// Deprecated: This function is for internal use only. It may be changed or removed in any release.
func SetInternalUpdateManyOptions(a *options.UpdateManyOptionsBuilder, key string, option any) error {
	typeErrFunc := func(t string) error {
		return fmt.Errorf("unexpected type for %s: %T is not %s", key, option, t)
	}
	switch key {
	case "rawData":
		b, ok := option.(bool)
		if !ok {
			return typeErrFunc("bool")
		}
		a.Opts = append(a.Opts, func(opts *options.UpdateManyOptions) error {
			opts.CustomOptions = optionsutil.WithValue(opts.CustomOptions, key, b)
			return nil
		})
	default:
		return fmt.Errorf("unsupported option: %s", key)
	}
	return nil
}

// SetInternalUpdateOneOptions sets internal options for UpdateOneOptions.
//
// Deprecated: This function is for internal use only. It may be changed or removed in any release.
func SetInternalUpdateOneOptions(a *options.UpdateOneOptionsBuilder, key string, option any) error {
	typeErrFunc := func(t string) error {
		return fmt.Errorf("unexpected type for %s: %T is not %s", key, option, t)
	}
	switch key {
	case "rawData":
		b, ok := option.(bool)
		if !ok {
			return typeErrFunc("bool")
		}
		a.Opts = append(a.Opts, func(opts *options.UpdateOneOptions) error {
			opts.CustomOptions = optionsutil.WithValue(opts.CustomOptions, key, b)
			return nil
		})
	default:
		return fmt.Errorf("unsupported option: %s", key)
	}
	return nil
}
