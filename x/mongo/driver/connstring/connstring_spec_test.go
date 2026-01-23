// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package connstring_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"path"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/internal/spectest"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/connstring"
)

type host struct {
	Type string
	Host string
	Port json.Number
}

type auth struct {
	Username string
	Password *string
	DB       string
}

type testCase struct {
	Description string
	URI         string
	Valid       bool
	Warning     bool
	Hosts       []host
	Auth        *auth
	Options     map[string]any
}

type testContainer struct {
	Tests []testCase
}

var (
	connstringTestsDir = spectest.Path("connection-string/tests")
	urioptionsTestDir  = spectest.Path("uri-options/tests")
)

func (h *host) toString() string {
	switch h.Type {
	case "unix":
		return h.Host
	case "ip_literal":
		if len(h.Port) == 0 {
			return "[" + h.Host + "]"
		}
		return "[" + h.Host + "]" + ":" + string(h.Port)
	case "ipv4":
		fallthrough
	case "hostname":
		if len(h.Port) == 0 {
			return h.Host
		}
		return h.Host + ":" + string(h.Port)
	}

	return ""
}

func hostsToStrings(hosts []host) []string {
	out := make([]string, len(hosts))

	for i, host := range hosts {
		out[i] = host.toString()
	}

	return out
}

func runTestsInFile(t *testing.T, dirname string, filename string, warningsError bool) {
	filepath := path.Join(dirname, filename)
	content, err := ioutil.ReadFile(filepath)
	require.NoError(t, err)

	var container testContainer
	require.NoError(t, json.Unmarshal(content, &container))

	t.Run(filename, func(t *testing.T) {
		for _, testCase := range container.Tests {
			testCase := testCase // Capture range variable.

			t.Run(testCase.Description, func(t *testing.T) {
				spectest.CheckSkip(t)

				runTest(t, testCase, warningsError)
			})
		}
	})
}

func runTest(t *testing.T, test testCase, warningsError bool) {
	spectest.CheckSkip(t)

	cs, err := connstring.ParseAndValidate(test.URI)
	// Since we don't have warnings in Go, we return warnings as errors.
	//
	// This is a bit unfortunate, but since we do raise warnings as errors with the newer
	// URI options, but don't with some of the older things, we do a switch on the filename
	// here. We are trying to not break existing user applications that have unrecognized
	// options.
	if test.Valid && (!test.Warning || !warningsError) {
		require.NoError(t, err)
	} else {
		require.Error(t, err)
		return
	}

	require.Equal(t, test.URI, cs.Original)

	if test.Hosts != nil {
		require.Equal(t, hostsToStrings(test.Hosts), cs.Hosts)
	}

	if test.Auth != nil {
		require.Equal(t, test.Auth.Username, cs.Username)

		if test.Auth.Password == nil {
			require.False(t, cs.PasswordSet)
		} else {
			require.True(t, cs.PasswordSet)
			require.Equal(t, *test.Auth.Password, cs.Password)
		}

		if test.Auth.DB != cs.Database {
			require.Equal(t, test.Auth.DB, cs.AuthSource)
		} else {
			require.Equal(t, test.Auth.DB, cs.Database)
		}
	}

	// Check that all options are present.
	verifyConnStringOptions(t, cs, test.Options)

	// Check that non-present options are unset. This will be redundant with the above checks
	// for options that are present.
	var ok bool

	_, ok = test.Options["maxPoolSize"]
	require.Equal(t, ok, cs.MaxPoolSizeSet)
}

// Test case for all connection string spec tests.
func TestConnStringSpec(t *testing.T) {
	for _, file := range spectest.FindJSONFilesInDir(t, connstringTestsDir) {
		runTestsInFile(t, connstringTestsDir, file, false)
	}
}

func TestURIOptionsSpec(t *testing.T) {
	for _, file := range spectest.FindJSONFilesInDir(t, urioptionsTestDir) {
		runTestsInFile(t, urioptionsTestDir, file, true)
	}
}

