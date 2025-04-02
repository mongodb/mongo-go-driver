// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package spectest

import "testing"

// skipTests is a map of "fully-qualified test name" to "the reason for skipping
// the test".
var skipTests = map[string]string{
	"TestURIOptionsSpec/single-threaded-options.json/Valid_options_specific_to_single-threaded_drivers_are_parsed_correctly": "The Go Driver is not single-threaded.",
	// GODRIVER-2348: The wtimeoutMS write concern option is not supported.
	"TestURIOptionsSpec/concern-options.json/Valid_read_and_write_concern_are_parsed_correctly": "The wtimeoutMS write concern option is not supported",

	// SPEC-1403: This test checks to see if the correct error is thrown when
	// auto encrypting with a server < 4.2. Currently, the test will fail
	// because a server < 4.2 wouldn't have mongocryptd, so Client construction
	// would fail with a mongocryptd spawn error.
	"TestClientSideEncryptionSpec/maxWireVersion.json/operation_fails_with_maxWireVersion_<_8": "Servers less than 4.2 do not have mongocryptd; see SPEC-1403",

	// GODRIVER-1827: These 2 tests assert that in-use connections are not
	// closed until checked back into a closed pool, but the Go connection pool
	// aggressively closes in-use connections. That behavior is currently
	// required by the "Client.Disconnect" API, so skip the tests.
	"TestCMAPSpec/pool-close-destroy-conns.json/When_a_pool_is_closed,_it_MUST_first_destroy_all_available_connections_in_that_pool": "Test requires that close does not aggressively close used connections",
	"TestCMAPSpec/pool-close-destroy-conns.json/must_destroy_checked_in_connection_if_pool_has_been_closed":                          "Test requires that close does not aggressively close used connections",

	// GODRIVER-1826: The load-balancer SDAM error handling test "errors during
	// authentication are processed" currently asserts that handshake errors
	// trigger events "pool cleared" then "connection closed". However, the
	// "error during minPoolSize population clears pool" test asserts that
	// handshake errors trigger events "connection closed" then "pool cleared".
	// The Go driver uses the same code path for creating all application
	// connections, so those opposing event orders cannot be satisfied
	// simultaneously.
	//
	// TODO(DRIVERS-1785): Re-enable this test once the spec test is updated to
	// use the same event order as the "errors during authentication are
	// processed" load-balancer SDAM spec test.
	"TestCMAPSpec/pool-create-min-size-error.json/error_during_minPoolSize_population_clears_pool": "Event ordering is incompatible with load-balancer SDAM spec test (DRIVERS-1785)",

	// GODRIVER-1826: The Go connection pool does not currently always deliver
	// connections created by maintain() to waiting check-outs. There is a race
	// condition between the goroutine started by maintain() to check-in a
	// requested connection and createConnections() picking up the next wantConn
	// created by the waiting check-outs. Most of the time, createConnections()
	// wins and starts creating new connections. That is not a problem for
	// general use cases, but it prevents the "threads blocked by maxConnecting
	// check out minPoolSize connections" test from passing.
	//
	// TODO(DRIVERS-2225): Re-enable this test once the spec test is updated to
	// support the Go pool minPoolSize maintain() behavior.
	"TestCMAPSpec/pool-checkout-minPoolSize-connection-maxConnecting.json/threads_blocked_by_maxConnecting_check_out_minPoolSize_connections": "Test requires that connections established by minPoolSize are immediately used to satisfy check-out requests (DRIVERS-2225)",

	// GODRIVER-1826: The Go connection pool currently delivers any available
	// connection to the earliest waiting check-out request, independent of if
	// that check-out request already requested a new connection. That behavior
	// is currently incompatible with the "threads blocked by maxConnecting
	// check out returned connections" test, which expects that check-out
	// requests that request a new connection cannot be satisfied by a check-in.
	//
	// TODO(DRIVERS-2223): Re-enable this test once the spec test is updated to
	// support the Go pool check-in behavior.
	"TestCMAPSpec/pool-checkout-returned-connection-maxConnecting.json/threads_blocked_by_maxConnecting_check_out_returned_connections": "Test requires a checked-in connections cannot satisfy a check-out waiting on a new connection (DRIVERS-2223)",

	// TODO(GODRIVER-2129): Re-enable this test once GODRIVER-2129 is done.
	"TestAuthSpec/connection-string.json/must_raise_an_error_when_the_hostname_canonicalization_is_invalid": "Support will be added with GODRIVER-2129.",

	// GODRIVER-1773: This test runs a "find" with limit=4 and batchSize=3. It
	// expects batchSize values of three for the "find" and one for the
	// "getMore", but we send three for both.
	"TestUnifiedSpec/command-monitoring/find.json/A_successful_find_event_with_a_getmore_and_the_server_kills_the_cursor_(<=_4.4)":                               "See GODRIVER-1773",
	"TestUnifiedSpec/unified-test-format/valid-pass/poc-command-monitoring.json/A_successful_find_event_with_a_getmore_and_the_server_kills_the_cursor_(<=_4.4)": "See GODRIVER-1773",

	// GODRIVER-2577: The following spec tests require canceling ops
	// immediately, but the current logic clears pools and cancels in-progress
	// ops after two the heartbeat failures.
	"TestUnifiedSpec/server-discovery-and-monitoring/unified/interruptInUse-pool-clear.json/Connection_pool_clear_uses_interruptInUseConnections=true_after_monitor_timeout":                      "Go Driver clears after multiple timeout",
	"TestUnifiedSpec/server-discovery-and-monitoring/unified/interruptInUse-pool-clear.json/Error_returned_from_connection_pool_clear_with_interruptInUseConnections=true_is_retryable":           "Go Driver clears after multiple timeout",
	"TestUnifiedSpec/server-discovery-and-monitoring/unified/interruptInUse-pool-clear.json/Error_returned_from_connection_pool_clear_with_interruptInUseConnections=true_is_retryable_for_write": "Go Driver clears after multiple timeout",

	// TODO(GODRIVER-2843): Fix and unskip these test cases.
	"TestUnifiedSpec/sessions/snapshot-sessions.json/Find_operation_with_snapshot":                                      "Test fails frequently. See GODRIVER-2843",
	"TestUnifiedSpec/sessions/snapshot-sessions.json/Write_commands_with_snapshot_session_do_not_affect_snapshot_reads": "Test fails frequently. See GODRIVER-2843",

	// TODO(GODRIVER-3043): Avoid Appending Write/Read Concern in Atlas Search
	// Index Helper Commands.
	"TestUnifiedSpec/index-management/searchIndexIgnoresReadWriteConcern.json/dropSearchIndex_ignores_read_and_write_concern":       "Sync GODRIVER-3074, but skip testing bug GODRIVER-3043",
	"TestUnifiedSpec/index-management/searchIndexIgnoresReadWriteConcern.json/listSearchIndexes_ignores_read_and_write_concern":     "Sync GODRIVER-3074, but skip testing bug GODRIVER-3043",
	"TestUnifiedSpec/index-management/searchIndexIgnoresReadWriteConcern.json/updateSearchIndex_ignores_the_read_and_write_concern": "Sync GODRIVER-3074, but skip testing bug GODRIVER-3043",

	// TODO(DRIVERS-2829): Create CSOT Legacy Timeout Analogues and
	// Compatibility Field
	"TestUnifiedSpec/server-discovery-and-monitoring/unified/auth-network-timeout-error.json/Reset_server_and_pool_after_network_timeout_error_during_authentication": "Uses unsupported socketTimeoutMS",
	"TestUnifiedSpec/server-discovery-and-monitoring/unified/find-network-timeout-error.json/Ignore_network_timeout_error_on_find":                                    "Uses unsupported socketTimeoutMS",
	"TestUnifiedSpec/command-monitoring/find.json/A_successful_find_with_options":                                                                                     "Uses unsupported maxTimeMS",
	"TestUnifiedSpec/crud/unified/estimatedDocumentCount.json/estimatedDocumentCount_with_maxTimeMS":                                                                  "Uses unsupported maxTimeMS",
	"TestUnifiedSpec/run-command/runCursorCommand.json/supports_configuring_getMore_maxTimeMS":                                                                        "Uses unsupported maxTimeMS",

	// TODO(GODRIVER-3137): Implement Gossip cluster time"
	"TestUnifiedSpec/transactions/unified/mongos-unpin.json/unpin_after_TransientTransactionError_error_on_commit": "Implement GODRIVER-3137",

	// TODO(GODRIVER-3034): Drivers should unpin connections when ending a
	// session.
	"TestUnifiedSpec/transactions/unified/mongos-unpin.json/unpin_on_successful_abort":                                   "Implement GODRIVER-3034",
	"TestUnifiedSpec/transactions/unified/mongos-unpin.json/unpin_after_non-transient_error_on_abort":                    "Implement GODRIVER-3034",
	"TestUnifiedSpec/transactions/unified/mongos-unpin.json/unpin_after_TransientTransactionError_error_on_abort":        "Implement GODRIVER-3034",
	"TestUnifiedSpec/transactions/unified/mongos-unpin.json/unpin_when_a_new_transaction_is_started":                     "Implement GODRIVER-3034",
	"TestUnifiedSpec/transactions/unified/mongos-unpin.json/unpin_when_a_non-transaction_write_operation_uses_a_session": "Implement GODRIVER-3034",
	"TestUnifiedSpec/transactions/unified/mongos-unpin.json/unpin_when_a_non-transaction_read_operation_uses_a_session":  "Implement GODRIVER-3034",

	// DRIVERS-2722: Setting "maxTimeMS" on a command that creates a cursor also
	// limits the lifetime of the cursor. That may be surprising to users, so
	// omit "maxTimeMS" from operations that return user-managed cursors.
	"TestUnifiedSpec/client-side-operations-timeout/gridfs-find.json/timeoutMS_can_be_overridden_for_a_find":                                                          "maxTimeMS is disabled on find and aggregate. See DRIVERS-2722.",
	"TestUnifiedSpec/client-side-operations-timeout/override-operation-timeoutMS.json/timeoutMS_can_be_configured_for_an_operation_-_find_on_collection":              "maxTimeMS is disabled on find and aggregate. See DRIVERS-2722.",
	"TestUnifiedSpec/client-side-operations-timeout/override-operation-timeoutMS.json/timeoutMS_can_be_configured_for_an_operation_-_aggregate_on_collection":         "maxTimeMS is disabled on find and aggregate. See DRIVERS-2722.",
	"TestUnifiedSpec/client-side-operations-timeout/override-operation-timeoutMS.json/timeoutMS_can_be_configured_for_an_operation_-_aggregate_on_database":           "maxTimeMS is disabled on find and aggregate. See DRIVERS-2722.",
	"TestUnifiedSpec/client-side-operations-timeout/global-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoClient_-_find_on_collection":                          "maxTimeMS is disabled on find and aggregate. See DRIVERS-2722.",
	"TestUnifiedSpec/client-side-operations-timeout/global-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoClient_-_aggregate_on_collection":                     "maxTimeMS is disabled on find and aggregate. See DRIVERS-2722.",
	"TestUnifiedSpec/client-side-operations-timeout/global-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoClient_-_aggregate_on_database":                       "maxTimeMS is disabled on find and aggregate. See DRIVERS-2722.",
	"TestUnifiedSpec/client-side-operations-timeout/retryability-timeoutMS.json/operation_is_retried_multiple_times_for_non-zero_timeoutMS_-_find_on_collection":      "maxTimeMS is disabled on find and aggregate. See DRIVERS-2722.",
	"TestUnifiedSpec/client-side-operations-timeout/retryability-timeoutMS.json/operation_is_retried_multiple_times_for_non-zero_timeoutMS_-_aggregate_on_collection": "maxTimeMS is disabled on find and aggregate. See DRIVERS-2722.",
	"TestUnifiedSpec/client-side-operations-timeout/retryability-timeoutMS.json/operation_is_retried_multiple_times_for_non-zero_timeoutMS_-_aggregate_on_database":   "maxTimeMS is disabled on find and aggregate. See DRIVERS-2722.",
	"TestUnifiedSpec/client-side-operations-timeout/gridfs-find.json/timeoutMS_applied_to_find_command":                                                               "maxTimeMS is disabled on find and aggregate. See DRIVERS-2722.",
	"TestUnifiedSpec/client-side-operations-timeout/tailable-awaitData.json/timeoutMS_applied_to_find":                                                                "maxTimeMS is disabled on find and aggregate. See DRIVERS-2722.",
	"TestUnifiedSpec/client-side-operations-timeout/tailable-awaitData.json/timeoutMS_is_refreshed_for_getMore_if_maxAwaitTimeMS_is_not_set":                          "maxTimeMS is disabled on find and aggregate. See DRIVERS-2722.",
	"TestUnifiedSpec/client-side-operations-timeout/tailable-awaitData.json/timeoutMS_is_refreshed_for_getMore_if_maxAwaitTimeMS_is_set":                              "maxTimeMS is disabled on find and aggregate. See DRIVERS-2722.",
	"TestUnifiedSpec/client-side-operations-timeout/tailable-awaitData.json/timeoutMS_is_refreshed_for_getMore_-_failure":                                             "maxTimeMS is disabled on find and aggregate. See DRIVERS-2722.",

	// DRIVERS-2953: This test requires that the driver sends a "getMore" with
	// "maxTimeMS" set. However, "getMore" can only include "maxTimeMS" for
	// tailable awaitData cursors. Including "maxTimeMS" on "getMore" for any
	// other cursor type results in a server error:
	//
	//  (BadValue) cannot set maxTimeMS on getMore command for a non-awaitData cursor
	//
	"TestUnifiedSpec/client-side-operations-timeout/runCursorCommand.json/Non-tailable_cursor_lifetime_remaining_timeoutMS_applied_to_getMore_if_timeoutMode_is_unset": "maxTimeMS can't be set on a getMore. See DRIVERS-2953",

	// TODO(GODRIVER-2466): Converting SDAM integration spec tests to unified
	// test format requires implementing new test operations, such as
	// "recordTopologyDescription". Un-skip whenever GODRIVER-2466 is completed.
	"TestUnifiedSpec/server-discovery-and-monitoring/unified/rediscover-quickly-after-step-down.json/Rediscover_quickly_after_replSetStepDown": "Implement GODRIVER-2466",

	// TODO(GODRIVER-2967): The Go Driver doesn't currently emit a
	// TopologyChangedEvent when a topology is closed. Un-skip whenever
	// GODRIVER-2967 is completed.
	"TestUnifiedSpec/server-discovery-and-monitoring/unified/logging-loadbalanced.json/Topology_lifecycle":                            "Implement GODRIVER-2967",
	"TestUnifiedSpec/server-discovery-and-monitoring/unified/logging-sharded.json/Topology_lifecycle":                                 "Implement GODRIVER-2967",
	"TestUnifiedSpec/server-discovery-and-monitoring/unified/logging-replicaset.json/Topology_lifecycle":                              "Implement GODRIVER-2967",
	"TestUnifiedSpec/server-discovery-and-monitoring/unified/logging-standalone.json/Topology_lifecycle":                              "Implement GODRIVER-2967",
	"TestUnifiedSpec/server-discovery-and-monitoring/unified/loadbalanced-emit-topology-changed-before-close.json/Topology_lifecycle": "Implement GODRIVER-2967",
	"TestUnifiedSpec/server-discovery-and-monitoring/unified/sharded-emit-topology-changed-before-close.json/Topology_lifecycle":      "Implement GODRIVER-2967",
	"TestUnifiedSpec/server-discovery-and-monitoring/unified/replicaset-emit-topology-changed-before-close.json/Topology_lifecycle":   "Implement GODRIVER-2967",
	"TestUnifiedSpec/server-discovery-and-monitoring/unified/standalone-emit-topology-changed-before-close.json/Topology_lifecycle":   "Implement GODRIVER-2967",

	// GODRIVER-3473: the implementation of DRIVERS-2868 makes it clear that the
	// Go Driver does not correctly implement the following validation for
	// tailable awaitData cursors:
	//
	//     Drivers MUST error if this option is set, timeoutMS is set to a
	//     non-zero value, and maxAwaitTimeMS is greater than or equal to
	//     timeoutMS.
	//
	// Once GODRIVER-3473 is completed, we can continue running these tests.
	"TestUnifiedSpec/client-side-operations-timeout/tailable-awaitData.json/apply_remaining_timeoutMS_if_less_than_maxAwaitTimeMS": "Go Driver does not implement this behavior. See GODRIVER-3473",
	"TestUnifiedSpec/client-side-operations-timeout/tailable-awaitData.json/error_if_maxAwaitTimeMS_is_equal_to_timeoutMS":         "Go Driver does not implement this behavior. See GODRIVER-3473",
}

// CheckSkip checks if the fully-qualified test name matches a skipped test
// name. If the test name matches, the reason is logged and the test is skipped.
func CheckSkip(t *testing.T) {
	if reason := skipTests[t.Name()]; reason != "" {
		t.Skipf("Skipping due to known failure: %q", reason)
	}
}
