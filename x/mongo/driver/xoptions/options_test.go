// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package xoptions

import (
	"fmt"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/mongoutil"
	"go.mongodb.org/mongo-driver/v2/internal/optionsutil"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/drivertest"
)

func TestSetInternalClientOptions(t *testing.T) {
	t.Parallel()

	cases := []struct {
		key   string
		value any
	}{
		{
			key:   "authenticateToAnything",
			value: true,
		},
	}
	for _, tc := range cases {
		tc := tc

		t.Run(fmt.Sprintf("set %s", tc.key), func(t *testing.T) {
			t.Parallel()

			opts := options.Client()
			err := SetInternalClientOptions(opts, tc.key, tc.value)
			require.NoError(t, err, "error setting %s: %v", tc.key, err)
			v := optionsutil.Value(opts.Custom, tc.key)
			require.Equal(t, tc.value, v, "expected %v, got %v", tc.value, v)
		})
	}

	t.Run("set crypt", func(t *testing.T) {
		t.Parallel()

		c := driver.NewCrypt(&driver.CryptOptions{})
		opts := options.Client()
		err := SetInternalClientOptions(opts, "crypt", c)
		require.NoError(t, err, "error setting crypt: %v", err)
		require.Equal(t, c, opts.Crypt, "expected %v, got %v", c, opts.Crypt)
	})

	t.Run("set crypt - wrong type", func(t *testing.T) {
		t.Parallel()

		opts := options.Client()
		err := SetInternalClientOptions(opts, "crypt", &drivertest.MockDeployment{})
		require.EqualError(t, err, "unexpected type for \"crypt\": *drivertest.MockDeployment is not driver.Crypt")
	})

	t.Run("set deployment", func(t *testing.T) {
		t.Parallel()

		d := &drivertest.MockDeployment{}
		opts := options.Client()
		err := SetInternalClientOptions(opts, "deployment", d)
		require.NoError(t, err, "error setting deployment: %v", err)
		require.Equal(t, d, opts.Deployment, "expected %v, got %v", d, opts.Deployment)
	})

	t.Run("set deployment - wrong type", func(t *testing.T) {
		t.Parallel()

		opts := options.Client()
		err := SetInternalClientOptions(opts, "deployment", driver.NewCrypt(&driver.CryptOptions{}))
		require.EqualError(t, err, "unexpected type for \"deployment\": *driver.crypt is not driver.Deployment")
	})

	t.Run("set unsupported option", func(t *testing.T) {
		t.Parallel()

		opts := options.Client()
		err := SetInternalClientOptions(opts, "unsupported", "unsupported")
		require.EqualError(t, err, "unsupported option: \"unsupported\"")
	})
}