// verifyConnStringOptions verifies the options on the connection string.
func verifyConnStringOptions(t *testing.T, cs *connstring.ConnString, options map[string]any) {
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
			convertedMap := value.(map[string]any)
			require.Equal(t,
				mapInterfaceToString(convertedMap),
				cs.AuthMechanismProperties)
		case "compressors":
			require.Equal(t, convertToStringSlice(value), cs.Compressors)
		case "connecttimeoutms":
			require.Equal(t, value, float64(cs.ConnectTimeout/time.Millisecond))
		case "directconnection":
			require.True(t, cs.DirectConnectionSet)
			require.Equal(t, value, cs.DirectConnection)
		case "heartbeatfrequencyms":
			require.Equal(t, value, float64(cs.HeartbeatInterval/time.Millisecond))
		case "journal":
			require.True(t, cs.JSet)
			require.Equal(t, value, cs.J)
		case "loadbalanced":
			require.True(t, cs.LoadBalancedSet)
			require.Equal(t, value, cs.LoadBalanced)
		case "localthresholdms":
			require.True(t, cs.LocalThresholdSet)
			require.Equal(t, value, float64(cs.LocalThreshold/time.Millisecond))
		case "maxidletimems":
			require.Equal(t, value, float64(cs.MaxConnIdleTime/time.Millisecond))
		case "maxpoolsize":
			require.True(t, cs.MaxPoolSizeSet)
			require.Equal(t, value, float64(cs.MaxPoolSize))
		case "maxstalenessseconds":
			require.True(t, cs.MaxStalenessSet)
			require.Equal(t, value, float64(cs.MaxStaleness/time.Second))
		case "minpoolsize":
			require.True(t, cs.MinPoolSizeSet)
			require.Equal(t, value, float64(cs.MinPoolSize))
		case "readpreference":
			require.Equal(t, value, cs.ReadPreference)
		case "readpreferencetags":
			sm, ok := value.([]any)
			require.True(t, ok)
			tags := make([]map[string]string, 0, len(sm))
			for _, i := range sm {
				m, ok := i.(map[string]any)
				require.True(t, ok)
				tags = append(tags, mapInterfaceToString(m))
			}
			require.Equal(t, tags, cs.ReadPreferenceTagSets)
		case "readconcernlevel":
			require.Equal(t, value, cs.ReadConcernLevel)
		case "replicaset":
			require.Equal(t, value, cs.ReplicaSet)
		case "retrywrites":
			require.True(t, cs.RetryWritesSet)
			require.Equal(t, value, cs.RetryWrites)
		case "serverselectiontimeoutms":
			require.Equal(t, value, float64(cs.ServerSelectionTimeout/time.Millisecond))
		case "srvmaxhosts":
			require.Equal(t, value, float64(cs.SRVMaxHosts))
		case "srvservicename":
			require.Equal(t, value, cs.SRVServiceName)
		case "ssl", "tls":
			require.Equal(t, value, cs.SSL)
		case "sockettimeoutms":
			require.Equal(t, value, float64(cs.SocketTimeout/time.Millisecond))
		case "tlsallowinvalidcertificates", "tlsallowinvalidhostnames", "tlsinsecure":
			require.True(t, cs.SSLInsecureSet)
			require.Equal(t, value, cs.SSLInsecure)
		case "tlscafile":
			require.True(t, cs.SSLCaFileSet)
			require.Equal(t, value, cs.SSLCaFile)
		case "tlscertificatekeyfile":
			require.True(t, cs.SSLClientCertificateKeyFileSet)
			require.Equal(t, value, cs.SSLClientCertificateKeyFile)
		case "tlscertificatekeyfilepassword":
			require.True(t, cs.SSLClientCertificateKeyPasswordSet)
			require.Equal(t, value, cs.SSLClientCertificateKeyPassword())
		case "w":
			if cs.WNumberSet {
				valueInt := getIntFromInterface(value)
				require.NotNil(t, valueInt)
				require.Equal(t, *valueInt, int64(cs.WNumber))
			} else {
				require.Equal(t, value, cs.WString)
			}
		case "waitqueuetimeoutms":
		case "zlibcompressionlevel":
			require.Equal(t, value, float64(cs.ZlibLevel))
		case "zstdcompressionlevel":
			require.Equal(t, value, float64(cs.ZstdLevel))
		case "tlsdisableocspendpointcheck":
			require.Equal(t, value, cs.SSLDisableOCSPEndpointCheck)
		case "servermonitoringmode":
			require.Equal(t, value, cs.ServerMonitoringMode)
		case "timeoutms":
			require.Equal(t, value, float64(cs.Timeout/time.Millisecond))
		case "maxconnecting":
			require.Equal(t, value, float64(cs.MaxConnecting))
		default:
			opt, ok := cs.UnknownOptions[key]
			require.True(t, ok)
			require.Contains(t, opt, fmt.Sprint(value))
		}
	}
}

// Convert each any value in the map to a string.
func mapInterfaceToString(m map[string]any) map[string]string {
	out := make(map[string]string)

	for key, value := range m {
		out[key] = fmt.Sprint(value)
	}

	return out
}

// getIntFromInterface attempts to convert an empty interface value to an integer.
//
// Returns nil if it is not possible.
func getIntFromInterface(i any) *int64 {
	var out int64

	switch v := i.(type) {
	case int:
		out = int64(v)
	case int32:
		out = int64(v)
	case int64:
		out = v
	case float32:
		f := float64(v)
		if math.Floor(f) != f || f > float64(math.MaxInt64) {
			break
		}

		out = int64(f)

	case float64:
		if math.Floor(v) != v || v > float64(math.MaxInt64) {
			break
		}

		out = int64(v)
	default:
		return nil
	}

	return &out
}

func convertToStringSlice(i any) []string {
	s, ok := i.([]any)
	if !ok {
		return nil
	}
	ret := make([]string, 0, len(s))
	for _, v := range s {
		str, ok := v.(string)
		if !ok {
			continue
		}
		ret = append(ret, str)
	}
	return ret
}
