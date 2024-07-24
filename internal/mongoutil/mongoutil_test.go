// Copyright (C) MongoDB, Inc. 2024-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongoutil

import (
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/internal/ptrutil"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestNewOptions(t *testing.T) {
	t.Parallel()

	// For simplicity, we just chose one options type to test on. This should be
	// WLOG since (1) a user cannot merge mixed options, and (2) exported data in
	// the options package cannot be backwards breaking. If
	// options-package-specific  functionality needs to be tested, it should be
	// done in a separate test.
	clientTests := []struct {
		name string
		opts []options.Lister[options.ClientOptions]
		want options.ClientOptions
	}{
		{
			name: "nil options",
			opts: nil,
			want: options.ClientOptions{},
		},
		{
			name: "no options",
			opts: []options.Lister[options.ClientOptions]{},
			want: options.ClientOptions{},
		},
		{
			name: "one option",
			opts: []options.Lister[options.ClientOptions]{
				options.Client().SetAppName("testApp"),
			},
			want: options.ClientOptions{AppName: ptrutil.Ptr[string]("testApp")},
		},
		{
			name: "one nil option",
			opts: []options.Lister[options.ClientOptions]{nil},
			want: options.ClientOptions{},
		},
		{
			name: "many same options",
			opts: []options.Lister[options.ClientOptions]{
				options.Client().SetAppName("testApp"),
				options.Client().SetAppName("testApp"),
			},
			want: options.ClientOptions{AppName: ptrutil.Ptr[string]("testApp")},
		},
		{
			name: "many different options (last one wins)",
			opts: []options.Lister[options.ClientOptions]{
				options.Client().SetAppName("testApp1"),
				options.Client().SetAppName("testApp2"),
			},
			want: options.ClientOptions{AppName: ptrutil.Ptr[string]("testApp2")},
		},
		{
			name: "many nil options",
			opts: []options.Lister[options.ClientOptions]{nil, nil},
			want: options.ClientOptions{},
		},
		{
			name: "many options where last is nil (non-nil wins)",
			opts: []options.Lister[options.ClientOptions]{
				options.Client().SetAppName("testApp"),
				nil,
			},
			want: options.ClientOptions{AppName: ptrutil.Ptr[string]("testApp")},
		},
		{
			name: "many nil options where first is nil (non-nil wins)",
			opts: []options.Lister[options.ClientOptions]{
				nil,
				options.Client().SetAppName("testApp"),
			},
			want: options.ClientOptions{AppName: ptrutil.Ptr[string]("testApp")},
		},
		{
			name: "many nil options where middle is non-nil (non-nil wins)",
			opts: []options.Lister[options.ClientOptions]{
				nil,
				options.Client().SetAppName("testApp"),
				nil,
			},
			want: options.ClientOptions{AppName: ptrutil.Ptr[string]("testApp")},
		},
	}

	for _, test := range clientTests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewOptions[options.ClientOptions](test.opts...)
			assert.NoError(t, err)

			// WLOG it should be enough to test a small subset of arguments.
			assert.Equal(t, test.want.AppName, got.AppName)
		})
	}
}

func TestNewOptionsLister(t *testing.T) {
	t.Parallel()

	// For simplicity, we just chose one options type to test on. This should be
	// WLOG since (1) a user cannot merge mixed options, and (2) exported data in
	// the options package cannot be backwards breaking. If
	// options-package-specific  functionality needs to be tested, it should be
	// done in a separate test.
	clientTests := []struct {
		name string
		args *options.ClientOptions
		want options.ClientOptions
	}{
		{
			name: "nil args",
			args: nil,
			want: options.ClientOptions{},
		},
		{
			name: "no args",
			args: &options.ClientOptions{},
			want: options.ClientOptions{},
		},
		{
			name: "args",
			args: &options.ClientOptions{AppName: ptrutil.Ptr[string]("testApp")},
			want: options.ClientOptions{AppName: ptrutil.Ptr[string]("testApp")},
		},
	}

	for _, test := range clientTests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			opts := NewOptionsLister(test.args, nil)

			got, err := NewOptions[options.ClientOptions](opts)
			assert.NoError(t, err)

			// WLOG it should be enough to test a small subset of arguments.
			assert.Equal(t, test.want.AppName, got.AppName)
		})
	}
}

func BenchmarkNewOptions(b *testing.B) {
	b.Run("reflect.ValueOf is always called", func(b *testing.B) {
		opts := make([]options.Lister[options.ClientOptions], b.N)

		// Create a huge string to see if we can force reflect.ValueOf to use heap
		// over stack.
		size := 16 * 1024 * 1024
		str := strings.Repeat("a", size)

		for i := 0; i < b.N; i++ {
			opts[i] = options.Client().ApplyURI("x").SetAppName(str).
				SetAuth(options.Credential{}).SetHosts([]string{"x", "y"}).
				SetDirect(true).SetTimeout(time.Second)
		}

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = NewOptions[options.ClientOptions](opts...)
		}
	})

	b.Run("reflect.ValuOf is never called", func(b *testing.B) {
		opts := make([]options.Lister[options.LoggerOptions], b.N)

		for i := 0; i < b.N; i++ {
			var lo *options.LoggerOptionsBuilder
			opts[i] = lo
		}

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = NewOptions[options.LoggerOptions](opts...)
		}
	})
}
