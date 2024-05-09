// Copyright (C) MongoDB, Inc. 2024-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongoutil

import (
	"testing"

	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/internal/ptrutil"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestNewArgumentsFromOptions(t *testing.T) {
	t.Parallel()

	// For simplicity, we just chose one options type to test on. This should be
	// WLOG since (1) a user cannot merge mixed options, and (2) exported data in
	// the options package cannot be backwards breaking. If
	// options-package-specific  functionality needs to be tested, it should be
	// done in a separate test.
	clientTests := []struct {
		name string
		opts []MongoOptions[options.ClientArgs]
		want options.ClientArgs
	}{
		{
			name: "nil options",
			opts: nil,
			want: options.ClientArgs{},
		},
		{
			name: "no options",
			opts: []MongoOptions[options.ClientArgs]{},
			want: options.ClientArgs{},
		},
		{
			name: "one option",
			opts: []MongoOptions[options.ClientArgs]{
				options.Client().SetAppName("testApp"),
			},
			want: options.ClientArgs{AppName: ptrutil.Ptr[string]("testApp")},
		},
		{
			name: "one nil option",
			opts: []MongoOptions[options.ClientArgs]{nil},
			want: options.ClientArgs{},
		},
		{
			name: "many same options",
			opts: []MongoOptions[options.ClientArgs]{
				options.Client().SetAppName("testApp"),
				options.Client().SetAppName("testApp"),
			},
			want: options.ClientArgs{AppName: ptrutil.Ptr[string]("testApp")},
		},
		{
			name: "many different options (last one wins)",
			opts: []MongoOptions[options.ClientArgs]{
				options.Client().SetAppName("testApp1"),
				options.Client().SetAppName("testApp2"),
			},
			want: options.ClientArgs{AppName: ptrutil.Ptr[string]("testApp2")},
		},
		{
			name: "many nil options",
			opts: []MongoOptions[options.ClientArgs]{nil, nil},
			want: options.ClientArgs{},
		},
		{
			name: "many options where last is nil (non-nil wins)",
			opts: []MongoOptions[options.ClientArgs]{
				options.Client().SetAppName("testApp"),
				nil,
			},
			want: options.ClientArgs{AppName: ptrutil.Ptr[string]("testApp")},
		},
		{
			name: "many nil options where first is nil (non-nil wins)",
			opts: []MongoOptions[options.ClientArgs]{
				nil,
				options.Client().SetAppName("testApp"),
			},
			want: options.ClientArgs{AppName: ptrutil.Ptr[string]("testApp")},
		},
		{
			name: "many nil options where middle is non-nil (non-nil wins)",
			opts: []MongoOptions[options.ClientArgs]{
				nil,
				options.Client().SetAppName("testApp"),
				nil,
			},
			want: options.ClientArgs{AppName: ptrutil.Ptr[string]("testApp")},
		},
	}

	for _, test := range clientTests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewArgsFromOptions[options.ClientArgs](test.opts...)
			assert.NoError(t, err)

			// WLOG it should be enough to test a small subset of arguments.
			assert.Equal(t, test.want.AppName, got.AppName)
		})
	}
}

func TestNewOptionsFromArgs(t *testing.T) {
	t.Parallel()

	// For simplicity, we just chose one options type to test on. This should be
	// WLOG since (1) a user cannot merge mixed options, and (2) exported data in
	// the options package cannot be backwards breaking. If
	// options-package-specific  functionality needs to be tested, it should be
	// done in a separate test.
	clientTests := []struct {
		name string
		args *options.ClientArgs
		clbk func(*options.ClientArgs) error
		want options.ClientArgs
	}{
		{
			name: "nil args",
			args: nil,
			clbk: nil,
			want: options.ClientArgs{},
		},
		{
			name: "no args",
			args: &options.ClientArgs{},
			clbk: nil,
			want: options.ClientArgs{},
		},
		{
			name: "no callback",
			args: &options.ClientArgs{AppName: ptrutil.Ptr[string]("testApp")},
			want: options.ClientArgs{AppName: ptrutil.Ptr[string]("testApp")},
		},
	}

	for _, test := range clientTests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			opts := NewOptionsFromArgs[options.ClientArgs](test.args)

			got, err := NewArgsFromOptions[options.ClientArgs](opts)
			assert.NoError(t, err)

			// WLOG it should be enough to test a small subset of arguments.
			assert.Equal(t, test.want.AppName, got.AppName)
		})
	}
}
