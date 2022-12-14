package unified

//func TestParseTestFile(t *testing.T) {
//	t.Parallel()
//
//	type expected struct {
//		runOnRequirements []mtest.RunOnBlock
//		testCases         []*TestCase
//	}
//
//	for _, tcase := range []struct {
//		name     string
//		json     []byte
//		opts     []*Options
//		expected expected
//	}{
//		{
//			name: "observeLogMessages",
//			json: []byte(`{
//	"createEntities": [
//		{
//			"client": {
//				"id": "client",
//				"observeLogMessages": {
//					"command": "debug",
//					"topology": "info",
//					"serverSelection": "warn",
//					"connection": "error"
//				}
//			}
//		}
//	],
//	"tests": [
//		{
//			"description": "observeLogMessages",
//			"expectLogMessages": [
//				{
//					"client": "client",
//					"messages": [
//						{
//							"level": "debug",
//							"component": "command",
//							"data": {
//								"message": "Command started"
//							}
//						}
//					]
//				}
//			]
//
//		}
//	]
//}`),
//			expected: expected{
//				testCases: []*TestCase{
//					{
//						createEntities: []map[string]*entityOptions{
//							{
//								"client": {
//									ObserveLogMessages: &observeLogMessages{
//										Command:         logger.DebugLevelLiteral,
//										Topology:        logger.InfoLevelLiteral,
//										ServerSelection: logger.WarnLevelLiteral,
//										Connection:      logger.ErrorLevelLiteral,
//									},
//								},
//							},
//						},
//						ExpectLogMessages: []*expectedLogMessagesForClient{
//							{
//								Client: "client",
//								Messages: []*expectedLogMessage{
//									{
//										LevelLiteral:     logger.DebugLevelLiteral,
//										ComponentLiteral: logger.CommandComponentLiteral,
//										Data: map[string]interface{}{
//											"message": "Command started",
//										},
//									},
//								},
//							},
//						},
//					},
//				},
//			},
//		},
//	} {
//		tcase := tcase
//
//		t.Run(tcase.name, func(t *testing.T) {
//			t.Parallel()
//
//			_, testCases, err := parseTestFile(tcase.json, tcase.opts...)
//			if err != nil {
//				t.Fatalf("error parsing test file: %v", err)
//			}
//
//			for i, tc := range testCases {
//				if len(tc.createEntities) != len(tcase.expected.testCases[i].createEntities) {
//					t.Fatalf("expected %d createEntities, got %d",
//						len(tcase.expected.testCases[i].createEntities), len(tc.createEntities))
//				}
//
//				// Compare the expected and actual createEntities
//				for ceIdx, entityMap := range tc.createEntities {
//					expected := tcase.expected.testCases[i].createEntities[ceIdx]
//					if len(entityMap) != len(expected) {
//						t.Fatalf("expected %d createEntities, got %d", len(expected),
//							len(entityMap))
//					}
//
//					for name, opts := range entityMap {
//						expected := expected[name]
//						expectedCmd := expected.ObserveLogMessages.Command
//
//						if opts.ObserveLogMessages.Command != expectedCmd {
//							t.Fatalf("expected %q, got %q", expectedCmd,
//								opts.ObserveLogMessages.Command)
//						}
//					}
//				}
//
//				// Compare the expected and actual expectLogMessages
//				for _, expected := range tcase.expected.testCases[i].ExpectLogMessages {
//					found := false
//					for _, actual := range tc.ExpectLogMessages {
//						if expected.Client == actual.Client {
//							found = true
//							if len(expected.Messages) != len(actual.Messages) {
//								t.Fatalf("expected %d messages, got %d",
//									len(expected.Messages), len(actual.Messages))
//							}
//
//							for i, expectedMsg := range expected.Messages {
//								actualMsg := actual.Messages[i]
//
//								if expectedMsg.LevelLiteral != actualMsg.LevelLiteral {
//									t.Fatalf("expected %q, got %q",
//										expectedMsg.LevelLiteral,
//										actualMsg.LevelLiteral)
//								}
//
//								if expectedMsg.ComponentLiteral != actualMsg.ComponentLiteral {
//									t.Fatalf("expected %q, got %q",
//										expectedMsg.ComponentLiteral,
//										actualMsg.ComponentLiteral)
//								}
//
//								if len(expectedMsg.Data) != len(actualMsg.Data) {
//									t.Fatalf("expected %d data items, got %d",
//										len(expectedMsg.Data),
//										len(actualMsg.Data))
//								}
//
//								for k, v := range expectedMsg.Data {
//									if actualMsg.Data[k] != v {
//										t.Fatalf("expected %v, got %v", v,
//											actualMsg.Data[k])
//									}
//								}
//							}
//						}
//					}
//
//					if !found {
//						t.Fatalf("expected to find client %q", expected.Client)
//					}
//				}
//
//			}
//		})
//	}
//}