// TestSetInternalAddCommandFields verifies that each collection- and
// index-level options setter that supports the "addCommandFields" key stores
// the provided bson.D in the resulting Internal options.
func TestSetInternalAddCommandFields(t *testing.T) {
	t.Parallel()

	want := bson.D{{Key: "collectionUUID", Value: "00000000-0000-0000-0000-000000000000"}}

	// Each case sets "addCommandFields" via a specific setter and returns the
	// Internal options that the setter populated.
	cases := []struct {
		name string
		set  func(t *testing.T) optionsutil.Options
	}{
		{"BulkWriteOptions", func(t *testing.T) optionsutil.Options {
			o := options.BulkWrite()
			require.NoError(t, SetInternalBulkWriteOptions(o, "addCommandFields", want))
			args, err := mongoutil.NewOptions[options.BulkWriteOptions](o)
			require.NoError(t, err)
			return args.Internal
		}},
		{"CountOptions", func(t *testing.T) optionsutil.Options {
			o := options.Count()
			require.NoError(t, SetInternalCountOptions(o, "addCommandFields", want))
			args, err := mongoutil.NewOptions[options.CountOptions](o)
			require.NoError(t, err)
			return args.Internal
		}},
		{"CreateIndexesOptions", func(t *testing.T) optionsutil.Options {
			o := options.CreateIndexes()
			require.NoError(t, SetInternalCreateIndexesOptions(o, "addCommandFields", want))
			args, err := mongoutil.NewOptions[options.CreateIndexesOptions](o)
			require.NoError(t, err)
			return args.Internal
		}},
		{"DeleteOneOptions", func(t *testing.T) optionsutil.Options {
			o := options.DeleteOne()
			require.NoError(t, SetInternalDeleteOneOptions(o, "addCommandFields", want))
			args, err := mongoutil.NewOptions[options.DeleteOneOptions](o)
			require.NoError(t, err)
			return args.Internal
		}},
		{"DeleteManyOptions", func(t *testing.T) optionsutil.Options {
			o := options.DeleteMany()
			require.NoError(t, SetInternalDeleteManyOptions(o, "addCommandFields", want))
			args, err := mongoutil.NewOptions[options.DeleteManyOptions](o)
			require.NoError(t, err)
			return args.Internal
		}},
		{"DistinctOptions", func(t *testing.T) optionsutil.Options {
			o := options.Distinct()
			require.NoError(t, SetInternalDistinctOptions(o, "addCommandFields", want))
			args, err := mongoutil.NewOptions[options.DistinctOptions](o)
			require.NoError(t, err)
			return args.Internal
		}},
		{"DropCollectionOptions", func(t *testing.T) optionsutil.Options {
			o := options.DropCollection()
			require.NoError(t, SetInternalDropCollectionOptions(o, "addCommandFields", want))
			args, err := mongoutil.NewOptions[options.DropCollectionOptions](o)
			require.NoError(t, err)
			return args.Internal
		}},
		{"DropIndexesOptions", func(t *testing.T) optionsutil.Options {
			o := options.DropIndexes()
			require.NoError(t, SetInternalDropIndexesOptions(o, "addCommandFields", want))
			args, err := mongoutil.NewOptions[options.DropIndexesOptions](o)
			require.NoError(t, err)
			return args.Internal
		}},
		{"EstimatedDocumentCountOptions", func(t *testing.T) optionsutil.Options {
			o := options.EstimatedDocumentCount()
			require.NoError(t, SetInternalEstimatedDocumentCountOptions(o, "addCommandFields", want))
			args, err := mongoutil.NewOptions[options.EstimatedDocumentCountOptions](o)
			require.NoError(t, err)
			return args.Internal
		}},
		{"FindOptions", func(t *testing.T) optionsutil.Options {
			o := options.Find()
			require.NoError(t, SetInternalFindOptions(o, "addCommandFields", want))
			args, err := mongoutil.NewOptions[options.FindOptions](o)
			require.NoError(t, err)
			return args.Internal
		}},
		{"FindOneOptions", func(t *testing.T) optionsutil.Options {
			o := options.FindOne()
			require.NoError(t, SetInternalFindOneOptions(o, "addCommandFields", want))
			args, err := mongoutil.NewOptions[options.FindOneOptions](o)
			require.NoError(t, err)
			return args.Internal
		}},
		{"FindOneAndReplaceOptions", func(t *testing.T) optionsutil.Options {
			o := options.FindOneAndReplace()
			require.NoError(t, SetInternalFindOneAndReplaceOptions(o, "addCommandFields", want))
			args, err := mongoutil.NewOptions[options.FindOneAndReplaceOptions](o)
			require.NoError(t, err)
			return args.Internal
		}},
		{"FindOneAndUpdateOptions", func(t *testing.T) optionsutil.Options {
			o := options.FindOneAndUpdate()
			require.NoError(t, SetInternalFindOneAndUpdateOptions(o, "addCommandFields", want))
			args, err := mongoutil.NewOptions[options.FindOneAndUpdateOptions](o)
			require.NoError(t, err)
			return args.Internal
		}},
		{"InsertOneOptions", func(t *testing.T) optionsutil.Options {
			o := options.InsertOne()
			require.NoError(t, SetInternalInsertOneOptions(o, "addCommandFields", want))
			args, err := mongoutil.NewOptions[options.InsertOneOptions](o)
			require.NoError(t, err)
			return args.Internal
		}},
		{"InsertManyOptions", func(t *testing.T) optionsutil.Options {
			o := options.InsertMany()
			require.NoError(t, SetInternalInsertManyOptions(o, "addCommandFields", want))
			args, err := mongoutil.NewOptions[options.InsertManyOptions](o)
			require.NoError(t, err)
			return args.Internal
		}},
		{"ListIndexesOptions", func(t *testing.T) optionsutil.Options {
			o := options.ListIndexes()
			require.NoError(t, SetInternalListIndexesOptions(o, "addCommandFields", want))
			args, err := mongoutil.NewOptions[options.ListIndexesOptions](o)
			require.NoError(t, err)
			return args.Internal
		}},
		{"ReplaceOptions", func(t *testing.T) optionsutil.Options {
			o := options.Replace()
			require.NoError(t, SetInternalReplaceOptions(o, "addCommandFields", want))
			args, err := mongoutil.NewOptions[options.ReplaceOptions](o)
			require.NoError(t, err)
			return args.Internal
		}},
		{"UpdateOneOptions", func(t *testing.T) optionsutil.Options {
			o := options.UpdateOne()
			require.NoError(t, SetInternalUpdateOneOptions(o, "addCommandFields", want))
			args, err := mongoutil.NewOptions[options.UpdateOneOptions](o)
			require.NoError(t, err)
			return args.Internal
		}},
		{"UpdateManyOptions", func(t *testing.T) optionsutil.Options {
			o := options.UpdateMany()
			require.NoError(t, SetInternalUpdateManyOptions(o, "addCommandFields", want))
			args, err := mongoutil.NewOptions[options.UpdateManyOptions](o)
			require.NoError(t, err)
			return args.Internal
		}},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			internal := tc.set(t)
			got, ok := optionsutil.Value(internal, "addCommandFields").(bson.D)
			require.True(t, ok, "expected addCommandFields to be a bson.D")
			require.Equal(t, want, got)
		})
	}
}
