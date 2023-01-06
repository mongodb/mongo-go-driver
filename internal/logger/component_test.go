package logger

//func TestGetEnvComponentLevels(t *testing.T) {
//	t.Parallel()
//
//	for _, tcase := range []struct {
//		name     string
//		setenv   func() error
//		expected map[LogComponent]LogLevel
//	}{
//		{
//			name:     "no env",
//			expected: map[LogComponent]LogLevel{},
//		},
//		{
//			name: "invalid env",
//			setenv: func() error {
//				return os.Setenv("MONGODB_LOG_ALL", "invalid")
//			},
//			expected: map[LogComponent]LogLevel{},
//		},
//		{
//			name: "all env are debug",
//			setenv: func() error {
//				return os.Setenv("MONGODB_LOG_ALL", "debug")
//			},
//			expected: map[LogComponent]LogLevel{
//				CommandLogComponent:         DebugLogLevel,
//				TopologyLogComponent:        DebugLogLevel,
//				ServerSelectionLogComponent: DebugLogLevel,
//				ConnectionLogComponent:      DebugLogLevel,
//			},
//		},
//		{
//			name: "all env are info",
//			setenv: func() error {
//				return os.Setenv("MONGODB_LOG_ALL", "info")
//			},
//			expected: map[LogComponent]LogLevel{
//				CommandLogComponent:         InfoLogLevel,
//				TopologyLogComponent:        InfoLogLevel,
//				ServerSelectionLogComponent: InfoLogLevel,
//				ConnectionLogComponent:      InfoLogLevel,
//			},
//		},
//		{
//			name: "all env are warn",
//			setenv: func() error {
//				return os.Setenv("MONGODB_LOG_ALL", "warn")
//			},
//			expected: map[LogComponent]LogLevel{
//				CommandLogComponent:         InfoLogLevel,
//				TopologyLogComponent:        InfoLogLevel,
//				ServerSelectionLogComponent: InfoLogLevel,
//				ConnectionLogComponent:      InfoLogLevel,
//			},
//		},
//		{
//			name: "all env are error",
//			setenv: func() error {
//				return os.Setenv("MONGODB_LOG_ALL", "error")
//			},
//			expected: map[LogComponent]LogLevel{
//				CommandLogComponent:         InfoLogLevel,
//				TopologyLogComponent:        InfoLogLevel,
//				ServerSelectionLogComponent: InfoLogLevel,
//				ConnectionLogComponent:      InfoLogLevel,
//			},
//		},
//		{
//			name: "all env are notice",
//			setenv: func() error {
//				return os.Setenv("MONGODB_LOG_ALL", "notice")
//			},
//			expected: map[LogComponent]LogLevel{
//				CommandLogComponent:         InfoLogLevel,
//				TopologyLogComponent:        InfoLogLevel,
//				ServerSelectionLogComponent: InfoLogLevel,
//				ConnectionLogComponent:      InfoLogLevel,
//			},
//		},
//		{
//			name: "all env are trace",
//			setenv: func() error {
//				return os.Setenv("MONGODB_LOG_ALL", "trace")
//			},
//			expected: map[LogComponent]LogLevel{
//				CommandLogComponent:         DebugLogLevel,
//				TopologyLogComponent:        DebugLogLevel,
//				ServerSelectionLogComponent: DebugLogLevel,
//				ConnectionLogComponent:      DebugLogLevel,
//			},
//		},
//		{
//			name: "all env are off",
//			setenv: func() error {
//				return os.Setenv("MONGODB_LOG_ALL", "off")
//			},
//			expected: map[LogComponent]LogLevel{},
//		},
//		{
//			name: "all env weird capitalization",
//			setenv: func() error {
//				return os.Setenv("MONGODB_LOG_ALL", "DeBuG")
//			},
//			expected: map[LogComponent]LogLevel{
//				CommandLogComponent:         DebugLogLevel,
//				TopologyLogComponent:        DebugLogLevel,
//				ServerSelectionLogComponent: DebugLogLevel,
//				ConnectionLogComponent:      DebugLogLevel,
//			},
//		},
//		{
//			name: "MONGODB_LOG_COMMAND",
//			setenv: func() error {
//				return os.Setenv("MONGODB_LOG_COMMAND", "debug")
//			},
//			expected: map[LogComponent]LogLevel{
//				CommandLogComponent: DebugLogLevel,
//			},
//		},
//		{
//			name: "MONGODB_LOG_TOPOLOGY",
//			setenv: func() error {
//				return os.Setenv("MONGODB_LOG_TOPOLOGY", "debug")
//			},
//			expected: map[LogComponent]LogLevel{
//				TopologyLogComponent: DebugLogLevel,
//			},
//		},
//		{
//			name: "MONGODB_LOG_SERVER_SELECTION",
//			setenv: func() error {
//				return os.Setenv("MONGODB_LOG_SERVER_SELECTION", "debug")
//			},
//			expected: map[LogComponent]LogLevel{
//				ServerSelectionLogComponent: DebugLogLevel,
//			},
//		},
//		{
//			name: "MONGODB_LOG_CONNECTION",
//			setenv: func() error {
//				return os.Setenv("MONGODB_LOG_CONNECTION", "debug")
//			},
//			expected: map[LogComponent]LogLevel{
//				ConnectionLogComponent: DebugLogLevel,
//			},
//		},
//		{
//			name: "MONGODB_LOG_ALL overrides other env",
//			setenv: func() error {
//				err := os.Setenv("MONGODB_LOG_ALL", "debug")
//				if err != nil {
//					return err
//				}
//				return os.Setenv("MONGODB_LOG_COMMAND", "info")
//			},
//			expected: map[LogComponent]LogLevel{
//				CommandLogComponent:         DebugLogLevel,
//				TopologyLogComponent:        DebugLogLevel,
//				ServerSelectionLogComponent: DebugLogLevel,
//				ConnectionLogComponent:      DebugLogLevel,
//			},
//		},
//		{
//			name: "multiple env",
//			setenv: func() error {
//				err := os.Setenv("MONGODB_LOG_COMMAND", "info")
//				if err != nil {
//					return err
//				}
//				return os.Setenv("MONGODB_LOG_TOPOLOGY", "debug")
//			},
//			expected: map[LogComponent]LogLevel{
//				CommandLogComponent:  InfoLogLevel,
//				TopologyLogComponent: DebugLogLevel,
//			},
//		},
//	} {
//		tcase := tcase
//
//		t.Run(tcase.name, func(t *testing.T) {
//			// These tests need to run synchronously since they rely on setting environment variables.
//			os.Clearenv()
//
//			if setter := tcase.setenv; setter != nil {
//				if err := setter(); err != nil {
//					t.Fatalf("error setting env: %v", err)
//				}
//			}
//
//			levels := getEnvComponentLevels()
//			for component, level := range tcase.expected {
//				if levels[component] != level {
//					t.Errorf("expected level %v for component %v, got %v", level, component,
//						levels[component])
//				}
//			}
//		})
//	}
//}
//
//func TestMergeComponentLevels(t *testing.T) {
//	t.Parallel()
//
//	for _, tcase := range []struct {
//		name     string
//		args     []map[LogComponent]LogLevel
//		expected map[LogComponent]LogLevel
//	}{
//		{
//			name:     "empty",
//			args:     []map[LogComponent]LogLevel{},
//			expected: map[LogComponent]LogLevel{},
//		},
//		{
//			name: "one",
//			args: []map[LogComponent]LogLevel{
//				{
//					CommandLogComponent: DebugLogLevel,
//				},
//			},
//			expected: map[LogComponent]LogLevel{
//				CommandLogComponent: DebugLogLevel,
//			},
//		},
//		{
//			name: "two",
//			args: []map[LogComponent]LogLevel{
//				{
//					CommandLogComponent: DebugLogLevel,
//				},
//				{
//					TopologyLogComponent: DebugLogLevel,
//				},
//			},
//			expected: map[LogComponent]LogLevel{
//				CommandLogComponent:  DebugLogLevel,
//				TopologyLogComponent: DebugLogLevel,
//			},
//		},
//		{
//			name: "two different",
//			args: []map[LogComponent]LogLevel{
//				{
//					CommandLogComponent:  DebugLogLevel,
//					TopologyLogComponent: DebugLogLevel,
//				},
//				{
//					CommandLogComponent: InfoLogLevel,
//				},
//			},
//			expected: map[LogComponent]LogLevel{
//				CommandLogComponent:  InfoLogLevel,
//				TopologyLogComponent: DebugLogLevel,
//			},
//		},
//	} {
//		tcase := tcase
//
//		t.Run(tcase.name, func(t *testing.T) {
//			t.Parallel()
//
//			levels := mergeComponentLevels(tcase.args...)
//			for component, level := range tcase.expected {
//				if levels[component] != level {
//					t.Errorf("expected level %v for component %v, got %v", level, component,
//						levels[component])
//				}
//			}
//		})
//	}
//}
