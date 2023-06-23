package unified

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/internal/logger"
	"go.mongodb.org/mongo-driver/internal/require"
)

func newTestLogMessage(t *testing.T, level int, msg string, args ...interface{}) *logMessage {
	t.Helper()

	message, err := newLogMessage(level, msg, args...)
	require.Nil(t, err, "failed to create test log message: %v", err)

	return message
}

func TestClientLogMessages(t *testing.T) {
	t.Parallel()

	t.Run("ignore", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name    string
			clm     clientLogMessages
			message *logMessage
			want    bool
		}{
			{
				name:    "empty",
				clm:     clientLogMessages{},
				message: &logMessage{},
				want:    false,
			},
			{
				name: "no match",
				clm: clientLogMessages{
					IgnoreMessages: []*logMessage{
						newTestLogMessage(t, int(logger.LevelDebug), logger.CommandFailed),
					},
				},
				message: newTestLogMessage(t, int(logger.LevelInfo), logger.CommandFailed),
				want:    false,
			},
			{
				name: "match",
				clm: clientLogMessages{
					IgnoreMessages: []*logMessage{
						newTestLogMessage(t, int(logger.LevelDebug), logger.CommandStarted),
					},
				},
				message: newTestLogMessage(t, int(logger.LevelDebug), logger.CommandStarted),
				want:    true,
			},
			{
				name: "match subset",
				clm: clientLogMessages{
					IgnoreMessages: []*logMessage{
						newTestLogMessage(t, int(logger.LevelDebug), logger.CommandStarted),
					},
				},
				message: newTestLogMessage(t, int(logger.LevelDebug), logger.CommandStarted, "extrakey", 1),
				want:    true,
			},
		}

		for _, test := range tests {
			test := test // capture the range variable

			t.Run(test.name, func(t *testing.T) {
				t.Parallel()

				got := test.clm.ignore(context.Background(), test.message)
				assert.Equal(t, test.want, got,
					"clientLogMessages.ignore() = %v, want: %v", got, test.want)
			})
		}
	})
}
