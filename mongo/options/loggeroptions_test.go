package options

//func TestSetComponentLevels(t *testing.T) {
//	t.Parallel()
//
//	for _, tcase := range []struct {
//		name     string
//		argMap   []map[LogComponent]LogLevel
//		expected map[LogComponent]LogLevel
//	}{
//		{
//			"empty",
//			[]map[LogComponent]LogLevel{{}},
//			map[LogComponent]LogLevel{},
//		},
//		{
//			"one",
//			[]map[LogComponent]LogLevel{{CommandLogComponent: InfoLogLevel}},
//			map[LogComponent]LogLevel{CommandLogComponent: InfoLogLevel},
//		},
//		{
//			"two",
//			[]map[LogComponent]LogLevel{
//				{CommandLogComponent: InfoLogLevel, TopologyLogComponent: DebugLogLevel},
//			},
//			map[LogComponent]LogLevel{
//				CommandLogComponent:  InfoLogLevel,
//				TopologyLogComponent: DebugLogLevel,
//			},
//		},
//		{
//			"same",
//			[]map[LogComponent]LogLevel{
//				{CommandLogComponent: InfoLogLevel},
//				{CommandLogComponent: InfoLogLevel},
//			},
//			map[LogComponent]LogLevel{CommandLogComponent: InfoLogLevel},
//		},
//		{
//			"override",
//			[]map[LogComponent]LogLevel{
//				{CommandLogComponent: InfoLogLevel},
//				{CommandLogComponent: DebugLogLevel},
//			},
//			map[LogComponent]LogLevel{CommandLogComponent: DebugLogLevel},
//		},
//	} {
//		tcase := tcase
//
//		t.Run(tcase.name, func(t *testing.T) {
//			t.Parallel()
//
//			opts := Logger()
//			for _, arg := range tcase.argMap {
//				opts.SetComponentLevels(arg)
//			}
//
//			if len(opts.ComponentLevels) != len(tcase.expected) {
//				t.Errorf("expected %d components, got %d", len(tcase.expected),
//					len(opts.ComponentLevels))
//			}
//
//			for k, v := range tcase.expected {
//				if opts.ComponentLevels[k] != v {
//					t.Errorf("expected %v for component %v, got %v", v, k, opts.ComponentLevels[k])
//				}
//			}
//		})
//	}
//}
