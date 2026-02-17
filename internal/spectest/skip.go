// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package spectest

import "testing"

// skipTests is a map of "fully-qualified test name" to "the reason for skipping
// the test".
var skipTests = map[string][]string{
	// SPEC-1403: This test checks to see if the correct error is thrown when auto
	// encrypting with a server < 4.2. Currently, the test will fail because a
	// server < 4.2 wouldn't have mongocryptd, so Client construction would fail
	// with a mongocryptd spawn error.
	"Servers less than 4.2 do not have mongocryptd; see SPEC-1403": {
		"TestClientSideEncryptionSpec/maxWireVersion.json/operation_fails_with_maxWireVersion_<_8",
	},

	// TODO(GODRIVER-1826): Tests for incompatible event ordering in load-balancer
	// SDAM spec tests.
	"Event ordering is incompatible with load-balancer SDAM spec test (DRIVERS-1785)": {
		"TestCMAPSpec/pool-create-min-size-error.json/error_during_minPoolSize_population_clears_pool",
	},

	// TODO(GODRIVER-1826): Race condition prevents the "threads blocked by
	// maxConnecting" test from passing.
	"Test requires that connections established by minPoolSize are immediately used to satisfy check-out requests (DRIVERS-2225)": {
		"TestCMAPSpec/pool-checkout-minPoolSize-connection-maxConnecting.json/threads_blocked_by_maxConnecting_check_out_minPoolSize_connections",
	},

	// TODO(GODRIVER-1826): The Go connection pool behavior for check-in requests
	// is incompatible with expected test behavior.
	"Test requires a checked-in connections cannot satisfy a check-out waiting on a new connection (DRIVERS-2223)": {
		"TestCMAPSpec/pool-checkout-returned-connection-maxConnecting.json/threads_blocked_by_maxConnecting_check_out_returned_connections",
	},

	// TODO(GODRIVER-2129): Re-enable this test once the feature is implemented.
	// TODO(GODRIVER-2129): Support for
	// sspiHostnamecanonicalization=none/forward/forwardAndReverse for Kerberos
	"Requires sspiHostnamecanonicalization=none/forward/forwardAndReverse support for Kerberos (GODRIVER-2129)": {
		"TestAuthSpec/connection-string.json/must_raise_an_error_when_the_hostname_canonicalization_is_invalid",
	},

	// TODO(GODRIVER-3614): Remove support for specifying MONGODB-AWS authentication properties explicitly
	"Should throw an exception if username provided (MONGODB-AWS) (GODRIVER-3614)": {
		"TestAuthSpec/connection-string.json/should_throw_an_exception_if_username_and_password_provided_(MONGODB-AWS)",
	},

	// TODO(GODRIVER-2183): Implementation of Socks5 Proxy Support is pending.
	"Requires Socks5 Proxy Support (GODRIVER-2183)": {
		"TestURIOptionsSpec/proxy-options.json/proxyPort_without_proxyHost",
		"TestURIOptionsSpec/proxy-options.json/proxyUsername_without_proxyHost",
		"TestURIOptionsSpec/proxy-options.json/proxyPassword_without_proxyHost",
		"TestURIOptionsSpec/proxy-options.json/all_other_proxy_options_without_proxyHost",
		"TestURIOptionsSpec/proxy-options.json/proxyUsername_without_proxyPassword",
		"TestURIOptionsSpec/proxy-options.json/proxyPassword_without_proxyUsername",
		"TestURIOptionsSpec/proxy-options.json/multiple_proxyHost_parameters",
		"TestURIOptionsSpec/proxy-options.json/multiple_proxyPort_parameters",
		"TestURIOptionsSpec/proxy-options.json/multiple_proxyUsername_parameters",
		"TestURIOptionsSpec/proxy-options.json/multiple_proxyPassword_parameters",
	},

	// The wtimeoutMS option for write concern is deprecated.
	"wtimeoutMS is deprecated": {
		"TestURIOptionsSpec/concern-options.json/Valid_read_and_write_concern_are_parsed_correctly",
		"TestURIOptionsSpec/concern-options.json/Non-numeric_wTimeoutMS_causes_a_warning",
		"TestURIOptionsSpec/concern-options.json/Too_low_wTimeoutMS_causes_a_warning",
		"TestReadWriteConcernSpec/connstring/write-concern.json/wtimeoutMS_as_an_invalid_number",
	},

	// Unsupported TLS behavior in connection strings.
	"unsupported connstring behavior": {
		"TestURIOptionsSpec/tls-options.json/tlsAllowInvalidCertificates_and_tlsDisableCertificateRevocationCheck_both_present_(and_true)_raises_an_error",
		"TestURIOptionsSpec/tls-options.json/tlsAllowInvalidCertificates=true_and_tlsDisableCertificateRevocationCheck=false_raises_an_error",
		"TestURIOptionsSpec/tls-options.json/tlsAllowInvalidCertificates=false_and_tlsDisableCertificateRevocationCheck=true_raises_an_error",
		"TestURIOptionsSpec/tls-options.json/tlsAllowInvalidCertificates_and_tlsDisableCertificateRevocationCheck_both_present_(and_false)_raises_an_error",
		"TestURIOptionsSpec/tls-options.json/tlsDisableCertificateRevocationCheck_and_tlsAllowInvalidCertificates_both_present_(and_true)_raises_an_error",
		"TestURIOptionsSpec/tls-options.json/tlsDisableCertificateRevocationCheck=true_and_tlsAllowInvalidCertificates=false_raises_an_error",
		"TestURIOptionsSpec/tls-options.json/tlsDisableCertificateRevocationCheck=false_and_tlsAllowInvalidCertificates=true_raises_an_error",
		"TestURIOptionsSpec/tls-options.json/tlsDisableCertificateRevocationCheck_and_tlsAllowInvalidCertificates_both_present_(and_false)_raises_an_error",
		"TestURIOptionsSpec/tls-options.json/tlsInsecure_and_tlsDisableCertificateRevocationCheck_both_present_(and_true)_raises_an_error",
		"TestURIOptionsSpec/tls-options.json/tlsInsecure=true_and_tlsDisableCertificateRevocationCheck=false_raises_an_error",
		"TestURIOptionsSpec/tls-options.json/tlsInsecure=false_and_tlsDisableCertificateRevocationCheck=true_raises_an_error",
		"TestURIOptionsSpec/tls-options.json/tlsInsecure_and_tlsDisableCertificateRevocationCheck_both_present_(and_false)_raises_an_error",
		"TestURIOptionsSpec/tls-options.json/tlsDisableCertificateRevocationCheck_and_tlsInsecure_both_present_(and_true)_raises_an_error",
		"TestURIOptionsSpec/tls-options.json/tlsDisableCertificateRevocationCheck=true_and_tlsInsecure=false_raises_an_error",
		"TestURIOptionsSpec/tls-options.json/tlsDisableCertificateRevocationCheck=false_and_tlsInsecure=true_raises_an_error",
		"TestURIOptionsSpec/tls-options.json/tlsDisableCertificateRevocationCheck_and_tlsInsecure_both_present_(and_false)_raises_an_error",
		"TestURIOptionsSpec/tls-options.json/tlsDisableCertificateRevocationCheck_and_tlsDisableOCSPEndpointCheck_both_present_(and_true)_raises_an_error",
		"TestURIOptionsSpec/tls-options.json/tlsDisableCertificateRevocationCheck=true_and_tlsDisableOCSPEndpointCheck=false_raises_an_error",
		"TestURIOptionsSpec/tls-options.json/tlsDisableCertificateRevocationCheck=false_and_tlsDisableOCSPEndpointCheck=true_raises_an_error",
		"TestURIOptionsSpec/tls-options.json/tlsDisableCertificateRevocationCheck_and_tlsDisableOCSPEndpointCheck_both_present_(and_false)_raises_an_error",
		"TestURIOptionsSpec/tls-options.json/tlsDisableOCSPEndpointCheck_and_tlsDisableCertificateRevocationCheck_both_present_(and_true)_raises_an_error",
		"TestURIOptionsSpec/tls-options.json/tlsDisableOCSPEndpointCheck=true_and_tlsDisableCertificateRevocationCheck=false_raises_an_error",
		"TestURIOptionsSpec/tls-options.json/tlsDisableOCSPEndpointCheck=false_and_tlsDisableCertificateRevocationCheck=true_raises_an_error",
		"TestURIOptionsSpec/tls-options.json/tlsDisableOCSPEndpointCheck_and_tlsDisableCertificateRevocationCheck_both_present_(and_false)_raises_an_error",
		"TestURIOptionsSpec/tls-options.json/tlsAllowInvalidCertificates_and_tlsDisableOCSPEndpointCheck_both_present_(and_true)_raises_an_error",
		"TestURIOptionsSpec/tls-options.json/tlsAllowInvalidCertificates=true_and_tlsDisableOCSPEndpointCheck=false_raises_an_error",
		"TestURIOptionsSpec/tls-options.json/tlsAllowInvalidCertificates=false_and_tlsDisableOCSPEndpointCheck=true_raises_an_error",
		"TestURIOptionsSpec/tls-options.json/tlsAllowInvalidCertificates_and_tlsDisableOCSPEndpointCheck_both_present_(and_false)_raises_an_error",
		"TestURIOptionsSpec/tls-options.json/tlsDisableOCSPEndpointCheck_and_tlsAllowInvalidCertificates_both_present_(and_true)_raises_an_error",
		"TestURIOptionsSpec/tls-options.json/tlsDisableOCSPEndpointCheck=true_and_tlsAllowInvalidCertificates=false_raises_an_error",
		"TestURIOptionsSpec/tls-options.json/tlsDisableOCSPEndpointCheck=false_and_tlsAllowInvalidCertificates=true_raises_an_error",
		"TestURIOptionsSpec/tls-options.json/tlsDisableOCSPEndpointCheck_and_tlsAllowInvalidCertificates_both_present_(and_false)_raises_an_error",
		"TestURIOptionsSpec/tls-options.json/Invalid_tlsAllowInvalidCertificates_causes_a_warning",
		"TestURIOptionsSpec/tls-options.json/tlsAllowInvalidCertificates_is_parsed_correctly",
		"TestURIOptionsSpec/tls-options.json/tlsAllowInvalidHostnames_is_parsed_correctly",
		"TestURIOptionsSpec/tls-options.json/Invalid_tlsAllowInvalidHostnames_causes_a_warning",
		"TestURIOptionsSpec/tls-options.json/tlsInsecure_and_tlsAllowInvalidCertificates_both_present_(and_true)_raises_an_error",
		"TestURIOptionsSpec/tls-options.json/tlsInsecure_and_tlsAllowInvalidCertificates_both_present_(and_false)_raises_an_error",
		"TestURIOptionsSpec/tls-options.json/tlsAllowInvalidCertificates_and_tlsInsecure_both_present_(and_true)_raises_an_error",
		"TestURIOptionsSpec/tls-options.json/tlsAllowInvalidCertificates_and_tlsInsecure_both_present_(and_false)_raises_an_error",
		"TestURIOptionsSpec/tls-options.json/tlsInsecure_and_tlsAllowInvalidHostnames_both_present_(and_true)_raises_an_error",
		"TestURIOptionsSpec/tls-options.json/tlsInsecure_and_tlsAllowInvalidHostnames_both_present_(and_false)_raises_an_error",
		"TestURIOptionsSpec/tls-options.json/tlsAllowInvalidHostnames_and_tlsInsecure_both_present_(and_true)_raises_an_error",
		"TestURIOptionsSpec/tls-options.json/tlsAllowInvalidHostnames_and_tlsInsecure_both_present_(and_false)_raises_an_error",
	},

	// TODO(GODRIVER-2991): Make delimiting slash between hosts and options
	// optional.
	"Requires making delimiting slash between hosts and options optional (GODRIVERS-2991)": {
		"TestConnStringSpec/valid-options.json/Missing_delimiting_slash_between_hosts_and_options",
	},

	// Connstring tests violate current Go Driver behavior.
	"unsupported behavior": {
		"TestURIOptionsSpec/connection-pool-options.json/maxConnecting=0_causes_a_warning",
		"TestURIOptionsSpec/single-threaded-options.json/Invalid_serverSelectionTryOnce_causes_a_warning",
		"TestConnStringSpec/valid-warnings.json/Empty_integer_option_values_are_ignored",
		"TestConnStringSpec/valid-warnings.json/Empty_boolean_option_value_are_ignored",
		"TestConnStringSpec/valid-warnings.json/Comma_in_a_key_value_pair_causes_a_warning",
	},

	// TODO(GODRIVER-3167): Support assertions on topologyDescriptionChangedEvent
	// in expectEvents.
	"Support assertions on topologyDescriptionChangedEvent in expectEvents (GODRIVER-3167)": {
		"TestUnifiedSpec/unified-test-format/tests/valid-pass/expectedEventsForClient-topologyDescriptionChangedEvent.json/can_assert_on_values_of_newDescription_and_previousDescription_fields",
	},

	// TODO(GODRIVER-3409): Add regression test for "number" alias in $$type
	// operator.
	"Regression test for 'number' alias in $$type operator (GODRIVER-3409)": {
		"TestUnifiedSpec/unified-test-format/tests/valid-pass/operator-type-number_alias.json/type_number_alias_matches_int32",
		"TestUnifiedSpec/unified-test-format/tests/valid-pass/operator-type-number_alias.json/type_number_alias_matches_int64",
		"TestUnifiedSpec/unified-test-format/tests/valid-pass/operator-type-number_alias.json/type_number_alias_matches_double",
		"TestUnifiedSpec/unified-test-format/tests/valid-pass/operator-type-number_alias.json/type_number_alias_matches_decimal128",
	},

	// TODO(GODRIVER-3143): Convert CRUD v1 spec tests to unified test format.
	"Convert CRUD v1 spec tests to unified test format (GODRIVER-3143)": {
		"TestUnifiedSpec/crud/tests/unified/bulkWrite-collation.json/BulkWrite_with_delete_operations_and_collation",
		"TestUnifiedSpec/crud/tests/unified/count.json/Count_documents_with_skip_and_limit",
		"TestUnifiedSpec/crud/tests/unified/findOne.json/FindOne_with_filter,_sort,_and_skip",
	},

	// TODO(GODRIVER-2125): Allow hint for unacknowledged writes using OP_MSG when
	// supported by the server.
	"Allow hint for unacknowledged writes using OP_MSG (GODRIVER-2125)": {
		"TestUnifiedSpec/crud/tests/unified/bulkWrite-deleteMany-hint-unacknowledged.json/Unacknowledged_deleteMany_with_hint_string_on_4.4+_server",
		"TestUnifiedSpec/crud/tests/unified/bulkWrite-deleteMany-hint-unacknowledged.json/Unacknowledged_deleteMany_with_hint_document_on_4.4+_server",
		"TestUnifiedSpec/crud/tests/unified/bulkWrite-deleteOne-hint-unacknowledged.json/Unacknowledged_deleteOne_with_hint_string_on_4.4+_server",
		"TestUnifiedSpec/crud/tests/unified/bulkWrite-deleteOne-hint-unacknowledged.json/Unacknowledged_deleteOne_with_hint_document_on_4.4+_server",
		"TestUnifiedSpec/crud/tests/unified/bulkWrite-replaceOne-hint-unacknowledged.json/Unacknowledged_replaceOne_with_hint_string_on_4.2+_server",
		"TestUnifiedSpec/crud/tests/unified/bulkWrite-replaceOne-hint-unacknowledged.json/Unacknowledged_replaceOne_with_hint_document_on_4.2+_server",
		"TestUnifiedSpec/crud/tests/unified/bulkWrite-updateMany-hint-unacknowledged.json/Unacknowledged_updateMany_with_hint_string_on_4.2+_server",
		"TestUnifiedSpec/crud/tests/unified/bulkWrite-updateMany-hint-unacknowledged.json/Unacknowledged_updateMany_with_hint_document_on_4.2+_server",
		"TestUnifiedSpec/crud/tests/unified/bulkWrite-updateOne-hint-unacknowledged.json/Unacknowledged_updateOne_with_hint_string_on_4.2+_server",
		"TestUnifiedSpec/crud/tests/unified/bulkWrite-updateOne-hint-unacknowledged.json/Unacknowledged_updateOne_with_hint_document_on_4.2+_server",
	},

	// TODO(GODRIVER-3407): Allow drivers to set bypassDocumentValidation: false
	// on write commands.
	"Allow drivers to set bypassDocumentValidation: false on write commands (GODRIVER-3407)": {
		"TestUnifiedSpec/crud/tests/unified/bypassDocumentValidation.json/Aggregate_with_$out_passes_bypassDocumentValidation:_false",
		"TestUnifiedSpec/crud/tests/unified/bypassDocumentValidation.json/BulkWrite_passes_bypassDocumentValidation:_false",
		"TestUnifiedSpec/crud/tests/unified/bypassDocumentValidation.json/FindOneAndReplace_passes_bypassDocumentValidation:_false",
		"TestUnifiedSpec/crud/tests/unified/bypassDocumentValidation.json/FindOneAndUpdate_passes_bypassDocumentValidation:_fals",
		"TestUnifiedSpec/crud/tests/unified/bypassDocumentValidation.json/FindOneAndUpdate_passes_bypassDocumentValidation:_false",
		"TestUnifiedSpec/crud/tests/unified/bypassDocumentValidation.json/InsertMany_passes_bypassDocumentValidation:_false",
		"TestUnifiedSpec/crud/tests/unified/bypassDocumentValidation.json/InsertOne_passes_bypassDocumentValidation:_false",
		"TestUnifiedSpec/crud/tests/unified/bypassDocumentValidation.json/ReplaceOne_passes_bypassDocumentValidation:_false",
		"TestUnifiedSpec/crud/tests/unified/bypassDocumentValidation.json/UpdateMany_passes_bypassDocumentValidation:_false",
		"TestUnifiedSpec/crud/tests/unified/bypassDocumentValidation.json/UpdateOne_passes_bypassDocumentValidation:_false",
		"TestUnifiedSpec/crud/tests/unified/deleteMany-hint-unacknowledged.json/Unacknowledged_deleteMany_with_hint_string_on_4.4+_server",
		"TestUnifiedSpec/crud/tests/unified/deleteMany-hint-unacknowledged.json/Unacknowledged_deleteMany_with_hint_document_on_4.4+_server",
		"TestUnifiedSpec/crud/tests/unified/deleteOne-hint-unacknowledged.json/Unacknowledged_deleteOne_with_hint_string_on_4.4+_server",
		"TestUnifiedSpec/crud/tests/unified/deleteOne-hint-unacknowledged.json/Unacknowledged_deleteOne_with_hint_document_on_4.4+_server",
		"TestUnifiedSpec/crud/tests/unified/findOneAndDelete-hint-unacknowledged.json/Unacknowledged_findOneAndDelete_with_hint_string_on_4.4+_server",
		"TestUnifiedSpec/crud/tests/unified/findOneAndDelete-hint-unacknowledged.json/Unacknowledged_findOneAndDelete_with_hint_document_on_4.4+_server",
		"TestUnifiedSpec/crud/tests/unified/findOneAndReplace-hint-unacknowledged.json/Unacknowledged_findOneAndReplace_with_hint_string_on_4.4+_server",
		"TestUnifiedSpec/crud/tests/unified/findOneAndReplace-hint-unacknowledged.json/Unacknowledged_findOneAndReplace_with_hint_document_on_4.4+_server",
		"TestUnifiedSpec/crud/tests/unified/findOneAndUpdate-hint-unacknowledged.json/Unacknowledged_findOneAndUpdate_with_hint_string_on_4.4+_server",
		"TestUnifiedSpec/crud/tests/unified/findOneAndUpdate-hint-unacknowledged.json/Unacknowledged_findOneAndUpdate_with_hint_document_on_4.4+_server",
		"TestUnifiedSpec/crud/tests/unified/replaceOne-hint-unacknowledged.json/Unacknowledged_replaceOne_with_hint_string_on_4.2+_server",
		"TestUnifiedSpec/crud/tests/unified/replaceOne-hint-unacknowledged.json/Unacknowledged_replaceOne_with_hint_document_on_4.2+_server",
		"TestUnifiedSpec/crud/tests/unified/updateMany-hint-unacknowledged.json/Unacknowledged_updateMany_with_hint_string_on_4.2+_server",
		"TestUnifiedSpec/crud/tests/unified/updateMany-hint-unacknowledged.json/Unacknowledged_updateMany_with_hint_document_on_4.2+_server",
		"TestUnifiedSpec/crud/tests/unified/updateOne-hint-unacknowledged.json/Unacknowledged_updateOne_with_hint_string_on_4.2+_server",
		"TestUnifiedSpec/crud/tests/unified/updateOne-hint-unacknowledged.json/Unacknowledged_updateOne_with_hint_document_on_4.2+_server",
	},

	// TODO(GODRIVER-3392): Test that inserts and upserts respect null _id values.
	"Test inserts and upserts respect null _id values (GODRIVER-3392)": {
		"TestUnifiedSpec/crud/tests/unified/create-null-ids.json/inserting__id_with_type_null_via_insertOne",
		"TestUnifiedSpec/crud/tests/unified/create-null-ids.json/inserting__id_with_type_null_via_clientBulkWrite",
	},

	// TODO(GODRIVER-3395): Ensure findOne does not set batchSize=1.
	"Ensure findOne does not set batchSize=1 (GODRIVER-3395)": {
		"TestUnifiedSpec/crud/tests/unified/find.json/Find_with_batchSize_equal_to_limit",
	},

	// TODO(GODRIVER-2016): Convert transactions spec tests to unified test
	// format.
	"Convert transactions spec tests to unified test format (GODRIVER-2016)": {
		"TestUnifiedSpec/transactions-convenient-api/tests/unified/callback-retry.json/callback_is_not_retried_after_non-transient_error_(DuplicateKeyError)",
		"TestUnifiedSpec/transactions-convenient-api/tests/unified/callback-retry.json/callback_succeeds_after_multiple_connection_errors",
		"TestUnifiedSpec/transactions-convenient-api/tests/unified/commit-retry.json/commitTransaction_retry_only_overwrites_write_concern_w_option",
		"TestUnifiedSpec/transactions-convenient-api/tests/unified/commit-retry.json/commit_is_not_retried_after_MaxTimeMSExpired_error",
		"TestUnifiedSpec/transactions-convenient-api/tests/unified/commit-writeconcernerror.json/commitTransaction_is_not_retried_after_UnknownReplWriteConcern_error",
		"TestUnifiedSpec/transactions-convenient-api/tests/unified/commit-writeconcernerror.json/commitTransaction_is_not_retried_after_UnsatisfiableWriteConcern_error",
		"TestUnifiedSpec/transactions-convenient-api/tests/unified/commit-writeconcernerror.json/commitTransaction_is_not_retried_after_MaxTimeMSExpired_error",
	},

	// TODO(GODRIVER-1773): Tests related to batch size expectation in "find" and
	// "getMore" events.
	"Tests for batch size expectation in 'find' and 'getMore' events (GODRIVER-1773)": {
		"TestUnifiedSpec/command-logging-and-monitoring/tests/monitoring/find.json/A_successful_find_event_with_a_getmore_and_the_server_kills_the_cursor_(<=_4.4)",
		"TestUnifiedSpec/unified-test-format/tests/valid-pass/poc-command-monitoring.json/A_successful_find_event_with_a_getmore_and_the_server_kills_the_cursor_(<=_4.4)",
	},

	// TODO(GODRIVER-2577): Tests require immediate operation canceling,
	// incompatible with current pool clearing logic.
	"Require immediate operation canceling for pool clearing (GODRIVER-2577)": {
		"TestUnifiedSpec/server-discovery-and-monitoring/tests/unified/interruptInUse-pool-clear.json/Connection_pool_clear_uses_interruptInUseConnections=true_after_monitor_timeout",
		"TestUnifiedSpec/server-discovery-and-monitoring/tests/unified/interruptInUse-pool-clear.json/Error_returned_from_connection_pool_clear_with_interruptInUseConnections=true_is_retryable",
		"TestUnifiedSpec/server-discovery-and-monitoring/tests/unified/interruptInUse-pool-clear.json/Error_returned_from_connection_pool_clear_with_interruptInUseConnections=true_is_retryable_for_write",
	},

	// TODO(GODRIVER-3043): Avoid Appending Write/Read Concern in Atlas Search
	// Index Helper Commands.
	"Sync tests but avoid write/read concern bug (GODRIVER-3043)": {
		"TestUnifiedSpec/index-management/tests/searchIndexIgnoresReadWriteConcern.json/dropSearchIndex_ignores_read_and_write_concern",
		"TestUnifiedSpec/index-management/tests/searchIndexIgnoresReadWriteConcern.json/listSearchIndexes_ignores_read_and_write_concern",
		"TestUnifiedSpec/index-management/tests/searchIndexIgnoresReadWriteConcern.json/updateSearchIndex_ignores_the_read_and_write_concern",
	},

	// TODO(DRIVERS-2829): Create CSOT Legacy Timeout Analogues and Compatibility
	// Field.
	"Handles socketTimeoutMS instead of CSOT (DRIVERS-2829)": {
		"TestUnifiedSpec/server-discovery-and-monitoring/tests/unified/auth-network-timeout-error.json/Reset_server_and_pool_after_network_timeout_error_during_authentication",
		"TestUnifiedSpec/server-discovery-and-monitoring/tests/unified/find-network-timeout-error.json/Ignore_network_timeout_error_on_find",
		"TestUnifiedSpec/command-logging-and-monitoring/tests/monitoring/find.json/A_successful_find_with_options",
		"TestUnifiedSpec/crud/tests/unified/estimatedDocumentCount.json/estimatedDocumentCount_with_maxTimeMS",
		"TestUnifiedSpec/run-command/tests/unified/runCursorCommand.json/supports_configuring_getMore_maxTimeMS",
	},

	// TODO(GODRIVER-3034): Drivers should unpin connections when ending a
	// session.
	"Unpin connections at session end (GODRIVER-3034)": {
		"TestUnifiedSpec/transactions/unified/mongos-unpin.json/unpin_on_successful_abort",
		"TestUnifiedSpec/transactions/unified/mongos-unpin.json/unpin_after_non-transient_error_on_abort",
		"TestUnifiedSpec/transactions/unified/mongos-unpin.json/unpin_after_TransientTransactionError_error_on_abort",
		"TestUnifiedSpec/transactions/unified/mongos-unpin.json/unpin_when_a_new_transaction_is_started",
		"TestUnifiedSpec/transactions/unified/mongos-unpin.json/unpin_when_a_non-transaction_write_operation_uses_a_session",
		"TestUnifiedSpec/transactions/unified/mongos-unpin.json/unpin_when_a_non-transaction_read_operation_uses_a_session",
	},

	// TODO(GODRIVER-3146): Convert retryable reads spec tests to unified test format.
	"Convert retryable reads spec tests to unified test format (GODRIVER-3146)": {
		"TestUnifiedSpec/retryable-reads/tests/unified/listCollectionObjects-serverErrors.json/ListCollectionObjects_succeeds_after_InterruptedAtShutdown",
		"TestUnifiedSpec/retryable-reads/tests/unified/listCollectionObjects-serverErrors.json/ListCollectionObjects_succeeds_after_InterruptedDueToReplStateChange",
		"TestUnifiedSpec/retryable-reads/tests/unified/listCollectionObjects-serverErrors.json/ListCollectionObjects_succeeds_after_NotWritablePrimary",
		"TestUnifiedSpec/retryable-reads/tests/unified/listCollectionObjects-serverErrors.json/ListCollectionObjects_succeeds_after_NotPrimaryNoSecondaryOk",
		"TestUnifiedSpec/retryable-reads/tests/unified/listCollectionObjects-serverErrors.json/ListCollectionObjects_succeeds_after_NotPrimaryOrSecondary",
		"TestUnifiedSpec/retryable-reads/tests/unified/listCollectionObjects-serverErrors.json/ListCollectionObjects_succeeds_after_PrimarySteppedDown",
		"TestUnifiedSpec/retryable-reads/tests/unified/listCollectionObjects-serverErrors.json/ListCollectionObjects_succeeds_after_ShutdownInProgress",
		"TestUnifiedSpec/retryable-reads/tests/unified/listCollectionObjects-serverErrors.json/ListCollectionObjects_succeeds_after_HostNotFound",
		"TestUnifiedSpec/retryable-reads/tests/unified/listCollectionObjects-serverErrors.json/ListCollectionObjects_succeeds_after_HostUnreachable",
		"TestUnifiedSpec/retryable-reads/tests/unified/listCollectionObjects-serverErrors.json/ListCollectionObjects_succeeds_after_NetworkTimeout",
		"TestUnifiedSpec/retryable-reads/tests/unified/listCollectionObjects-serverErrors.json/ListCollectionObjects_succeeds_after_SocketException",
		"TestUnifiedSpec/retryable-reads/tests/unified/listCollectionObjects-serverErrors.json/ListCollectionObjects_fails_after_two_NotWritablePrimary_errors",
		"TestUnifiedSpec/retryable-reads/tests/unified/listCollectionObjects-serverErrors.json/ListCollectionObjects_fails_after_NotWritablePrimary_when_retryReads_is_false",
		"TestUnifiedSpec/retryable-reads/tests/unified/listCollectionObjects.json/ListCollectionObjects_succeeds_on_first_attempt",
		"TestUnifiedSpec/retryable-reads/tests/unified/listCollectionObjects.json/ListCollectionObjects_succeeds_on_second_attempt",
		"TestUnifiedSpec/retryable-reads/tests/unified/listCollectionObjects.json/ListCollectionObjects_fails_on_first_attempt",
		"TestUnifiedSpec/retryable-reads/tests/unified/listCollectionObjects.json/ListCollectionObjects_fails_on_second_attempt",
		"TestUnifiedSpec/retryable-reads/tests/unified/listDatabaseObjects-serverErrors.json/ListDatabaseObjects_succeeds_after_InterruptedAtShutdown",
		"TestUnifiedSpec/retryable-reads/tests/unified/listDatabaseObjects-serverErrors.json/ListDatabaseObjects_succeeds_after_InterruptedDueToReplStateChange",
		"TestUnifiedSpec/retryable-reads/tests/unified/listDatabaseObjects-serverErrors.json/ListDatabaseObjects_succeeds_after_NotWritablePrimary",
		"TestUnifiedSpec/retryable-reads/tests/unified/listDatabaseObjects-serverErrors.json/ListDatabaseObjects_succeeds_after_NotPrimaryNoSecondaryOk",
		"TestUnifiedSpec/retryable-reads/tests/unified/listDatabaseObjects-serverErrors.json/ListDatabaseObjects_succeeds_after_NotPrimaryOrSecondary",
		"TestUnifiedSpec/retryable-reads/tests/unified/listDatabaseObjects-serverErrors.json/ListDatabaseObjects_succeeds_after_PrimarySteppedDown",
		"TestUnifiedSpec/retryable-reads/tests/unified/listDatabaseObjects-serverErrors.json/ListDatabaseObjects_succeeds_after_ShutdownInProgress",
		"TestUnifiedSpec/retryable-reads/tests/unified/listDatabaseObjects-serverErrors.json/ListDatabaseObjects_succeeds_after_HostNotFound",
		"TestUnifiedSpec/retryable-reads/tests/unified/listDatabaseObjects-serverErrors.json/ListDatabaseObjects_succeeds_after_HostUnreachable",
		"TestUnifiedSpec/retryable-reads/tests/unified/listDatabaseObjects-serverErrors.json/ListDatabaseObjects_succeeds_after_NetworkTimeout",
		"TestUnifiedSpec/retryable-reads/tests/unified/listDatabaseObjects-serverErrors.json/ListDatabaseObjects_succeeds_after_SocketException",
		"TestUnifiedSpec/retryable-reads/tests/unified/listDatabaseObjects-serverErrors.json/ListDatabaseObjects_fails_after_two_NotWritablePrimary_errors",
		"TestUnifiedSpec/retryable-reads/tests/unified/listDatabaseObjects-serverErrors.json/ListDatabaseObjects_fails_after_NotWritablePrimary_when_retryReads_is_false",
		"TestUnifiedSpec/retryable-reads/tests/unified/listDatabaseObjects.json/ListDatabaseObjects_succeeds_on_first_attempt",
		"TestUnifiedSpec/retryable-reads/tests/unified/listDatabaseObjects.json/ListDatabaseObjects_succeeds_on_second_attempt",
		"TestUnifiedSpec/retryable-reads/tests/unified/listDatabaseObjects.json/ListDatabaseObjects_fails_on_first_attempt",
		"TestUnifiedSpec/retryable-reads/tests/unified/listDatabaseObjects.json/ListDatabaseObjects_fails_on_second_attempt",
		"TestUnifiedSpec/retryable-reads/tests/unified/mapReduce.json/MapReduce_succeeds_with_retry_on",
		"TestUnifiedSpec/retryable-reads/tests/unified/mapReduce.json/MapReduce_fails_with_retry_on",
		"TestUnifiedSpec/retryable-reads/tests/unified/mapReduce.json/MapReduce_fails_with_retry_off",
	},

	// TODO(GODRIVER-2944): Setting "maxTimeMS" on a command that creates a cursor
	// may be surprising to users, so omit this from operations that return
	// user-managed cursors.
	"Omit maxTimeMS from user-managed cursor operations (DRIVERS-2722)": {
		"TestUnifiedSpec/client-side-operations-timeout/tests/gridfs-find.json/timeoutMS_can_be_overridden_for_a_find",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-operation-timeoutMS.json/timeoutMS_can_be_configured_for_an_operation_-_find_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-operation-timeoutMS.json/timeoutMS_can_be_configured_for_an_operation_-_aggregate_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-operation-timeoutMS.json/timeoutMS_can_be_configured_for_an_operation_-_aggregate_on_database",
		"TestUnifiedSpec/client-side-operations-timeout/tests/global-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoClient_-_find_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/global-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoClient_-_aggregate_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/global-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoClient_-_aggregate_on_database",
		"TestUnifiedSpec/client-side-operations-timeout/tests/retryability-timeoutMS.json/operation_is_retried_multiple_times_for_non-zero_timeoutMS_-_find_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/retryability-timeoutMS.json/operation_is_retried_multiple_times_for_non-zero_timeoutMS_-_aggregate_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/retryability-timeoutMS.json/operation_is_retried_multiple_times_for_non-zero_timeoutMS_-_aggregate_on_database",
		"TestUnifiedSpec/client-side-operations-timeout/tests/gridfs-find.json/timeoutMS_applied_to_find_command",
		"TestUnifiedSpec/client-side-operations-timeout/tests/tailable-awaitData.json/timeoutMS_applied_to_find",
		"TestUnifiedSpec/client-side-operations-timeout/tests/tailable-awaitData.json/timeoutMS_is_refreshed_for_getMore_if_maxAwaitTimeMS_is_not_set",
		"TestUnifiedSpec/client-side-operations-timeout/tests/tailable-awaitData.json/timeoutMS_is_refreshed_for_getMore_if_maxAwaitTimeMS_is_set",
		"TestUnifiedSpec/client-side-operations-timeout/tests/tailable-awaitData.json/timeoutMS_is_refreshed_for_getMore_-_failure",
	},

	// TODO(GODRIVER-3411): Tests require "getMore" with "maxTimeMS" settings. Not
	// supported for non-awaitData cursors.
	"Omit maxTimeMS from a getMore (DRIVERS-2953)": {
		"TestUnifiedSpec/client-side-operations-timeout/runCursorCommand.json/Non-tailable_cursor_lifetime_remaining_timeoutMS_applied_to_getMore_if_timeoutMode_is_unset",
	},

	// TODO(GODRIVER-2466): Convert SDAM integration spec tests to unified test format.
	"Convert SDAM integration tests to unified format (GODRIVER-2466)": {
		"TestUnifiedSpec/server-discovery-and-monitoring/tests/unified/rediscover-quickly-after-step-down.json/Rediscover_quickly_after_replSetStepDown",
	},

	// TODO(GODRIVER-2967): Implement TopologyChangedEvent on topology close.
	"Implement TopologyChangedEvent on close (GODRIVER-2967)": {
		"TestUnifiedSpec/server-discovery-and-monitoring/tests/unified/logging-loadbalanced.json/Topology_lifecycle",
		"TestUnifiedSpec/server-discovery-and-monitoring/tests/unified/logging-sharded.json/Topology_lifecycle",
		"TestUnifiedSpec/server-discovery-and-monitoring/tests/unified/logging-replicaset.json/Topology_lifecycle",
		"TestUnifiedSpec/server-discovery-and-monitoring/tests/unified/logging-standalone.json/Topology_lifecycle",
		"TestUnifiedSpec/server-discovery-and-monitoring/tests/unified/loadbalanced-emit-topology-changed-before-close.json/Topology_lifecycle",
		"TestUnifiedSpec/server-discovery-and-monitoring/tests/unified/sharded-emit-topology-changed-before-close.json/Topology_lifecycle",
		"TestUnifiedSpec/server-discovery-and-monitoring/tests/unified/replicaset-emit-topology-changed-before-close.json/Topology_lifecycle",
		"TestUnifiedSpec/server-discovery-and-monitoring/tests/unified/standalone-emit-topology-changed-before-close.json/Topology_lifecycle",
	},

	// Unknown BSON format.
	"Unsupported format": {
		"TestBsonBinaryVectorSpec/Tests_of_Binary_subtype_9,_Vectors,_with_dtype_FLOAT32/Infinity_Vector_FLOAT32/Marshaling",
		"TestBsonBinaryVectorSpec/Tests_of_Binary_subtype_9,_Vectors,_with_dtype_FLOAT32/Infinity_Vector_FLOAT32/Unmarshaling",
	},

	// Unsupported BSON binary vector tests.
	"compile-time restriction": {
		"TestBsonBinaryVectorSpec/Tests_of_Binary_subtype_9,_Vectors,_with_dtype_PACKED_BIT/Overflow_Vector_PACKED_BIT/Marshaling",
		"TestBsonBinaryVectorSpec/Tests_of_Binary_subtype_9,_Vectors,_with_dtype_INT8/Underflow_Vector_INT8/Marshaling",
		"TestBsonBinaryVectorSpec/Tests_of_Binary_subtype_9,_Vectors,_with_dtype_INT8/Overflow_Vector_INT8/Marshaling",
		"TestBsonBinaryVectorSpec/Tests_of_Binary_subtype_9,_Vectors,_with_dtype_PACKED_BIT/Negative_padding_PACKED_BIT/Marshaling",
		"TestBsonBinaryVectorSpec/Tests_of_Binary_subtype_9,_Vectors,_with_dtype_PACKED_BIT/Vector_with_float_values_PACKED_BIT/Marshaling",
		"TestBsonBinaryVectorSpec/Tests_of_Binary_subtype_9,_Vectors,_with_dtype_PACKED_BIT/Underflow_Vector_PACKED_BIT/Marshaling",
		"TestBsonBinaryVectorSpec/Tests_of_Binary_subtype_9,_Vectors,_with_dtype_INT8/INT8_with_float_inputs/Marshaling",
	},

	// Unsupported BSON binary vector padding.
	"private padding field": {
		"TestBsonBinaryVectorSpec/Tests_of_Binary_subtype_9,_Vectors,_with_dtype_INT8/INT8_with_padding/Marshaling",
		"TestBsonBinaryVectorSpec/Tests_of_Binary_subtype_9,_Vectors,_with_dtype_FLOAT32/FLOAT32_with_padding/Marshaling",
	},

	// Invalid BSON vector cases.
	"invalid case": {
		"TestBsonBinaryVectorSpec/Tests_of_Binary_subtype_9,_Vectors,_with_dtype_FLOAT32/Insufficient_vector_data_with_3_bytes_FLOAT32/Marshaling",
		"TestBsonBinaryVectorSpec/Tests_of_Binary_subtype_9,_Vectors,_with_dtype_FLOAT32/Insufficient_vector_data_with_5_bytes_FLOAT32/Marshaling",
	},

	// TODO(GODRIVER-3521): Extend Legacy Unified Spec Runner to include
	// client-side-encryption timeoutMS.
	"Extend Legacy Unified Spec Runner for client-side-encryption timeoutMS (GODRIVER-3521)": {
		"TestClientSideEncryptionSpec/timeoutMS.json/remaining_timeoutMS_applied_to_find_to_get_keyvault_data",
		"TestClientSideEncryptionSpec/timeoutMS.json/timeoutMS_applied_to_listCollections_to_get_collection_schema",
		"TestUnifiedSpec/client-side-encryption/tests/unified/timeoutMS.json/remaining_timeoutMS_applied_to_find_to_get_keyvault_data",
		"TestUnifiedSpec/client-side-encryption/tests/unified/timeoutMS.json/timeoutMS_applied_to_listCollections_to_get_collection_schema",
	},

	// TODO(GODRIVER-3076): CSFLE/QE Support for more than 1 KMS provider per
	// type.
	"Support multiple KMS providers per type (GODRIVER-3076)": {
		"TestUnifiedSpec/client-side-encryption/tests/unified/namedKMS-createDataKey.json/create_datakey_with_named_Azure_KMS_provider",
		"TestUnifiedSpec/client-side-encryption/tests/unified/namedKMS-createDataKey.json/create_datakey_with_named_GCP_KMS_provider",
		"TestUnifiedSpec/client-side-encryption/tests/unified/namedKMS-createDataKey.json/create_datakey_with_named_KMIP_KMS_provider",
		"TestUnifiedSpec/client-side-encryption/tests/unified/namedKMS-createDataKey.json/create_datakey_with_named_local_KMS_provider",
		"TestUnifiedSpec/client-side-encryption/tests/unified/namedKMS-explicit.json/can_explicitly_encrypt_with_a_named_KMS_provider",
		"TestUnifiedSpec/client-side-encryption/tests/unified/namedKMS-explicit.json/can_explicitly_decrypt_with_a_named_KMS_provider",
		"TestUnifiedSpec/client-side-encryption/tests/unified/namedKMS-rewrapManyDataKey.json/rewrap_to_aws:name1",
		"TestUnifiedSpec/client-side-encryption/tests/unified/namedKMS-rewrapManyDataKey.json/rewrap_to_azure:name1",
		"TestUnifiedSpec/client-side-encryption/tests/unified/namedKMS-rewrapManyDataKey.json/rewrap_to_gcp:name1",
		"TestUnifiedSpec/client-side-encryption/tests/unified/namedKMS-rewrapManyDataKey.json/rewrap_to_kmip:name1",
		"TestUnifiedSpec/client-side-encryption/tests/unified/namedKMS-rewrapManyDataKey.json/rewrap_to_local:name1",
		"TestUnifiedSpec/client-side-encryption/tests/unified/namedKMS-rewrapManyDataKey.json/rewrap_from_local:name1_to_local:name2",
		"TestUnifiedSpec/client-side-encryption/tests/unified/namedKMS-rewrapManyDataKey.json/rewrap_from_aws:name1_to_aws:name2",
		"TestUnifiedSpec/client-side-encryption/tests/unified/namedKMS-createDataKey.json/create_data_key_with_named_AWS_KMS_provider",
		"TestClientSideEncryptionSpec/namedKMS.json/Automatically_encrypt_and_decrypt_with_a_named_KMS_provider",
	},

	// TODO(GODRIVER-3380): Change stream should resume with CSOT failure.
	"Ensure change streams resume with CSOT failure (GODRIVER-3380)": {
		"TestUnifiedSpec/client-side-operations-timeout/tests/change-streams.json/timeoutMS_is_refreshed_for_getMore_if_maxAwaitTimeMS_is_not_set",
		"TestUnifiedSpec/client-side-operations-timeout/tests/change-streams.json/timeoutMS_is_refreshed_for_getMore_if_maxAwaitTimeMS_is_set",
		"TestUnifiedSpec/client-side-operations-timeout/tests/change-streams.json/timeoutMS_applies_to_full_resume_attempt_in_a_next_call",
		"TestUnifiedSpec/client-side-operations-timeout/tests/change-streams.json/change_stream_can_be_iterated_again_if_previous_iteration_times_out",
		"TestUnifiedSpec/client-side-operations-timeout/tests/change-streams.json/timeoutMS_is_refreshed_for_getMore_-_failure",
		"TestUnifiedSpec/client-side-operations-timeout/tests/change-streams.json/error_if_maxAwaitTimeMS_is_greater_than_timeoutMS",
	},

	// Unknown CSOT:
	"CSOT test not implemented": {
		"TestUnifiedSpec/client-side-operations-timeout/tests/close-cursors.json/timeoutMS_is_refreshed_for_close",
		"TestUnifiedSpec/client-side-operations-timeout/tests/convenient-transactions.json/withTransaction_raises_a_client-side_error_if_timeoutMS_is_overridden_inside_the_callback",
		"TestUnifiedSpec/client-side-operations-timeout/tests/convenient-transactions.json/timeoutMS_is_not_refreshed_for_each_operation_in_the_callback",
		"TestUnifiedSpec/client-side-operations-timeout/tests/cursors.json/find_errors_if_timeoutMode_is_set_and_timeoutMS_is_not",
		"TestUnifiedSpec/client-side-operations-timeout/tests/cursors.json/collection_aggregate_errors_if_timeoutMode_is_set_and_timeoutMS_is_not",
		"TestUnifiedSpec/client-side-operations-timeout/tests/cursors.json/database_aggregate_errors_if_timeoutMode_is_set_and_timeoutMS_is_not",
		"TestUnifiedSpec/client-side-operations-timeout/tests/cursors.json/listCollections_errors_if_timeoutMode_is_set_and_timeoutMS_is_not",
		"TestUnifiedSpec/client-side-operations-timeout/tests/cursors.json/listIndexes_errors_if_timeoutMode_is_set_and_timeoutMS_is_not",
		"TestUnifiedSpec/client-side-operations-timeout/tests/non-tailable-cursors.json/timeoutMS_applied_to_find_if_timeoutMode_is_cursor_lifetime",
		"TestUnifiedSpec/client-side-operations-timeout/tests/non-tailable-cursors.json/remaining_timeoutMS_applied_to_getMore_if_timeoutMode_is_unset",
		"TestUnifiedSpec/client-side-operations-timeout/tests/non-tailable-cursors.json/remaining_timeoutMS_applied_to_getMore_if_timeoutMode_is_cursor_lifetime",
		"TestUnifiedSpec/client-side-operations-timeout/tests/non-tailable-cursors.json/timeoutMS_applied_to_find_if_timeoutMode_is_iteration",
		"TestUnifiedSpec/client-side-operations-timeout/tests/non-tailable-cursors.json/timeoutMS_is_refreshed_for_getMore_if_timeoutMode_is_iteration_-_success",
		"TestUnifiedSpec/client-side-operations-timeout/tests/non-tailable-cursors.json/timeoutMS_is_refreshed_for_getMore_if_timeoutMode_is_iteration_-_failure",
		"TestUnifiedSpec/client-side-operations-timeout/tests/non-tailable-cursors.json/aggregate_with_$out_errors_if_timeoutMode_is_iteration",
		"TestUnifiedSpec/client-side-operations-timeout/tests/non-tailable-cursors.json/aggregate_with_$merge_errors_if_timeoutMode_is_iteration",
		"TestUnifiedSpec/client-side-operations-timeout/tests/gridfs-download.json/timeoutMS_applied_to_find_to_get_chunks",
		"TestUnifiedSpec/client-side-operations-timeout/tests/gridfs-download.json/timeoutMS_applied_to_entire_download,_not_individual_parts",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoCollection_-_aggregate_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoCollection_-_aggregate_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoCollection_-_count_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoCollection_-_count_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoCollection_-_countDocuments_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoCollection_-_countDocuments_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoCollection_-_estimatedDocumentCount_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoCollection_-_estimatedDocumentCount_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoCollection_-_distinct_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoCollection_-_distinct_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoCollection_-_find_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoCollection_-_find_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoCollection_-_findOne_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoCollection_-_findOne_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoCollection_-_listIndexes_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoCollection_-_listIndexes_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoCollection_-_listIndexNames_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoCollection_-_listIndexNames_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoCollection_-_createChangeStream_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoCollection_-_createChangeStream_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoCollection_-_insertOne_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoCollection_-_insertOne_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoCollection_-_insertMany_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoCollection_-_insertMany_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoCollection_-_deleteOne_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoCollection_-_deleteOne_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoCollection_-_deleteMany_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoCollection_-_deleteMany_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoCollection_-_replaceOne_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoCollection_-_replaceOne_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoCollection_-_updateOne_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoCollection_-_updateOne_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoCollection_-_updateMany_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoCollection_-_updateMany_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoCollection_-_findOneAndDelete_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoCollection_-_findOneAndDelete_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoCollection_-_findOneAndReplace_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoCollection_-_findOneAndReplace_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoCollection_-_findOneAndUpdate_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoCollection_-_findOneAndUpdate_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoCollection_-_bulkWrite_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoCollection_-_bulkWrite_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoCollection_-_createIndex_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoCollection_-_createIndex_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoCollection_-_dropIndex_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoCollection_-_dropIndex_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoCollection_-_dropIndexes_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-collection-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoCollection_-_dropIndexes_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoDatabase_-_aggregate_on_database",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoDatabase_-_aggregate_on_database",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoDatabase_-_listCollections_on_database",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoDatabase_-_listCollections_on_database",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoDatabase_-_listCollectionNames_on_database",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoDatabase_-_listCollectionNames_on_database",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoDatabase_-_runCommand_on_database",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoDatabase_-_runCommand_on_database",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoDatabase_-_createChangeStream_on_database",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoDatabase_-_createChangeStream_on_database",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoDatabase_-_aggregate_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoDatabase_-_aggregate_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoDatabase_-_count_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoDatabase_-_count_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoDatabase_-_countDocuments_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoDatabase_-_countDocuments_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoDatabase_-_estimatedDocumentCount_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoDatabase_-_estimatedDocumentCount_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoDatabase_-_distinct_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoDatabase_-_distinct_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoDatabase_-_find_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoDatabase_-_find_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoDatabase_-_findOne_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoDatabase_-_findOne_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoDatabase_-_listIndexes_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoDatabase_-_listIndexes_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoDatabase_-_listIndexNames_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoDatabase_-_listIndexNames_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoDatabase_-_createChangeStream_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoDatabase_-_createChangeStream_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoDatabase_-_insertOne_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoDatabase_-_insertOne_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoDatabase_-_insertMany_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoDatabase_-_insertMany_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoDatabase_-_deleteOne_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoDatabase_-_deleteOne_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoDatabase_-_deleteMany_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoDatabase_-_deleteMany_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoDatabase_-_replaceOne_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoDatabase_-_replaceOne_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoDatabase_-_updateOne_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoDatabase_-_updateOne_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoDatabase_-_updateMany_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoDatabase_-_updateMany_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoDatabase_-_findOneAndDelete_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoDatabase_-_findOneAndDelete_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoDatabase_-_findOneAndReplace_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoDatabase_-_findOneAndReplace_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoDatabase_-_findOneAndUpdate_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoDatabase_-_findOneAndUpdate_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoDatabase_-_bulkWrite_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoDatabase_-_bulkWrite_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoDatabase_-_createIndex_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoDatabase_-_createIndex_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoDatabase_-_dropIndex_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoDatabase_-_dropIndex_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_configured_on_a_MongoDatabase_-_dropIndexes_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/override-database-timeoutMS.json/timeoutMS_can_be_set_to_0_on_a_MongoDatabase_-_dropIndexes_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/sessions-inherit-timeoutMS.json/timeoutMS_applied_to_commitTransaction",
		"TestUnifiedSpec/client-side-operations-timeout/tests/sessions-inherit-timeoutMS.json/timeoutMS_applied_to_abortTransaction",
		"TestUnifiedSpec/client-side-operations-timeout/tests/sessions-inherit-timeoutMS.json/timeoutMS_applied_to_withTransaction",
		"TestUnifiedSpec/client-side-operations-timeout/tests/sessions-override-operation-timeoutMS.json/timeoutMS_applied_to_withTransaction",
		"TestUnifiedSpec/client-side-operations-timeout/tests/sessions-override-timeoutMS.json",
		"TestUnifiedSpec/client-side-operations-timeout/tests/tailable-awaitData.json/error_if_timeoutMode_is_cursor_lifetime",
		"TestUnifiedSpec/client-side-operations-timeout/tests/tailable-awaitData.json/timeoutMS_applied_to_find",
		"TestUnifiedSpec/client-side-operations-timeout/tests/tailable-awaitData.json/timeoutMS_is_refreshed_for_getMore_if_maxAwaitTimeMS_is_not_set",
		"TestUnifiedSpec/client-side-operations-timeout/tests/tailable-awaitData.json/timeoutMS_is_refreshed_for_getMore_if_maxAwaitTimeMS_is_set",
		"TestUnifiedSpec/client-side-operations-timeout/tests/tailable-awaitData.json/timeoutMS_is_refreshed_for_getMore_-_failure",
		"TestUnifiedSpec/client-side-operations-timeout/tests/tailable-awaitData.json/apply_maxAwaitTimeMS_if_less_than_remaining_timeout",
		"TestUnifiedSpec/client-side-operations-timeout/tests/tailable-non-awaitData.json/error_if_timeoutMode_is_cursor_lifetime",
		"TestUnifiedSpec/client-side-operations-timeout/tests/tailable-non-awaitData.json/timeoutMS_applied_to_find",
		"TestUnifiedSpec/client-side-operations-timeout/tests/tailable-non-awaitData.json/timeoutMS_is_refreshed_for_getMore_-_success",
		"TestUnifiedSpec/client-side-operations-timeout/tests/tailable-non-awaitData.json/timeoutMS_is_refreshed_for_getMore_-_failure",
	},

	"CSOT deprecated options test skipped": {
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/commitTransaction_ignores_socketTimeoutMS_if_timeoutMS_is_set",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/commitTransaction_ignores_wTimeoutMS_if_timeoutMS_is_set",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/commitTransaction_ignores_maxCommitTimeMS_if_timeoutMS_is_set",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/abortTransaction_ignores_socketTimeoutMS_if_timeoutMS_is_set",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/abortTransaction_ignores_wTimeoutMS_if_timeoutMS_is_set",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/withTransaction_ignores_socketTimeoutMS_if_timeoutMS_is_set",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/withTransaction_ignores_wTimeoutMS_if_timeoutMS_is_set",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/withTransaction_ignores_maxCommitTimeMS_if_timeoutMS_is_set",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/socketTimeoutMS_is_ignored_if_timeoutMS_is_set_-_listDatabases_on_client",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/wTimeoutMS_is_ignored_if_timeoutMS_is_set_-_listDatabases_on_client",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/socketTimeoutMS_is_ignored_if_timeoutMS_is_set_-_listDatabaseNames_on_client",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/wTimeoutMS_is_ignored_if_timeoutMS_is_set_-_listDatabaseNames_on_client",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/socketTimeoutMS_is_ignored_if_timeoutMS_is_set_-_createChangeStream_on_client",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/wTimeoutMS_is_ignored_if_timeoutMS_is_set_-_createChangeStream_on_client",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/socketTimeoutMS_is_ignored_if_timeoutMS_is_set_-_aggregate_on_database",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/wTimeoutMS_is_ignored_if_timeoutMS_is_set_-_aggregate_on_database",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/maxTimeMS_is_ignored_if_timeoutMS_is_set_-_aggregate_on_database",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/socketTimeoutMS_is_ignored_if_timeoutMS_is_set_-_listCollections_on_database",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/wTimeoutMS_is_ignored_if_timeoutMS_is_set_-_listCollections_on_database",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/socketTimeoutMS_is_ignored_if_timeoutMS_is_set_-_listCollectionNames_on_database",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/wTimeoutMS_is_ignored_if_timeoutMS_is_set_-_listCollectionNames_on_database",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/socketTimeoutMS_is_ignored_if_timeoutMS_is_set_-_runCommand_on_database",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/wTimeoutMS_is_ignored_if_timeoutMS_is_set_-_runCommand_on_database",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/socketTimeoutMS_is_ignored_if_timeoutMS_is_set_-_createChangeStream_on_database",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/wTimeoutMS_is_ignored_if_timeoutMS_is_set_-_createChangeStream_on_database",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/socketTimeoutMS_is_ignored_if_timeoutMS_is_set_-_aggregate_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/wTimeoutMS_is_ignored_if_timeoutMS_is_set_-_aggregate_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/maxTimeMS_is_ignored_if_timeoutMS_is_set_-_aggregate_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/socketTimeoutMS_is_ignored_if_timeoutMS_is_set_-_count_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/wTimeoutMS_is_ignored_if_timeoutMS_is_set_-_count_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/maxTimeMS_is_ignored_if_timeoutMS_is_set_-_count_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/socketTimeoutMS_is_ignored_if_timeoutMS_is_set_-_countDocuments_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/wTimeoutMS_is_ignored_if_timeoutMS_is_set_-_countDocuments_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/socketTimeoutMS_is_ignored_if_timeoutMS_is_set_-_estimatedDocumentCount_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/wTimeoutMS_is_ignored_if_timeoutMS_is_set_-_estimatedDocumentCount_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/maxTimeMS_is_ignored_if_timeoutMS_is_set_-_estimatedDocumentCount_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/socketTimeoutMS_is_ignored_if_timeoutMS_is_set_-_distinct_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/wTimeoutMS_is_ignored_if_timeoutMS_is_set_-_distinct_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/maxTimeMS_is_ignored_if_timeoutMS_is_set_-_distinct_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/socketTimeoutMS_is_ignored_if_timeoutMS_is_set_-_find_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/wTimeoutMS_is_ignored_if_timeoutMS_is_set_-_find_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/maxTimeMS_is_ignored_if_timeoutMS_is_set_-_find_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/socketTimeoutMS_is_ignored_if_timeoutMS_is_set_-_findOne_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/wTimeoutMS_is_ignored_if_timeoutMS_is_set_-_findOne_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/maxTimeMS_is_ignored_if_timeoutMS_is_set_-_findOne_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/socketTimeoutMS_is_ignored_if_timeoutMS_is_set_-_listIndexes_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/wTimeoutMS_is_ignored_if_timeoutMS_is_set_-_listIndexes_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/socketTimeoutMS_is_ignored_if_timeoutMS_is_set_-_listIndexNames_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/wTimeoutMS_is_ignored_if_timeoutMS_is_set_-_listIndexNames_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/socketTimeoutMS_is_ignored_if_timeoutMS_is_set_-_createChangeStream_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/wTimeoutMS_is_ignored_if_timeoutMS_is_set_-_createChangeStream_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/socketTimeoutMS_is_ignored_if_timeoutMS_is_set_-_insertOne_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/wTimeoutMS_is_ignored_if_timeoutMS_is_set_-_insertOne_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/socketTimeoutMS_is_ignored_if_timeoutMS_is_set_-_insertMany_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/wTimeoutMS_is_ignored_if_timeoutMS_is_set_-_insertMany_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/socketTimeoutMS_is_ignored_if_timeoutMS_is_set_-_deleteOne_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/wTimeoutMS_is_ignored_if_timeoutMS_is_set_-_deleteOne_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/socketTimeoutMS_is_ignored_if_timeoutMS_is_set_-_deleteMany_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/wTimeoutMS_is_ignored_if_timeoutMS_is_set_-_deleteMany_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/socketTimeoutMS_is_ignored_if_timeoutMS_is_set_-_replaceOne_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/wTimeoutMS_is_ignored_if_timeoutMS_is_set_-_replaceOne_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/socketTimeoutMS_is_ignored_if_timeoutMS_is_set_-_updateOne_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/wTimeoutMS_is_ignored_if_timeoutMS_is_set_-_updateOne_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/socketTimeoutMS_is_ignored_if_timeoutMS_is_set_-_updateMany_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/wTimeoutMS_is_ignored_if_timeoutMS_is_set_-_updateMany_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/socketTimeoutMS_is_ignored_if_timeoutMS_is_set_-_findOneAndDelete_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/wTimeoutMS_is_ignored_if_timeoutMS_is_set_-_findOneAndDelete_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/maxTimeMS_is_ignored_if_timeoutMS_is_set_-_findOneAndDelete_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/socketTimeoutMS_is_ignored_if_timeoutMS_is_set_-_findOneAndReplace_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/wTimeoutMS_is_ignored_if_timeoutMS_is_set_-_findOneAndReplace_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/maxTimeMS_is_ignored_if_timeoutMS_is_set_-_findOneAndReplace_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/socketTimeoutMS_is_ignored_if_timeoutMS_is_set_-_findOneAndUpdate_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/wTimeoutMS_is_ignored_if_timeoutMS_is_set_-_findOneAndUpdate_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/maxTimeMS_is_ignored_if_timeoutMS_is_set_-_findOneAndUpdate_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/socketTimeoutMS_is_ignored_if_timeoutMS_is_set_-_bulkWrite_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/wTimeoutMS_is_ignored_if_timeoutMS_is_set_-_bulkWrite_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/socketTimeoutMS_is_ignored_if_timeoutMS_is_set_-_createIndex_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/wTimeoutMS_is_ignored_if_timeoutMS_is_set_-_createIndex_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/maxTimeMS_is_ignored_if_timeoutMS_is_set_-_createIndex_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/socketTimeoutMS_is_ignored_if_timeoutMS_is_set_-_dropIndex_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/wTimeoutMS_is_ignored_if_timeoutMS_is_set_-_dropIndex_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/maxTimeMS_is_ignored_if_timeoutMS_is_set_-_dropIndex_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/socketTimeoutMS_is_ignored_if_timeoutMS_is_set_-_dropIndexes_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/wTimeoutMS_is_ignored_if_timeoutMS_is_set_-_dropIndexes_on_collection",
		"TestUnifiedSpec/client-side-operations-timeout/tests/deprecated-options.json/maxTimeMS_is_ignored_if_timeoutMS_is_set_-_dropIndexes_on_collection",
	},

	"Go Driver does not support legacy CSOT": {
		"TestUnifiedSpec/client-side-operations-timeout/tests/legacy-timeouts.json/wTimeoutMS_is_not_used_to_derive_a_maxTimeMS_command_field",
		"TestUnifiedSpec/client-side-operations-timeout/tests/legacy-timeouts.json/maxTimeMS_option_is_used_directly_as_the_maxTimeMS_field_on_a_command",
		"TestUnifiedSpec/client-side-operations-timeout/tests/legacy-timeouts.json/maxCommitTimeMS_option_is_used_directly_as_the_maxTimeMS_field_on_a_commitTransaction_command",
	},

	// TODO(GODRIVER-3106): "hello" failpoint in CSOT command-execution UST is premature.
	"Address premature 'hello' failpoint in CSOT command-execution (GODRIVER-3106)": {
		"TestUnifiedSpec/client-side-operations-timeout/tests/command-execution.json/maxTimeMS_value_in_the_command_is_less_than_timeoutMS",
	},

	// TODO(GODRIVER-3415): Add performant "rename all revisions by filename"
	// feature - rename_by_name.
	"Add performant 'rename all revisions by filename' feature (GODRIVER-3415)": {
		"TestUnifiedSpec/gridfs/tests/deleteByName.json/delete_when_multiple_revisions_of_the_file_exist",
		"TestUnifiedSpec/gridfs/tests/deleteByName.json/delete_when_file_name_does_not_exist",
		"TestUnifiedSpec/gridfs/tests/renameByName.json/rename_when_multiple_revisions_of_the_file_exist",
		"TestUnifiedSpec/gridfs/tests/renameByName.json/rename_when_file_name_does_not_exist",
	},

	// TODO(GODRIVER-3393): Add test that PoolClearedEvent is emitted before
	// ConnectionCheckedInEvent/ConnectionCheckOutFailedEvent.
	"Ensure PoolClearedEvent is emitted before ConnectionCheckedIn/ConnectionCheckOutFailed (GODRIVER-3393)": {
		"TestUnifiedSpec/server-discovery-and-monitoring/tests/unified/pool-clear-checkout-error.json/Pool_is_cleared_before_connection_is_closed_(authentication_error)",
		"TestUnifiedSpec/server-discovery-and-monitoring/tests/unified/pool-clear-checkout-error.json/Pool_is_cleared_before_connection_is_closed_(handshake_error)",
		"TestUnifiedSpec/server-discovery-and-monitoring/tests/unified/pool-clear-min-pool-size-error.json/Pool_is_cleared_on_authentication_error_during_minPoolSize_population",
		"TestUnifiedSpec/server-discovery-and-monitoring/tests/unified/pool-clear-min-pool-size-error.json/Pool_is_cleared_on_handshake_error_during_minPoolSize_population",
	},

	// TODO(GODRIVER-3155): Convert read/write concern spec tests to unified test
	// format.
	"Convert read/write concern spec tests to unified format (GODRIVER-3155)": {
		"TestUnifiedSpecs/read-write-concern/tests/operation/default-write-concern-2.6.json",
		"TestUnifiedSpecs/read-write-concern/tests/operation/default-write-concern-3.2.json",
		"TestUnifiedSpecs/read-write-concern/tests/operation/default-write-concern-3.4.json",
		"TestUnifiedSpecs/read-write-concern/tests/operation/default-write-concern-4.2.json",
	},

	// TODO(GODRIVER-2191): Drivers should retry operations if connection
	// handshake fails.
	"Retry operations on handshake failure (GODRIVER-2191)": {
		"TestUnifiedSpec/retryable-reads/tests/unified/handshakeError.json/client.listDatabases_succeeds_after_retryable_handshake_network_error",
		"TestUnifiedSpec/retryable-reads/tests/unified/handshakeError.json/client.listDatabases_succeeds_after_retryable_handshake_server_error_(ShutdownInProgress)",
		"TestUnifiedSpec/retryable-reads/tests/unified/handshakeError.json/client.listDatabaseNames_succeeds_after_retryable_handshake_network_error",
		"TestUnifiedSpec/retryable-reads/tests/unified/handshakeError.json/client.listDatabaseNames_succeeds_after_retryable_handshake_server_error_(ShutdownInProgress)",
		"TestUnifiedSpec/retryable-reads/tests/unified/handshakeError.json/client.createChangeStream_succeeds_after_retryable_handshake_network_error",
		"TestUnifiedSpec/retryable-reads/tests/unified/handshakeError.json/client.createChangeStream_succeeds_after_retryable_handshake_server_error_(ShutdownInProgress)",
		"TestUnifiedSpec/retryable-reads/tests/unified/handshakeError.json/database.aggregate_succeeds_after_retryable_handshake_network_error",
		"TestUnifiedSpec/retryable-reads/tests/unified/handshakeError.json/database.aggregate_succeeds_after_retryable_handshake_server_error_(ShutdownInProgress)",
		"TestUnifiedSpec/retryable-reads/tests/unified/handshakeError.json/database.listCollections_succeeds_after_retryable_handshake_network_error",
		"TestUnifiedSpec/retryable-reads/tests/unified/handshakeError.json/database.listCollections_succeeds_after_retryable_handshake_server_error_(ShutdownInProgress)",
		"TestUnifiedSpec/retryable-reads/tests/unified/handshakeError.json/database.listCollectionNames_succeeds_after_retryable_handshake_network_error",
		"TestUnifiedSpec/retryable-reads/tests/unified/handshakeError.json/database.listCollectionNames_succeeds_after_retryable_handshake_server_error_(ShutdownInProgress)",
		"TestUnifiedSpec/retryable-reads/tests/unified/handshakeError.json/database.createChangeStream_succeeds_after_retryable_handshake_network_error",
		"TestUnifiedSpec/retryable-reads/tests/unified/handshakeError.json/database.createChangeStream_succeeds_after_retryable_handshake_server_error_(ShutdownInProgress)",
		"TestUnifiedSpec/retryable-reads/tests/unified/handshakeError.json/collection.aggregate_succeeds_after_retryable_handshake_network_error",
		"TestUnifiedSpec/retryable-reads/tests/unified/handshakeError.json/collection.aggregate_succeeds_after_retryable_handshake_server_error_(ShutdownInProgress)",
		"TestUnifiedSpec/retryable-reads/tests/unified/handshakeError.json/collection.countDocuments_succeeds_after_retryable_handshake_network_error",
		"TestUnifiedSpec/retryable-reads/tests/unified/handshakeError.json/collection.countDocuments_succeeds_after_retryable_handshake_server_error_(ShutdownInProgress)",
		"TestUnifiedSpec/retryable-reads/tests/unified/handshakeError.json/collection.estimatedDocumentCount_succeeds_after_retryable_handshake_network_error",
		"TestUnifiedSpec/retryable-reads/tests/unified/handshakeError.json/collection.estimatedDocumentCount_succeeds_after_retryable_handshake_server_error_(ShutdownInProgress)",
		"TestUnifiedSpec/retryable-reads/tests/unified/handshakeError.json/collection.distinct_succeeds_after_retryable_handshake_network_error",
		"TestUnifiedSpec/retryable-reads/tests/unified/handshakeError.json/collection.distinct_succeeds_after_retryable_handshake_server_error_(ShutdownInProgress)",
		"TestUnifiedSpec/retryable-reads/tests/unified/handshakeError.json/collection.find_succeeds_after_retryable_handshake_network_error",
		"TestUnifiedSpec/retryable-reads/tests/unified/handshakeError.json/collection.find_succeeds_after_retryable_handshake_server_error_(ShutdownInProgress)",
		"TestUnifiedSpec/retryable-reads/tests/unified/handshakeError.json/collection.findOne_succeeds_after_retryable_handshake_network_error",
		"TestUnifiedSpec/retryable-reads/tests/unified/handshakeError.json/collection.findOne_succeeds_after_retryable_handshake_server_error_(ShutdownInProgress)",
		"TestUnifiedSpec/retryable-reads/tests/unified/handshakeError.json/collection.listIndexes_succeeds_after_retryable_handshake_network_error",
		"TestUnifiedSpec/retryable-reads/tests/unified/handshakeError.json/collection.listIndexes_succeeds_after_retryable_handshake_server_error_(ShutdownInProgress)",
		"TestUnifiedSpec/retryable-reads/tests/unified/handshakeError.json/collection.createChangeStream_succeeds_after_retryable_handshake_network_error",
		"TestUnifiedSpec/retryable-reads/tests/unified/handshakeError.json/collection.createChangeStream_succeeds_after_retryable_handshake_server_error_(ShutdownInProgress)",
		"TestUnifiedSpec/retryable-writes/tests/unified/handshakeError.json",
		"TestUnifiedSpec/retryable-writes/tests/unified/handshakeError.json/client.clientBulkWrite_succeeds_after_retryable_handshake_network_error",
		"TestUnifiedSpec/retryable-writes/tests/unified/handshakeError.json/client.clientBulkWrite_succeeds_after_retryable_handshake_server_error_(ShutdownInProgress)",
		"TestUnifiedSpec/retryable-writes/tests/unified/handshakeError.json/collection.insertOne_succeeds_after_retryable_handshake_network_error",
		"TestUnifiedSpec/retryable-writes/tests/unified/handshakeError.json/collection.insertOne_succeeds_after_retryable_handshake_server_error_(ShutdownInProgress)",
		"TestUnifiedSpec/retryable-writes/tests/unified/handshakeError.json/collection.insertMany_succeeds_after_retryable_handshake_network_error",
		"TestUnifiedSpec/retryable-writes/tests/unified/handshakeError.json/collection.insertMany_succeeds_after_retryable_handshake_server_error_(ShutdownInProgress)",
		"TestUnifiedSpec/retryable-writes/tests/unified/handshakeError.json/collection.deleteOne_succeeds_after_retryable_handshake_network_error",
		"TestUnifiedSpec/retryable-writes/tests/unified/handshakeError.json/collection.deleteOne_succeeds_after_retryable_handshake_server_error_(ShutdownInProgress)",
		"TestUnifiedSpec/retryable-writes/tests/unified/handshakeError.json/collection.replaceOne_succeeds_after_retryable_handshake_network_error",
		"TestUnifiedSpec/retryable-writes/tests/unified/handshakeError.json/collection.replaceOne_succeeds_after_retryable_handshake_server_error_(ShutdownInProgress)",
		"TestUnifiedSpec/retryable-writes/tests/unified/handshakeError.json/collection.updateOne_succeeds_after_retryable_handshake_network_error",
		"TestUnifiedSpec/retryable-writes/tests/unified/handshakeError.json/collection.updateOne_succeeds_after_retryable_handshake_server_error_(ShutdownInProgress)",
		"TestUnifiedSpec/retryable-writes/tests/unified/handshakeError.json/collection.findOneAndDelete_succeeds_after_retryable_handshake_network_error",
		"TestUnifiedSpec/retryable-writes/tests/unified/handshakeError.json/collection.findOneAndDelete_succeeds_after_retryable_handshake_server_error_(ShutdownInProgress)",
		"TestUnifiedSpec/retryable-writes/tests/unified/handshakeError.json/collection.findOneAndReplace_succeeds_after_retryable_handshake_network_error",
		"TestUnifiedSpec/retryable-writes/tests/unified/handshakeError.json/collection.findOneAndReplace_succeeds_after_retryable_handshake_server_error_(ShutdownInProgress)",
		"TestUnifiedSpec/retryable-writes/tests/unified/handshakeError.json/collection.findOneAndUpdate_succeeds_after_retryable_handshake_network_error",
		"TestUnifiedSpec/retryable-writes/tests/unified/handshakeError.json/collection.findOneAndUpdate_succeeds_after_retryable_handshake_server_error_(ShutdownInProgress)",
		"TestUnifiedSpec/retryable-writes/tests/unified/handshakeError.json/collection.bulkWrite_succeeds_after_retryable_handshake_network_error",
		"TestUnifiedSpec/retryable-writes/tests/unified/handshakeError.json/collection.bulkWrite_succeeds_after_retryable_handshake_server_error_(ShutdownInProgress)",
	},

	// TODO(GODRIVER-3524): Change streams expanded events present by default in
	// 8.2+.
	"Change streams expanded events for MongoDB 8.2+ (GODRIVER-3524)": {
		"TestUnifiedSpec/change-streams/tests/unified/change-streams-disambiguatedPaths.json/disambiguatedPaths_is_not_present_when_showExpandedEvents_is_false/unset",
		"TestUnifiedSpec/change-streams/tests/unified/change-streams.json/Test_insert,_update,_replace,_and_delete_event_types",
		"TestUnifiedSpec/change-streams/tests/unified/change-streams.json/Test_array_truncation",
	},

	// TODO(GODRIVER-3137): Gossip cluster time from internal MongoClient to
	// session entities.
	"Must advance cluster times in unified spec runner (GODRIVER-3137)": {
		"TestUnifiedSpec/transactions/unified/mongos-unpin.json/unpin_after_TransientTransactionError_error_on_commit",
		// This test fails with the same error as GODRIVER-3137, but is not
		// directly referenced as an impacted test case by DRIVERS-2816. It
		// seems likely that the same change will resolve the failure, so I'm
		// including it here.
		"TestUnifiedSpec/unified-test-format/tests/valid-pass/poc-transactions-convenient-api.json/withTransaction_and_no_transaction_options_set",
		"TestUnifiedSpec/unified-test-format/tests/valid-pass/poc-transactions-convenient-api.json/withTransaction_inherits_transaction_options_from_client",
		"TestUnifiedSpec/unified-test-format/tests/valid-pass/poc-transactions-convenient-api.json/withTransaction_inherits_transaction_options_from_defaultTransactionOptions",
		"TestUnifiedSpec/unified-test-format/tests/valid-pass/poc-transactions-convenient-api.json/withTransaction_explicit_transaction_options",
		"TestUnifiedSpec/unified-test-format/tests/valid-pass/poc-transactions-mongos-pin-auto.json/remain_pinned_after_non-transient_Interrupted_error_on_insertOne",
		"TestUnifiedSpec/unified-test-format/tests/valid-pass/poc-transactions-mongos-pin-auto.json/unpin_after_transient_error_within_a_transaction",
		"TestUnifiedSpec/transactions-convenient-api/tests/unified/callback-aborts.json/withTransaction_succeeds_if_callback_aborts",
		"TestUnifiedSpec/transactions-convenient-api/tests/unified/callback-commits.json/withTransaction_succeeds_if_callback_commits",
		"TestUnifiedSpec/transactions-convenient-api/tests/unified/callback-commits.json/withTransaction_still_succeeds_if_callback_commits_and_runs_extra_op",
		"TestUnifiedSpec/transactions-convenient-api/tests/unified/callback-aborts.json/withTransaction_still_succeeds_if_callback_aborts_and_runs_extra_op",
		"TestUnifiedSpec/transactions-convenient-api/tests/unified/commit.json/withTransaction_commits_after_callback_returns_(second_transaction)",
		"TestUnifiedSpec/transactions-convenient-api/tests/unified/transaction-options.json/withTransaction_and_no_transaction_options_set",
		"TestUnifiedSpec/transactions-convenient-api/tests/unified/transaction-options.json/withTransaction_inherits_transaction_options_from_client",
		"TestUnifiedSpec/transactions-convenient-api/tests/unified/transaction-options.json/withTransaction_inherits_transaction_options_from_defaultTransactionOptions",
		"TestUnifiedSpec/transactions-convenient-api/tests/unified/transaction-options.json/withTransaction_explicit_transaction_options",
		"TestUnifiedSpec/transactions-convenient-api/tests/unified/transaction-options.json/withTransaction_explicit_transaction_options_override_defaultTransactionOptions",
		"TestUnifiedSpec/transactions-convenient-api/tests/unified/transaction-options.json/withTransaction_explicit_transaction_options_override_client_options",
		"TestUnifiedSpec/transactions-convenient-api/tests/unified/commit.json/withTransaction_commits_after_callback_returns",
	},

	"Address CSOT Compliance Issue in Timeout Handling for Cursor Constructors (GODRIVER-3480)": {
		"TestUnifiedSpec/client-side-operations-timeout/tests/tailable-awaitData.json/apply_remaining_timeoutMS_if_less_than_maxAwaitTimeMS",
	},

	// The Go Driver does not support "iteration" mode for cursors. That is,
	// we do not apply the timeout used to construct the cursor when using the
	// cursor, rather we apply the context-level timeout if one is provided. It's
	// doubtful that we will ever support this mode, so we skip these tests.
	//
	// If we do ever support this mode, it will be done as part of DRIVERS-2722
	// which does not currently have driver-specific tickets.
	//
	// Note that we have integration tests that cover the cases described in these
	// tests upto what is supported in the Go Driver. See GODRIVER-3473
	"Change CSOT default cursor timeout mode to ITERATION (DRIVERS-2772)": {
		"TestUnifiedSpec/client-side-operations-timeout/tests/tailable-awaitData.json/apply_remaining_timeoutMS_if_less_than_maxAwaitTimeMS",
		"TestUnifiedSpec/client-side-operations-timeout/tests/tailable-awaitData.json/error_if_maxAwaitTimeMS_is_equal_to_timeoutMS",
		"TestUnifiedSpec/client-side-operations-timeout/tests/change-streams.json/error_if_maxAwaitTimeMS_is_equal_to_timeoutMS",
		"TestUnifiedSpec/client-side-operations-timeout/tests/tailable-awaitData.json/error_if_maxAwaitTimeMS_is_greater_than_timeoutMS",
	},

	// TODO(GODRIVER-3641): Ensure Driver Errors for Tailable AwaitData Cursors on
	// Invalid maxAwaitTimeMS.
	"Ensure Driver Errors for Tailable AwaitData Cursors on Invalid maxAwaitTimeMS (GODRIVER-3641)": {
		"TestUnifiedSpec/client-side-operations-timeout/tests/tailable-awaitData.json/error_on_find_if_maxAwaitTimeMS_is_greater_than_timeoutMS",
		"TestUnifiedSpec/client-side-operations-timeout/tests/tailable-awaitData.json/error_on_aggregate_if_maxAwaitTimeMS_is_greater_than_timeoutMS",
		"TestUnifiedSpec/client-side-operations-timeout/tests/tailable-awaitData.json/error_on_watch_if_maxAwaitTimeMS_is_greater_than_timeoutMS",
		"TestUnifiedSpec/client-side-operations-timeout/tests/tailable-awaitData.json/error_on_find_if_maxAwaitTimeMS_is_equal_to_timeoutMS",
		"TestUnifiedSpec/client-side-operations-timeout/tests/tailable-awaitData.json/error_on_aggregate_if_maxAwaitTimeMS_is_equal_to_timeoutMS",
		"TestUnifiedSpec/client-side-operations-timeout/tests/tailable-awaitData.json/error_on_watch_if_maxAwaitTimeMS_is_equal_to_timeoutMS",
	},

	// TODO(GODRIVER-3403): Support queryable encryption in Client.BulkWrite.
	"Support queryable encryption in Client.BulkWrite (GODRIVER-3403)": {
		"TestUnifiedSpec/crud/tests/unified/client-bulkWrite-qe.json",
		"TestUnifiedSpec/client-side-encryption/tests/unified/client-bulkWrite-qe.json",
	},

	// Pre-4.2 SDAM tests
	"Pre-4.2 SDAM tests": {
		"TestSDAMSpec/errors/pre-42-InterruptedAtShutdown.json",
		"TestSDAMSpec/errors/pre-42-InterruptedDueToReplStateChange.json",
		"TestSDAMSpec/errors/pre-42-LegacyNotPrimary.json",
		"TestSDAMSpec/errors/pre-42-NotPrimaryNoSecondaryOk.json",
		"TestSDAMSpec/errors/pre-42-NotPrimaryOrSecondary.json",
		"TestSDAMSpec/errors/pre-42-NotWritablePrimary.json",
		"TestSDAMSpec/errors/pre-42-PrimarySteppedDown.json",
		"TestSDAMSpec/errors/pre-42-ShutdownInProgress.json",
	},

	// TODO(DRIVERS-3356): Unskip this test when the spec test bug is fixed.
	"Handshake spec test 'metadata-not-propagated.yml' fails on sharded clusters (DRIVERS-3356)": {
		"TestUnifiedSpec/mongodb-handshake/tests/unified/metadata-not-propagated.json/metadata_append_does_not_create_new_connections_or_close_existing_ones_and_no_hello_command_is_sent",
	},

	// TODO(GODRIVER-3637): Implement client backpressure.
	"Implement client backpressure (GODRIVER-3637)": {
		"TestUnifiedSpec/server-discovery-and-monitoring/tests/unified/backpressure-network-error-fail-replicaset.json/apply_backpressure_on_network_connection_errors_during_connection_establishment",
		"TestUnifiedSpec/server-discovery-and-monitoring/tests/unified/backpressure-network-error-fail-single.json/apply_backpressure_on_network_connection_errors_during_connection_establishment",
		"TestUnifiedSpec/server-discovery-and-monitoring/tests/unified/backpressure-server-description-unchanged-on-min-pool-size-population-error.json/the_server_description_is_not_changed_on_handshake_error_during_minPoolSize_population",
		"TestUnifiedSpec/server-discovery-and-monitoring/tests/unified/backpressure-network-error-fail.json/apply_backpressure_on_network_connection_errors_during_connection_establishment",
		"TestUnifiedSpec/server-discovery-and-monitoring/tests/unified/pool-clear-min-pool-size-error.json/Pool_is_not_cleared_on_handshake_error_during_minPoolSize_population",
		"TestSDAMSpec/errors/error_handling_handshake.json",
	},
}

// CheckSkip checks if the fully-qualified test name matches a list of skipped test names for a given reason.
// If the test name matches any from a list, the reason is logged and the test is skipped.
func CheckSkip(t *testing.T) {
	for reason, tests := range skipTests {
		for _, testName := range tests {
			if t.Name() == testName {
				t.Skipf("Skipping due to known failure: %q", reason)
			}
		}
	}
}
