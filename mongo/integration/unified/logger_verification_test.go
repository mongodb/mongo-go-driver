package unified

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/logger"
)

func TestLoggerVerification(t *testing.T) {
	t.Parallel()

	t.Run("newLogMessage", func(t *testing.T) {
		t.Parallel()

		for _, tcase := range []struct {
			name  string
			level int
			args  []interface{}
			want  *logMessage
			err   error
		}{
			{
				"no args",
				int(logger.InfoLevel),
				nil,
				&logMessage{
					LevelLiteral: logger.InfoLevelLiteral,
				},
				nil,
			},
			{
				"one arg",
				int(logger.InfoLevel),
				[]interface{}{"hello"},
				&logMessage{
					LevelLiteral: logger.InfoLevelLiteral,
				},
				errLogStructureInvalid,
			},
			{
				"two args",
				int(logger.InfoLevel),
				[]interface{}{"hello", "world"},
				&logMessage{
					LevelLiteral: logger.InfoLevelLiteral,
					Data: func() bson.Raw {
						raw, _ := bson.Marshal(bson.D{{"hello", "world"}})
						return raw
					}(),
				},
				nil,
			},
		} {
			tcase := tcase

			t.Run(tcase.name, func(t *testing.T) {
				t.Parallel()

				got, err := newLogMessage(tcase.level, tcase.args...)
				if tcase.err != nil {
					if !errors.Is(err, tcase.err) {
						t.Fatalf("newLogMessage error = %v, want %v", err, tcase.err)
					}

					return
				}

				err = verifyLogMessagesMatch(context.Background(), tcase.want, got)
				if err != nil {
					t.Fatalf("newLogMessage = %v, want %v", got, tcase.want)
				}
			})
		}
	})

	t.Run("newLogMessageValidator", func(t *testing.T) {
		t.Parallel()

		for _, tcase := range []struct {
			name     string
			testCase *TestCase
			want     *logMessageValidator
			err      error
		}{
			{
				"nil",
				nil,
				nil,
				errTestCaseRequired,
			},
			{
				"empty test case",
				&TestCase{},
				nil,
				errEntitiesRequired,
			},
			{
				"no log messages",
				&TestCase{
					entities: &EntityMap{
						clientEntities: map[string]*clientEntity{
							"client0": {},
						},
					},
				},
				&logMessageValidator{
					expected: []*clientLogMessages{},
					done:     make(chan struct{}, 1),
					err:      make(chan error, 1),
				},
				nil,
			},
			{
				"one log message",
				&TestCase{
					entities: &EntityMap{
						clientEntities: map[string]*clientEntity{
							"client0": {
								logQueue: make(chan orderedLogMessage, 1),
							},
						},
					},
					ExpectLogMessages: []*clientLogMessages{
						{
							Client: "client0",
							LogMessages: []*logMessage{
								{
									LevelLiteral: logger.InfoLevelLiteral,
								},
							},
						},
					},
				},
				&logMessageValidator{
					expected: []*clientLogMessages{
						{
							Client: "client0",
							LogMessages: []*logMessage{
								{
									LevelLiteral: logger.InfoLevelLiteral,
								},
							},
						},
					},
					actualQueues: map[string]chan orderedLogMessage{
						"client0": make(chan orderedLogMessage, 1),
					},
					done: make(chan struct{}, 1),
					err:  make(chan error, 1),
				},
				nil,
			},
		} {
			tcase := tcase

			t.Run(tcase.name, func(t *testing.T) {
				t.Parallel()

				got, err := newLogMessageValidator(tcase.testCase)
				if tcase.err != nil {
					if !errors.Is(err, tcase.err) {
						t.Fatalf("newLogMessageValidator error = %v, want %v", err, tcase.err)
					}

					return
				}

				if got == nil {
					t.Fatalf("newLogMessageValidator = nil, want %v", tcase.want)
				}

				if !reflect.DeepEqual(got.expected, tcase.want.expected) {
					t.Fatalf("newLogMessageValidator expected = %v, want %v", got.expected,
						tcase.want.expected)
				}

				for k, v := range got.actualQueues {
					if _, ok := tcase.want.actualQueues[k]; !ok {
						t.Fatalf("newLogMessageValidator actualQueues = %v, want %v",
							got.actualQueues,
							tcase.want.actualQueues)
					}

					if cap(v) != cap(tcase.want.actualQueues[k]) {
						t.Fatalf("newLogMessageValidator actualQueues = %v, want %v",
							got.actualQueues,
							tcase.want.actualQueues)
					}

					if len(v) != len(tcase.want.actualQueues[k]) {
						t.Fatalf("newLogMessageValidator actualQueues = %v, want %v",
							got.actualQueues,
							tcase.want.actualQueues)
					}
				}

				if len(got.done) != len(tcase.want.done) {
					t.Fatalf("newLogMessageValidator done = %v, want %v",
						len(got.done),
						len(tcase.want.done))
				}

				if len(got.err) != len(tcase.want.err) {
					t.Fatalf("newLogMessageValidator err = %v, want %v",
						len(got.err),
						len(tcase.want.err))
				}
			})
		}
	})

	t.Run("validateLogMessage", func(t *testing.T) {
		t.Parallel()

		for _, tcase := range []struct {
			name    string
			message *logMessage
			want    error
		}{
			{
				"valid",
				&logMessage{
					LevelLiteral:     logger.InfoLevelLiteral,
					ComponentLiteral: logger.CommandComponentLiteral,
					Data:             bson.Raw{},
				},
				nil,
			},
			{
				"empty level",
				&logMessage{
					LevelLiteral:     "",
					ComponentLiteral: logger.CommandComponentLiteral,
					Data:             bson.Raw{},
				},
				errLogLevelRequired,
			},
			{
				"empty component",
				&logMessage{
					LevelLiteral:     logger.InfoLevelLiteral,
					ComponentLiteral: "",
					Data:             bson.Raw{},
				},
				errLogComponentRequired,
			},
		} {
			tcase := tcase

			t.Run(tcase.name, func(t *testing.T) {
				t.Parallel()

				got := validateLogMessage(context.Background(), tcase.message)
				if !errors.Is(got, tcase.want) {
					t.Errorf("expected error %v, got %v", tcase.want, got)
				}
			})
		}
	})

	t.Run("verifyLogMessagesMatch", func(t *testing.T) {

		t.Parallel()

		for _, tcase := range []struct {
			name     string
			expected *logMessage
			actual   *logMessage
			want     []error
		}{
			{
				"empty",
				&logMessage{},
				&logMessage{},
				nil,
			},
			{
				"match",
				&logMessage{
					LevelLiteral:     logger.DebugLevelLiteral,
					ComponentLiteral: logger.CommandComponentLiteral,
					Data: func() bson.Raw {
						data, _ := bson.Marshal(bson.D{
							{"message", "Command started"},
							{"databaseName", "logging-tests"},
							{"commandName", "ping"},
						})

						return data
					}(),
				},
				&logMessage{
					LevelLiteral:     logger.DebugLevelLiteral,
					ComponentLiteral: logger.CommandComponentLiteral,
					Data: func() bson.Raw {
						data, _ := bson.Marshal(bson.D{
							{"message", "Command started"},
							{"databaseName", "logging-tests"},
							{"commandName", "ping"},
						})

						return data
					}(),
				},
				nil,
			},
			{
				"mismatch level",
				&logMessage{
					LevelLiteral:     logger.DebugLevelLiteral,
					ComponentLiteral: logger.CommandComponentLiteral,
					Data: func() bson.Raw {
						data, _ := bson.Marshal(bson.D{
							{"message", "Command started"},
							{"databaseName", "logging-tests"},
							{"commandName", "ping"},
						})

						return data
					}(),
				},
				&logMessage{
					LevelLiteral:     logger.InfoLevelLiteral,
					ComponentLiteral: logger.CommandComponentLiteral,
					Data: func() bson.Raw {
						data, _ := bson.Marshal(bson.D{
							{"message", "Command started"},
							{"databaseName", "logging-tests"},
							{"commandName", "ping"},
						})

						return data
					}(),
				},
				[]error{errLogLevelMismatch},
			},
			{
				"mismatch message",
				&logMessage{
					LevelLiteral:     logger.DebugLevelLiteral,
					ComponentLiteral: logger.CommandComponentLiteral,
					Data: func() bson.Raw {
						data, _ := bson.Marshal(bson.D{
							{"message", "Command started"},
							{"databaseName", "logging-tests"},
							{"commandName", "ping"},
						})

						return data
					}(),
				},
				&logMessage{
					LevelLiteral:     logger.DebugLevelLiteral,
					ComponentLiteral: logger.CommandComponentLiteral,
					Data: func() bson.Raw {
						data, _ := bson.Marshal(bson.D{
							{"message", "Command succeeded"},
							{"databaseName", "logging-tests"},
							{"commandName", "ping"},
						})

						return data
					}(),
				},
				[]error{errLogDocumentMismatch},
			},
		} {
			tcase := tcase

			t.Run(tcase.name, func(t *testing.T) {
				t.Parallel()

				got := verifyLogMessagesMatch(context.Background(), tcase.expected, tcase.actual)
				for _, err := range tcase.want {
					if !errors.Is(got, err) {
						t.Errorf("expected error %v, got %v", err, got)
					}
				}
			})
		}

	})

	t.Run("validateClientLogMessages", func(t *testing.T) {
		t.Parallel()

		for _, tcase := range []struct {
			name              string
			clientLogMessages *clientLogMessages
			want              error
		}{
			{
				"empty",
				&clientLogMessages{},
				errLogClientRequired,
			},
			{
				"no messages",
				&clientLogMessages{
					Client: "client",
				},
				errLogMessagesRequired,
			},
		} {
			tcase := tcase

			t.Run(tcase.name, func(t *testing.T) {
				t.Parallel()

				got := validateClientLogMessages(context.Background(), tcase.clientLogMessages)
				if !errors.Is(got, tcase.want) {
					t.Errorf("expected error %v, got %v", tcase.want, got)
				}
			})
		}
	})

	t.Run("validateExpectLogMessages", func(t *testing.T) {
		t.Parallel()

		for _, tcase := range []struct {
			name              string
			expectLogMessages []*clientLogMessages
			want              error
		}{
			{
				"empty",
				[]*clientLogMessages{},
				nil,
			},
			{
				"duplicated clients",
				[]*clientLogMessages{
					{
						Client: "client",
						LogMessages: []*logMessage{
							{
								LevelLiteral:     logger.DebugLevelLiteral,
								ComponentLiteral: logger.CommandComponentLiteral,
								Data:             []byte(`{x: 1}`),
							},
						},
					},
					{
						Client: "client",
						LogMessages: []*logMessage{
							{
								LevelLiteral:     logger.DebugLevelLiteral,
								ComponentLiteral: logger.CommandComponentLiteral,
								Data:             []byte(`{x: 1}`),
							},
						},
					},
				},
				errLogClientDuplicate,
			},
		} {
			tcase := tcase

			t.Run(tcase.name, func(t *testing.T) {
				t.Parallel()

				got := validateExpectLogMessages(context.Background(), tcase.expectLogMessages)
				if !errors.Is(got, tcase.want) {
					t.Errorf("expected error %v, got %v", tcase.want, got)
				}
			})
		}
	})

	t.Run("findClientLogMessages", func(t *testing.T) {
		t.Parallel()

		for _, tcase := range []struct {
			name              string
			clientLogMessages []*clientLogMessages
			clientID          string
			want              *clientLogMessages
		}{
			{
				"empty",
				[]*clientLogMessages{},
				"client",
				nil,
			},
			{
				"not found",
				[]*clientLogMessages{
					{
						Client: "client",
						LogMessages: []*logMessage{
							{
								LevelLiteral:     logger.DebugLevelLiteral,
								ComponentLiteral: logger.CommandComponentLiteral,
								Data:             []byte(`{x: 1}`),
							},
						},
					},
				},
				"client2",
				nil,
			},
			{
				"found",
				[]*clientLogMessages{
					{
						Client: "client",
						LogMessages: []*logMessage{
							{
								LevelLiteral:     logger.DebugLevelLiteral,
								ComponentLiteral: logger.CommandComponentLiteral,
								Data:             []byte(`{x: 1}`),
							},
						},
					},
				},
				"client",
				&clientLogMessages{
					Client: "client",
					LogMessages: []*logMessage{
						{
							LevelLiteral:     logger.DebugLevelLiteral,
							ComponentLiteral: logger.CommandComponentLiteral,
							Data:             []byte(`{x: 1}`),
						},
					},
				},
			},
		} {
			tcase := tcase

			t.Run(tcase.name, func(t *testing.T) {
				t.Parallel()

				got := findClientLogMessages(tcase.clientID, tcase.clientLogMessages)
				if got == nil && tcase.want == nil {
					return
				}

				if got.Client != tcase.want.Client {
					t.Errorf("expected client %s, got %s", tcase.want.Client, got.Client)
				}

				for idx, logMessage := range got.LogMessages {
					err := verifyLogMessagesMatch(context.Background(), logMessage,
						tcase.want.LogMessages[idx])

					if err != nil {
						t.Errorf("expected log messages to match, got %v", err)
					}
				}
			})
		}
	})

	t.Run("findClientLogMessagesVolume", func(t *testing.T) {
		t.Parallel()

		for _, tcase := range []struct {
			name              string
			clientLogMessages []*clientLogMessages
			clientID          string
			want              int
		}{
			{
				"empty",
				[]*clientLogMessages{},
				"client",
				0,
			},
			{
				"not found",
				[]*clientLogMessages{
					{
						Client: "client",
						LogMessages: []*logMessage{
							{
								LevelLiteral:     logger.DebugLevelLiteral,
								ComponentLiteral: logger.CommandComponentLiteral,
								Data:             []byte(`{x: 1}`),
							},
						},
					},
				},
				"client2",
				0,
			},
			{
				"found",
				[]*clientLogMessages{
					{
						Client: "client",
						LogMessages: []*logMessage{
							{
								LevelLiteral:     logger.DebugLevelLiteral,
								ComponentLiteral: logger.CommandComponentLiteral,
								Data:             []byte(`{x: 1}`),
							},
						},
					},
				},
				"client",
				1,
			},
		} {
			tcase := tcase

			t.Run(tcase.name, func(t *testing.T) {
				t.Parallel()

				got := findClientLogMessagesVolume(tcase.clientID, tcase.clientLogMessages)
				if got != tcase.want {
					t.Errorf("expected volume %d, got %d", tcase.want, got)
				}
			})
		}
	})

	t.Run("startLogMessageVerificationWorkers", func(t *testing.T) {
		t.Parallel()

		for _, tcase := range []struct {
			name      string
			validator *logMessageValidator
			want      error
			deadline  time.Duration
		}{
			{
				"empty",
				&logMessageValidator{},
				nil,
				10 * time.Millisecond,
			},
			{
				"one message verified",
				createMockLogMessageValidator(t, mockLogMessageValidatorConfig{
					size:          1,
					sizePerClient: 1,
				}),
				nil,
				10 * time.Millisecond,
			},
			{
				"one-hundred messages verified",
				createMockLogMessageValidator(t, mockLogMessageValidatorConfig{
					size:          100,
					sizePerClient: 1,
				}),
				nil,
				10 * time.Millisecond,
			},
			{
				"one-hundred messages verified with one-thousand logs per client",
				createMockLogMessageValidator(t, mockLogMessageValidatorConfig{
					size:          100,
					sizePerClient: 1000,
				}),
				nil,
				10 * time.Millisecond,
			},
			{
				"fail propagation",
				createMockLogMessageValidator(t, mockLogMessageValidatorConfig{
					size:            2,
					sizePerClient:   1,
					failPropagation: 1,
				}),
				errLogContextCanceled,
				10 * time.Millisecond,
			},
		} {
			tcase := tcase

			t.Run(tcase.name, func(t *testing.T) {
				t.Parallel()

				testCtx := context.Background()

				go startLogMessageVerificationWorkers(testCtx, tcase.validator)

				ctx, cancel := context.WithDeadline(testCtx, time.Now().Add(tcase.deadline))
				defer cancel()

				err := stopLogMessageVerificationWorkers(ctx, tcase.validator)

				// Compare the error to the test case's expected error.
				if !errors.Is(err, tcase.want) {
					t.Errorf("expected error %v, got %v", tcase.want, err)

					return
				}
			})
		}
	})
}

