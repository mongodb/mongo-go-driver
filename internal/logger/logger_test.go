// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package logger

import (
	"os"
	"reflect"
	"testing"
)

type mockLogSink struct{}

func (mockLogSink) Info(level int, msg string, keysAndValues ...interface{})  {}
func (mockLogSink) Error(err error, msg string, keysAndValues ...interface{}) {}

func BenchmarkLogger(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	b.Run("Print", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		logger, err := New(mockLogSink{}, 0, map[Component]Level{
			ComponentCommand: LevelDebug,
		})

		if err != nil {
			b.Fatal(err)
		}

		for i := 0; i < b.N; i++ {
			logger.Print(LevelInfo, ComponentCommand, "foo", "bar", "baz")
		}
	})
}

func TestSelectMaxDocumentLength(t *testing.T) {
	t.Parallel()

	for _, tcase := range []struct {
		name     string
		arg      uint
		expected uint
		env      map[string]string
	}{
		{
			name:     "default",
			arg:      0,
			expected: DefaultMaxDocumentLength,
		},
		{
			name:     "non-zero",
			arg:      100,
			expected: 100,
		},
		{
			name:     "valid env",
			arg:      0,
			expected: 100,
			env: map[string]string{
				maxDocumentLengthEnvVar: "100",
			},
		},
		{
			name:     "invalid env",
			arg:      0,
			expected: DefaultMaxDocumentLength,
			env: map[string]string{
				maxDocumentLengthEnvVar: "foo",
			},
		},
	} {
		tcase := tcase

		t.Run(tcase.name, func(t *testing.T) {
			for k, v := range tcase.env {
				t.Setenv(k, v)
			}

			actual := selectMaxDocumentLength(tcase.arg)
			if actual != tcase.expected {
				t.Errorf("expected %d, got %d", tcase.expected, actual)
			}
		})
	}
}

func TestSelectLogSink(t *testing.T) {
	t.Parallel()

	for _, tcase := range []struct {
		name     string
		arg      LogSink
		expected LogSink
		env      map[string]string
	}{
		{
			name:     "default",
			arg:      nil,
			expected: NewIOSink(os.Stderr),
		},
		{
			name:     "non-nil",
			arg:      mockLogSink{},
			expected: mockLogSink{},
		},
		{
			name:     "stdout",
			arg:      nil,
			expected: NewIOSink(os.Stdout),
			env: map[string]string{
				logSinkPathEnvVar: logSinkPathStdout,
			},
		},
		{
			name:     "stderr",
			arg:      nil,
			expected: NewIOSink(os.Stderr),
			env: map[string]string{
				logSinkPathEnvVar: logSinkPathStderr,
			},
		},
	} {
		tcase := tcase

		t.Run(tcase.name, func(t *testing.T) {
			for k, v := range tcase.env {
				t.Setenv(k, v)
			}

			actual, _, _ := selectLogSink(tcase.arg)
			if !reflect.DeepEqual(actual, tcase.expected) {
				t.Errorf("expected %+v, got %+v", tcase.expected, actual)
			}
		})
	}
}

func TestSelectedComponentLevels(t *testing.T) {
	t.Parallel()

	for _, tcase := range []struct {
		name     string
		arg      map[Component]Level
		expected map[Component]Level
		env      map[string]string
	}{
		{
			name: "default",
			arg:  nil,
			expected: map[Component]Level{
				ComponentCommand:         LevelOff,
				ComponentTopology:        LevelOff,
				ComponentServerSelection: LevelOff,
				ComponentConnection:      LevelOff,
			},
		},
		{
			name: "non-nil",
			arg: map[Component]Level{
				ComponentCommand: LevelDebug,
			},
			expected: map[Component]Level{
				ComponentCommand:         LevelDebug,
				ComponentTopology:        LevelOff,
				ComponentServerSelection: LevelOff,
				ComponentConnection:      LevelOff,
			},
		},
		{
			name: "valid env",
			arg:  nil,
			expected: map[Component]Level{
				ComponentCommand:         LevelDebug,
				ComponentTopology:        LevelInfo,
				ComponentServerSelection: LevelOff,
				ComponentConnection:      LevelOff,
			},
			env: map[string]string{
				mongoDBLogCommandEnvVar:  levelLiteralDebug,
				mongoDBLogTopologyEnvVar: levelLiteralInfo,
			},
		},
		{
			name: "invalid env",
			arg:  nil,
			expected: map[Component]Level{
				ComponentCommand:         LevelOff,
				ComponentTopology:        LevelOff,
				ComponentServerSelection: LevelOff,
				ComponentConnection:      LevelOff,
			},
			env: map[string]string{
				mongoDBLogCommandEnvVar:  "foo",
				mongoDBLogTopologyEnvVar: "bar",
			},
		},
	} {
		tcase := tcase

		t.Run(tcase.name, func(t *testing.T) {
			for k, v := range tcase.env {
				t.Setenv(k, v)
			}

			actual := selectComponentLevels(tcase.arg)
			for k, v := range tcase.expected {
				if actual[k] != v {
					t.Errorf("expected %d, got %d", v, actual[k])
				}
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	t.Parallel()

	for _, tcase := range []struct {
		name     string
		arg      string
		width    uint
		expected string
	}{
		{
			name:     "empty",
			arg:      "",
			width:    0,
			expected: "",
		},
		{
			name:     "short",
			arg:      "foo",
			width:    DefaultMaxDocumentLength,
			expected: "foo",
		},
		{
			name:     "long",
			arg:      "foo bar baz",
			width:    9,
			expected: "foo bar b...",
		},
		{
			name:     "multi-byte",
			arg:      "你好",
			width:    4,
			expected: "你...",
		},
	} {
		tcase := tcase

		t.Run(tcase.name, func(t *testing.T) {
			t.Parallel()

			actual := truncate(tcase.arg, tcase.width)
			if actual != tcase.expected {
				t.Errorf("expected %q, got %q", tcase.expected, actual)
			}
		})
	}

}
