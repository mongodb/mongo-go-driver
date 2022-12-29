package unified

import (
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/logger"
)

func TestLogMessage(t *testing.T) {
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

				if err := tcase.want.is(got); err != nil {
					t.Fatalf("newLogMessage = %v, want %v", got, tcase.want)
				}
			})
		}
	})

	t.Run("validate", func(t *testing.T) {
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

				got := tcase.message.validate()
				if !errors.Is(got, tcase.want) {
					t.Errorf("expected error %v, got %v", tcase.want, got)
				}
			})
		}
	})

	t.Run("isLogActual", func(t *testing.T) {
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

				got := tcase.expected.is(tcase.actual)
				for _, err := range tcase.want {
					if !errors.Is(got, err) {
						t.Errorf("expected error %v, got %v", err, got)
					}
				}
			})
		}
	})
}

func TestClientLog(t *testing.T) {
	t.Parallel()

	t.Run("validate", func(t *testing.T) {
		for _, tcase := range []struct {
			name     string
			messages *clientLogMessages
			want     []error
		}{
			{
				"empty",
				&clientLogMessages{},
				[]error{errLogClientRequired},
			},
			{
				"valid",
				&clientLogMessages{
					Client: "client",
					LogMessages: []*logMessage{
						{
							LevelLiteral:     logger.DebugLevelLiteral,
							ComponentLiteral: logger.CommandComponentLiteral,
							Data:             bson.Raw{},
						},
					},
				},
				nil,
			},
			{
				"missing messages empty",
				&clientLogMessages{
					Client:      "client",
					LogMessages: []*logMessage{},
				},
				[]error{errLogMessagesRequired},
			},
			{
				"missing messages nil",
				&clientLogMessages{
					Client:      "client",
					LogMessages: nil,
				},
				[]error{errLogMessagesRequired},
			},
			{
				"invalid messages",
				&clientLogMessages{
					Client: "client",
					LogMessages: []*logMessage{
						{
							LevelLiteral: logger.DebugLevelLiteral,
							Data:         bson.Raw{},
						},
					},
				},
				[]error{errLogMessageInvalid},
			},
		} {
			tcase := tcase

			t.Run(tcase.name, func(t *testing.T) {
				t.Parallel()

				got := tcase.messages.validate()
				for _, err := range tcase.want {
					if !errors.Is(got, err) {
						t.Errorf("expected %v, got %v", err, got)
					}
				}
			})
		}
	})
}

func TestClientLogs(t *testing.T) {
	t.Parallel()

	t.Run("validate", func(t *testing.T) {
		t.Parallel()

		for _, tcase := range []struct {
			name     string
			messages clientLogs
			want     []error
		}{
			{
				"empty",
				clientLogs{},
				nil,
			},
			{
				"valid",
				clientLogs{
					{
						Client: "client",
						LogMessages: []*logMessage{
							{
								LevelLiteral:     logger.DebugLevelLiteral,
								ComponentLiteral: logger.CommandComponentLiteral,
								Data:             bson.Raw{},
							},
						},
					},
				},
				nil,
			},
			{
				"invalid client messages",
				clientLogs{
					{
						Client: "client",
						LogMessages: []*logMessage{
							{
								LevelLiteral: logger.DebugLevelLiteral,
								Data:         bson.Raw{},
							},
						},
					},
				},
				[]error{errLogClientInvalid},
			},
			{
				"multiple same clients",
				clientLogs{
					{
						Client: "client",
						LogMessages: []*logMessage{
							{
								LevelLiteral:     logger.DebugLevelLiteral,
								ComponentLiteral: logger.CommandComponentLiteral,
								Data:             bson.Raw{},
							},
						},
					},
					{
						Client: "client",
						LogMessages: []*logMessage{
							{
								LevelLiteral:     logger.DebugLevelLiteral,
								ComponentLiteral: logger.CommandComponentLiteral,
								Data:             bson.Raw{},
							},
						},
					},
				},
				[]error{errLogClientDuplicate},
			},
		} {
			tcase := tcase

			t.Run(tcase.name, func(t *testing.T) {
				t.Parallel()

				got := tcase.messages.validate()
				for _, err := range tcase.want {
					if !errors.Is(got, err) {
						t.Errorf("expected %v, got %v", err, got)
					}
				}
			})
		}
	})
}

//
//func TestLogMesssageClientValidator(t *testing.T) {
//	t.Parallel()
//
//	t.Run("validate", func(t *testing.T) {
//		t.Parallel()
//
//		for _, tcase := range []struct {
//			name      string
//			validator *clientLogValidator
//			want      []error
//		}{
//			{
//				"empty",
//				&clientLogValidator{},
//				nil,
//			},
//			{
//				"valid",
//				&clientLogValidator{
//					want: &clientLog{
//						Client: "client",
//						Messages: []*logMessage{
//							{
//								LevelLiteral:     logger.DebugLevelLiteral,
//								ComponentLiteral: logger.CommandComponentLiteral,
//								Data:             bson.Raw{},
//							},
//						},
//					},
//				},
//				nil,
//			},
//			{
//				"invalid messages",
//				&clientLogValidator{
//					want: &clientLog{
//						Client
//						Messages: []*expectedLogMessage{
//							{
//								LevelLiteral: logger.DebugLevelLiteral,
//								Data:         bson.Raw{},
//							},
//						},
//					},
//				},
//
//
//		} {
//			tcase := tcase
//
//			t.Run(tcase.name, func(t *testing.T) {
//				t.Parallel()
//
//				got := tcase.validator.validate()
//				for _, err := range tcase.want {
//					if !errors.Is(got, err) {
//						t.Errorf("expected %v, got %v", err, got)
//					}
//				}
//			})
//		}
//	})
//}