type mockLogMessageValidatorConfig struct {
	size             int
	sizePerClient    int
	duplicateClients bool
	failPropagation  int // Fail to send N log messages to the "actual" channel.
}

func createMockLogMessageValidator(t *testing.T, cfg mockLogMessageValidatorConfig) *logMessageValidator {
	t.Helper()

	validator := &logMessageValidator{
		done: make(chan struct{}, cfg.size),
		err:  make(chan error, 1),
	}

	{
		// Populate the expected log messages.
		validator.expected = make([]*clientLogMessages, 0, cfg.size)
		for i := 0; i < cfg.size; i++ {
			clientName := fmt.Sprintf("client-%d", i)

			// For the client, create "sizePerClient" log messages.
			logMessages := make([]*logMessage, 0, cfg.sizePerClient)
			for j := 0; j < cfg.sizePerClient; j++ {
				logMessages = append(logMessages, &logMessage{
					LevelLiteral:     logger.DebugLevelLiteral,
					ComponentLiteral: logger.CommandComponentLiteral,
					Data:             []byte(fmt.Sprintf(`{"x": %d}`, j)),
				})
			}

			validator.expected = append(validator.expected, &clientLogMessages{
				Client:      clientName,
				LogMessages: logMessages,
			})
		}

		// If the test case requires duplicate clients and size > 1, then replace the last log with the first.
		if cfg.duplicateClients && cfg.size > 1 {
			validator.expected[cfg.size-1] = validator.expected[0]
		}
	}

	{
		// Create the actual queues.
		validator.actualQueues = make(map[string]chan orderedLogMessage, cfg.size)

		for i := 0; i < cfg.size; i++ {
			clientName := fmt.Sprintf("client-%d", i)
			validator.actualQueues[clientName] = make(chan orderedLogMessage, cfg.sizePerClient)

			// For the client, create "sizePerClient" log messages.
			for j := 0; j < cfg.sizePerClient-cfg.failPropagation; j++ {
				validator.actualQueues[clientName] <- orderedLogMessage{
					order: j + 1,
					logMessage: &logMessage{
						LevelLiteral:     logger.DebugLevelLiteral,
						ComponentLiteral: logger.CommandComponentLiteral,
						Data:             []byte(fmt.Sprintf(`{"x": %d}`, j)),
					},
				}
			}

			// If we fail to propage any number of messages, the log sink will not close the log queue
			// channel.
			if cfg.failPropagation == 0 {
				close(validator.actualQueues[clientName])
			}
		}
	}

	return validator
}
