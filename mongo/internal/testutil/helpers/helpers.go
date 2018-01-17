// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package testhelpers

import (
	"fmt"
	"io/ioutil"
	"path"
	"strings"
	"time"

	"testing"

	"io"

	"github.com/10gen/mongo-go-driver/mongo/connstring"
	"github.com/stretchr/testify/require"
)

// FindJSONFilesInDir finds the JSON files in a directory.
func FindJSONFilesInDir(t *testing.T, dir string) []string {
	files := make([]string, 0)

	entries, err := ioutil.ReadDir(dir)
	require.NoError(t, err)

	for _, entry := range entries {
		if entry.IsDir() || path.Ext(entry.Name()) != ".json" {
			continue
		}

		files = append(files, entry.Name())
	}

	return files
}

// RequireNoErrorOnClose ensures there is not an error when calling Close.
func RequireNoErrorOnClose(t *testing.T, c io.Closer) {
	require.NoError(t, c.Close())
}

// VerifyConnStringOptions verifies the options on the connection string.
func VerifyConnStringOptions(t *testing.T, cs connstring.ConnString, options map[string]interface{}) {
	// Check that all options are present.
	for key, value := range options {

		key = strings.ToLower(key)
		switch key {
		case "appname":
			require.Equal(t, value, cs.AppName)
		case "authsource":
			require.Equal(t, value, cs.AuthSource)
		case "authmechanism":
			require.Equal(t, value, cs.AuthMechanism)
		case "authmechanismproperties":
			convertedMap := value.(map[string]interface{})
			require.Equal(t,
				mapInterfaceToString(convertedMap),
				cs.AuthMechanismProperties)
		case "connecttimeoutms":
			require.Equal(t, value, float64(cs.ConnectTimeout/time.Millisecond))
		case "heartbeatfrequencyms":
			require.Equal(t, value, float64(cs.HeartbeatInterval/time.Millisecond))
		case "maxidletimems":
			require.Equal(t, value, cs.MaxConnIdleTime)
		case "maxconnlifetimems":
			require.Equal(t, value, cs.MaxConnLifeTime)
		case "maxconnsperhost":
			require.True(t, cs.MaxIdleConnsPerHostSet)
			require.Equal(t, value, cs.MaxIdleConnsPerHost)
		case "maxidleconnsperhost":
			require.True(t, cs.MaxIdleConnsPerHostSet)
			require.Equal(t, value, cs.MaxIdleConnsPerHost)
		case "readpreference":
			require.Equal(t, value, cs.ReadPreference)
		case "readpreferencetags":
			require.Equal(t, value, cs.ReadPreferenceTagSets)
		case "replicaset":
			require.Equal(t, value, cs.ReplicaSet)
		case "serverselectiontimeoutms":
			require.Equal(t, value, float64(cs.ServerSelectionTimeout/time.Millisecond))
		case "ssl":
			require.Equal(t, value, cs.SSL)
		case "sockettimeoutms":
			require.Equal(t, value, float64(cs.SocketTimeout/time.Millisecond))
		case "wtimeoutms":
			require.Equal(t, value, float64(cs.WTimeout/time.Millisecond))
		default:
			opt, ok := cs.UnknownOptions[key]
			require.True(t, ok)
			require.Contains(t, opt, fmt.Sprint(value))
		}
	}
}

// Convert each interface{} value in the map to a string.
func mapInterfaceToString(m map[string]interface{}) map[string]string {
	out := make(map[string]string)

	for key, value := range m {
		out[key] = fmt.Sprint(value)
	}

	return out
}
