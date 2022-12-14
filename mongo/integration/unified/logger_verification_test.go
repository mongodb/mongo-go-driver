package unified

//func TestExpectedLogMessageIsLogActual(t *testing.T) {
//	t.Parallel()
//
//	for _, tcase := range []struct {
//		name     string
//		expected *expectedLogMessage
//		actual   logActual
//		want     bool
//	}{
//		{
//			"empty",
//			&expectedLogMessage{},
//			logActual{},
//			false,
//		},
//		{
//			"match",
//			&expectedLogMessage{
//				LevelLiteral:     logger.DebugLevelLiteral,
//				ComponentLiteral: logger.CommandComponentLiteral,
//				Data: map[string]interface{}{
//					"message":      "Command started",
//					"databaseName": "logging-tests",
//					"commandName":  "ping",
//				},
//			},
//			logActual{
//				level:   int(logger.DebugLevel),
//				message: logger.CommandMessageStarted,
//				args: []interface{}{
//					"message", logger.CommandMessageStarted,
//					"databaseName", "logging-tests",
//					"commandName", "ping",
//				},
//			},
//			true,
//		},
//		{
//			"mismatch level",
//			&expectedLogMessage{
//				LevelLiteral:     logger.DebugLevelLiteral,
//				ComponentLiteral: logger.CommandComponentLiteral,
//				Data: map[string]interface{}{
//					"message":      "Command started",
//					"databaseName": "logging-tests",
//					"commandName":  "ping",
//				},
//			},
//			logActual{
//				level:   int(logger.InfoLevel),
//				message: logger.CommandMessageStarted,
//				args: []interface{}{
//					"message", logger.CommandMessageStarted,
//					"databaseName", "logging-tests",
//					"commandName", "ping",
//				},
//			},
//			false,
//		},
//		{
//			"mismatch message",
//			&expectedLogMessage{
//				LevelLiteral:     logger.DebugLevelLiteral,
//				ComponentLiteral: logger.CommandComponentLiteral,
//				Data: map[string]interface{}{
//					"message":      "Command started",
//					"databaseName": "logging-tests",
//					"commandName":  "ping",
//				},
//			},
//			logActual{
//				level:   int(logger.DebugLevel),
//				message: logger.CommandMessageSucceeded,
//				args: []interface{}{
//					"message", logger.CommandMessageSucceeded,
//					"databaseName", "logging-tests",
//					"commandName", "ping",
//				},
//			},
//			false,
//		},
//	} {
//		tcase := tcase
//
//		t.Run(tcase.name, func(t *testing.T) {
//			t.Parallel()
//
//			got := tcase.expected.isLogActual(tcase.actual)
//			if got != tcase.want {
//				t.Errorf("expected %v, got %v", tcase.want, got)
//			}
//		})
//	}
//}
