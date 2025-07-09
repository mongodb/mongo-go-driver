// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

// const baseURL = "https://performance-monitoring-and-analysis.server-tig.prod.corp.mongodb.com/baron"

type CPFilter struct {
	Active   bool        `bson:"active"`
	Name     string      `bson:"name"`
	Operator string      `bson:"operator"`
	Type     string      `bson:"type"`
	Value    interface{} `json:"value,omitempty"`
}

/*
[{"active":true,"name":"commit","operator":"matches","type":"regex","value":""},
{"active":true,"name":"commit_date","operator":"after","type":"date"},
{"active":true,"name":"calculated_on","operator":"after","type":"date"},
{"active":true,"name":"project","operator":"matches","type":"regex","value":""},
{"active":true,"name":"variant","operator":"matches","type":"regex","value":""},
{"active":true,"name":"task","operator":"matches","type":"regex","value":""},
{"active":true,"name":"test","operator":"matches","type":"regex","value":""},
{"active":true,"name":"measurement","operator":"matches","type":"regex","value":""},
{"active":true,"name":"args","operator":"eq","type":"json"},
{"active":true,"name":"percent_change","operator":"gt","type":"number"},
{"active":true,"name":"z_score_change","operator":"gt","type":"number"},
{"active":true,"name":"h_score","operator":"gt","type":"number"},
{"active":true,"name":"absolute_change","operator":"gt","type":"number"},
{"active":true,"name":"build_failures","operator":"matches","type":"regex","value":""},
{"active":true,"name":"bf_suggestions","operator":"inlist","type":"listSelect"},
{"active":true,"name":"triage_status","operator":"inlist","type":"listSelect"},
{"active":true,"name":"changeType","operator":"inlist","type":"listSelect"},
{"active":true,"name":"triage_contexts","operator":"inlist","type":"listSelect","value":["GoDriver perf (h-score)"]}]
*/
