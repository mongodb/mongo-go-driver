// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sync"
	"testing"

	"go.mongodb.org/mongo-driver/internal/assert"
)

type mockLogSink struct{}

func (mockLogSink) Info(int, string, ...interface{})    {}
func (mockLogSink) Error(error, string, ...interface{}) {}

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

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				logger.Print(LevelInfo, ComponentCommand, "foo", "bar", "baz")
			}
		})
	})
}

func mockKeyValues(length int) (KeyValues, map[string]interface{}) {
	keysAndValues := KeyValues{}
	m := map[string]interface{}{}

	for i := 0; i < length; i++ {
		keyName := fmt.Sprintf("key%d", i)
		valueName := fmt.Sprintf("value%d", i)

		keysAndValues.Add(keyName, valueName)
		m[keyName] = valueName
	}

	return keysAndValues, m
}

func BenchmarkIOSinkInfo(b *testing.B) {
	keysAndValues, _ := mockKeyValues(10)

	b.ReportAllocs()
	b.ResetTimer()

	sink := NewIOSink(bytes.NewBuffer(nil))

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			sink.Info(0, "foo", keysAndValues...)
		}
	})
}

func TestIOSinkInfo(t *testing.T) {
	t.Parallel()

	const threshold = 1000

	mockKeyValues, kvmap := mockKeyValues(10)

	buf := new(bytes.Buffer)
	sink := NewIOSink(buf)

	wg := sync.WaitGroup{}
	wg.Add(threshold)

	for i := 0; i < threshold; i++ {
		go func() {
			defer wg.Done()

			sink.Info(0, "foo", mockKeyValues...)
		}()
	}

	wg.Wait()

	dec := json.NewDecoder(buf)
	for dec.More() {
		var m map[string]interface{}
		if err := dec.Decode(&m); err != nil {
			t.Fatalf("error unmarshaling JSON: %v", err)
		}

		delete(m, KeyTimestamp)
		delete(m, KeyMessage)

		if !reflect.DeepEqual(m, kvmap) {
			t.Fatalf("expected %v, got %v", kvmap, m)
		}
	}
}

func TestSelectMaxDocumentLength(t *testing.T) {
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

func TestLogger_LevelComponentEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		logger    Logger
		level     Level
		component Component
		want      bool
	}{
		{
			name:      "zero",
			logger:    Logger{},
			level:     LevelOff,
			component: ComponentCommand,
			want:      false,
		},
		{
			name: "empty",
			logger: Logger{
				ComponentLevels: map[Component]Level{},
			},
			level:     LevelOff,
			component: ComponentCommand,
			want:      false, // LevelOff should never be considered enabled.
		},
		{
			name: "one level below",
			logger: Logger{
				ComponentLevels: map[Component]Level{
					ComponentCommand: LevelDebug,
				},
			},
			level:     LevelInfo,
			component: ComponentCommand,
			want:      true,
		},
		{
			name: "equal levels",
			logger: Logger{
				ComponentLevels: map[Component]Level{
					ComponentCommand: LevelDebug,
				},
			},
			level:     LevelDebug,
			component: ComponentCommand,
			want:      true,
		},
		{
			name: "one level above",
			logger: Logger{
				ComponentLevels: map[Component]Level{
					ComponentCommand: LevelInfo,
				},
			},
			level:     LevelDebug,
			component: ComponentCommand,
			want:      false,
		},
		{
			name: "component mismatch",
			logger: Logger{
				ComponentLevels: map[Component]Level{
					ComponentCommand: LevelDebug,
				},
			},
			level:     LevelDebug,
			component: ComponentTopology,
			want:      false,
		},
		{
			name: "component all enables with topology",
			logger: Logger{
				ComponentLevels: map[Component]Level{
					ComponentAll: LevelDebug,
				},
			},
			level:     LevelDebug,
			component: ComponentTopology,
			want:      true,
		},
		{
			name: "component all enables with server selection",
			logger: Logger{
				ComponentLevels: map[Component]Level{
					ComponentAll: LevelDebug,
				},
			},
			level:     LevelDebug,
			component: ComponentServerSelection,
			want:      true,
		},
		{
			name: "component all enables with connection",
			logger: Logger{
				ComponentLevels: map[Component]Level{
					ComponentAll: LevelDebug,
				},
			},
			level:     LevelDebug,
			component: ComponentConnection,
			want:      true,
		},
		{
			name: "component all enables with command",
			logger: Logger{
				ComponentLevels: map[Component]Level{
					ComponentAll: LevelDebug,
				},
			},
			level:     LevelDebug,
			component: ComponentCommand,
			want:      true,
		},
		{
			name: "component all enables with all",
			logger: Logger{
				ComponentLevels: map[Component]Level{
					ComponentAll: LevelDebug,
				},
			},
			level:     LevelDebug,
			component: ComponentAll,
			want:      true,
		},
		{
			name: "component all does not enable with lower level",
			logger: Logger{
				ComponentLevels: map[Component]Level{
					ComponentAll: LevelInfo,
				},
			},
			level:     LevelDebug,
			component: ComponentCommand,
			want:      false,
		},
		{
			name: "component all has a lower log level than command",
			logger: Logger{
				ComponentLevels: map[Component]Level{
					ComponentAll:     LevelInfo,
					ComponentCommand: LevelDebug,
				},
			},
			level:     LevelDebug,
			component: ComponentCommand,
			want:      true,
		},
		{
			name: "component all has a higher log level than command",
			logger: Logger{
				ComponentLevels: map[Component]Level{
					ComponentAll:     LevelDebug,
					ComponentCommand: LevelInfo,
				},
			},
			level:     LevelDebug,
			component: ComponentCommand,
			want:      true,
		},
	}

	for _, tcase := range tests {
		tcase := tcase // Capture the range variable.

		t.Run(tcase.name, func(t *testing.T) {
			t.Parallel()

			got := tcase.logger.LevelComponentEnabled(tcase.level, tcase.component)
			assert.Equal(t, tcase.want, got, "unexpected result for LevelComponentEnabled")
		})
	}
}
