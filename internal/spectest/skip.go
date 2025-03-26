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

	// TODO(GODRIVER-2183): Socks5 Proxy Support.
	"TestURIOptionsSpec/proxy-options.json/proxyPort_without_proxyHost":               "Implement GODRIVER-2183 for Socks5 Proxy Support",
	"TestURIOptionsSpec/proxy-options.json/proxyUsername_without_proxyHost":           "Implement GODRIVER-2183 for Socks5 Proxy Support",
	"TestURIOptionsSpec/proxy-options.json/proxyPassword_without_proxyHost":           "Implement GODRIVER-2183 for Socks5 Proxy Support",
	"TestURIOptionsSpec/proxy-options.json/all_other_proxy_options_without_proxyHost": "Implement GODRIVER-2183 for Socks5 Proxy Support",
	"TestURIOptionsSpec/proxy-options.json/proxyUsername_without_proxyPassword":       "Implement GODRIVER-2183 for Socks5 Proxy Support",
	"TestURIOptionsSpec/proxy-options.json/proxyPassword_without_proxyUsername":       "Implement GODRIVER-2183 for Socks5 Proxy Support",
	"TestURIOptionsSpec/proxy-options.json/multiple_proxyHost_parameters":             "Implement GODRIVER-2183 for Socks5 Proxy Support",
	"TestURIOptionsSpec/proxy-options.json/multiple_proxyPort_parameters":             "Implement GODRIVER-2183 for Socks5 Proxy Support",
	"TestURIOptionsSpec/proxy-options.json/multiple_proxyUsername_parameters":         "Implement GODRIVER-2183 for Socks5 Proxy Support",
	"TestURIOptionsSpec/proxy-options.json/multiple_proxyPassword_parameters":         "Implement GODRIVER-2183 for Socks5 Proxy Support",

	// wtimeoutMS write concern option is not supported.
	"TestURIOptionsSpec/concern-options.json/Valid_read_and_write_concern_are_parsed_correctly": "wtimeoutMS is deprecated",
	"TestURIOptionsSpec/concern-options.json/Non-numeric_wTimeoutMS_causes_a_warning":           "wtimeoutMS is deprecated",
	"TestURIOptionsSpec/concern-options.json/Too_low_wTimeoutMS_causes_a_warning":               "wtimeoutMS is deprecated",
	"TestReadWriteConcernSpec/connstring/write-concern.json/wtimeoutMS_as_an_invalid_number":    "wtimeoutMS is deprecated",

	// Unsupported TLS behavior in connection strings.
	"TestURIOptionsSpec/tls-options.json/tlsAllowInvalidCertificates_and_tlsDisableCertificateRevocationCheck_both_present_(and_true)_raises_an_error":  "unsupported connstring behavior",
	"TestURIOptionsSpec/tls-options.json/tlsAllowInvalidCertificates=true_and_tlsDisableCertificateRevocationCheck=false_raises_an_error":               "unsupported connstring behavior",
	"TestURIOptionsSpec/tls-options.json/tlsAllowInvalidCertificates=false_and_tlsDisableCertificateRevocationCheck=true_raises_an_error":               "unsupported connstring behavior",
	"TestURIOptionsSpec/tls-options.json/tlsAllowInvalidCertificates_and_tlsDisableCertificateRevocationCheck_both_present_(and_false)_raises_an_error": "unsupported connstring behavior",
	"TestURIOptionsSpec/tls-options.json/tlsDisableCertificateRevocationCheck_and_tlsAllowInvalidCertificates_both_present_(and_true)_raises_an_error":  "unsupported connstring behavior",
	"TestURIOptionsSpec/tls-options.json/tlsDisableCertificateRevocationCheck=true_and_tlsAllowInvalidCertificates=false_raises_an_error":               "unsupported connstring behavior",
	"TestURIOptionsSpec/tls-options.json/tlsDisableCertificateRevocationCheck=false_and_tlsAllowInvalidCertificates=true_raises_an_error":               "unsupported connstring behavior",
	"TestURIOptionsSpec/tls-options.json/tlsDisableCertificateRevocationCheck_and_tlsAllowInvalidCertificates_both_present_(and_false)_raises_an_error": "unsupported connstring behavior",
	"TestURIOptionsSpec/tls-options.json/tlsInsecure_and_tlsDisableCertificateRevocationCheck_both_present_(and_true)_raises_an_error":                  "unsupported connstring behavior",
	"TestURIOptionsSpec/tls-options.json/tlsInsecure=true_and_tlsDisableCertificateRevocationCheck=false_raises_an_error":                               "unsupported connstring behavior",
	"TestURIOptionsSpec/tls-options.json/tlsInsecure=false_and_tlsDisableCertificateRevocationCheck=true_raises_an_error":                               "unsupported connstring behavior",
	"TestURIOptionsSpec/tls-options.json/tlsInsecure_and_tlsDisableCertificateRevocationCheck_both_present_(and_false)_raises_an_error":                 "unsupported connstring behavior",
	"TestURIOptionsSpec/tls-options.json/tlsDisableCertificateRevocationCheck_and_tlsInsecure_both_present_(and_true)_raises_an_error":                  "unsupported connstring behavior",
	"TestURIOptionsSpec/tls-options.json/tlsDisableCertificateRevocationCheck=true_and_tlsInsecure=false_raises_an_error":                               "unsupported connstring behavior",
	"TestURIOptionsSpec/tls-options.json/tlsDisableCertificateRevocationCheck=false_and_tlsInsecure=true_raises_an_error":                               "unsupported connstring behavior",
	"TestURIOptionsSpec/tls-options.json/tlsDisableCertificateRevocationCheck_and_tlsInsecure_both_present_(and_false)_raises_an_error":                 "unsupported connstring behavior",
	"TestURIOptionsSpec/tls-options.json/tlsDisableCertificateRevocationCheck_and_tlsDisableOCSPEndpointCheck_both_present_(and_true)_raises_an_error":  "unsupported connstring behavior",
	"TestURIOptionsSpec/tls-options.json/tlsDisableCertificateRevocationCheck=true_and_tlsDisableOCSPEndpointCheck=false_raises_an_error":               "unsupported connstring behavior",
	"TestURIOptionsSpec/tls-options.json/tlsDisableCertificateRevocationCheck=false_and_tlsDisableOCSPEndpointCheck=true_raises_an_error":               "unsupported connstring behavior",
	"TestURIOptionsSpec/tls-options.json/tlsDisableCertificateRevocationCheck_and_tlsDisableOCSPEndpointCheck_both_present_(and_false)_raises_an_error": "unsupported connstring behavior",
	"TestURIOptionsSpec/tls-options.json/tlsDisableOCSPEndpointCheck_and_tlsDisableCertificateRevocationCheck_both_present_(and_true)_raises_an_error":  "unsupported connstring behavior",
	"TestURIOptionsSpec/tls-options.json/tlsDisableOCSPEndpointCheck=true_and_tlsDisableCertificateRevocationCheck=false_raises_an_error":               "unsupported connstring behavior",
	"TestURIOptionsSpec/tls-options.json/tlsDisableOCSPEndpointCheck=false_and_tlsDisableCertificateRevocationCheck=true_raises_an_error":               "unsupported connstring behavior",
	"TestURIOptionsSpec/tls-options.json/tlsDisableOCSPEndpointCheck_and_tlsDisableCertificateRevocationCheck_both_present_(and_false)_raises_an_error": "unsupported connstring behavior",
	"TestURIOptionsSpec/tls-options.json/tlsAllowInvalidCertificates_and_tlsDisableOCSPEndpointCheck_both_present_(and_true)_raises_an_error":           "unsupported connstring behavior",
	"TestURIOptionsSpec/tls-options.json/tlsAllowInvalidCertificates=true_and_tlsDisableOCSPEndpointCheck=false_raises_an_error":                        "unsupported connstring behavior",
	"TestURIOptionsSpec/tls-options.json/tlsAllowInvalidCertificates=false_and_tlsDisableOCSPEndpointCheck=true_raises_an_error":                        "unsupported connstring behavior",
	"TestURIOptionsSpec/tls-options.json/tlsAllowInvalidCertificates_and_tlsDisableOCSPEndpointCheck_both_present_(and_false)_raises_an_error":          "unsupported connstring behavior",
	"TestURIOptionsSpec/tls-options.json/tlsDisableOCSPEndpointCheck_and_tlsAllowInvalidCertificates_both_present_(and_true)_raises_an_error":           "unsupported connstring behavior",
	"TestURIOptionsSpec/tls-options.json/tlsDisableOCSPEndpointCheck=true_and_tlsAllowInvalidCertificates=false_raises_an_error":                        "unsupported connstring behavior",
	"TestURIOptionsSpec/tls-options.json/tlsDisableOCSPEndpointCheck=false_and_tlsAllowInvalidCertificates=true_raises_an_error":                        "unsupported connstring behavior",
	"TestURIOptionsSpec/tls-options.json/tlsDisableOCSPEndpointCheck_and_tlsAllowInvalidCertificates_both_present_(and_false)_raises_an_error":          "unsupported connstring behavior",
	"TestURIOptionsSpec/tls-options.json/Invalid_tlsAllowInvalidCertificates_causes_a_warning":                                                          "unsupported connstring behavior",
	"TestURIOptionsSpec/tls-options.json/tlsAllowInvalidCertificates_is_parsed_correctly":                                                               "unsupported connstring behavior",
	"TestURIOptionsSpec/tls-options.json/tlsAllowInvalidHostnames_is_parsed_correctly":                                                                  "unsupported connstring behavior",
	"TestURIOptionsSpec/tls-options.json/Invalid_tlsAllowInvalidHostnames_causes_a_warning":                                                             "unsupported connstring behavior",
	"TestURIOptionsSpec/tls-options.json/tlsInsecure_and_tlsAllowInvalidCertificates_both_present_(and_true)_raises_an_error":                           "unsupported connstring behavior",
	"TestURIOptionsSpec/tls-options.json/tlsInsecure_and_tlsAllowInvalidCertificates_both_present_(and_false)_raises_an_error":                          "unsupported connstring behavior",
	"TestURIOptionsSpec/tls-options.json/tlsAllowInvalidCertificates_and_tlsInsecure_both_present_(and_true)_raises_an_error":                           "unsupported connstring behavior",
	"TestURIOptionsSpec/tls-options.json/tlsAllowInvalidCertificates_and_tlsInsecure_both_present_(and_false)_raises_an_error":                          "unsupported connstring behavior",
	"TestURIOptionsSpec/tls-options.json/tlsInsecure_and_tlsAllowInvalidHostnames_both_present_(and_true)_raises_an_error":                              "unsupported connstring behavior",
	"TestURIOptionsSpec/tls-options.json/tlsInsecure_and_tlsAllowInvalidHostnames_both_present_(and_false)_raises_an_error":                             "unsupported connstring behavior",
	"TestURIOptionsSpec/tls-options.json/tlsAllowInvalidHostnames_and_tlsInsecure_both_present_(and_true)_raises_an_error":                              "unsupported connstring behavior",
	"TestURIOptionsSpec/tls-options.json/tlsAllowInvalidHostnames_and_tlsInsecure_both_present_(and_false)_raises_an_error":                             "unsupported connstring behavior",

	// TODO(GODRIVER-2991): make delimiting slash between hosts and options
	// optional.
	"TestConnStringSpec/valid-options.json/Missing_delimiting_slash_between_hosts_and_options": "Implement GODRIVER-2991 making delimiting slash between hosts and options optional",

	// Connstring tests violate current Go Driver behavior
	"TestURIOptionsSpec/connection-pool-options.json/maxConnecting=0_causes_a_warning":                "unsupported behavior",
	"TestURIOptionsSpec/single-threaded-options.json/Invalid_serverSelectionTryOnce_causes_a_warning": "unsupported behavior",
	"TestConnStringSpec/valid-warnings.json/Empty_integer_option_values_are_ignored":                  "SPEC-1545: unsupported behavior",
	"TestConnStringSpec/valid-warnings.json/Empty_boolean_option_value_are_ignored":                   "SPEC-1545: unsupported behavior",
	"TestConnStringSpec/valid-warnings.json/Comma_in_a_key_value_pair_causes_a_warning":               "DRIVERS-2915: unsupported behavior",

	// TODO(GODRIVER-3167): Support assertions on topologyDescriptionChangedEvent
	// in expectEvents
	"TestUnifiedSpec/specifications/source/unified-test-format/tests/valid-pass/expectedEventsForClient-topologyDescriptionChangedEvent.json/can_assert_on_values_of_newDescription_and_previousDescription_fields": "Implement GODRIVER-3161",

	// TODO(GODRIVER-3409): Regression test for "number" alias in $$type operator
	"TestUnifiedSpec/specifications/source/unified-test-format/tests/valid-pass/operator-type-number_alias.json/type_number_alias_matches_int32":      "Implement GODRIVER-3409",
	"TestUnifiedSpec/specifications/source/unified-test-format/tests/valid-pass/operator-type-number_alias.json/type_number_alias_matches_int64":      "Implement GODRIVER-3409",
	"TestUnifiedSpec/specifications/source/unified-test-format/tests/valid-pass/operator-type-number_alias.json/type_number_alias_matches_double":     "Implement GODRIVER-3409",
	"TestUnifiedSpec/specifications/source/unified-test-format/tests/valid-pass/operator-type-number_alias.json/type_number_alias_matches_decimal128": "Implement GODRIVER-3409",

	// TODO(GODRIVER-3143): Convert CRUD v1 spec tests to unified test format
	"TestUnifiedSpec/specifications/source/crud/tests/unified/bulkWrite-collation.json/BulkWrite_with_delete_operations_and_collation": "Implement GODRIVER-3143",
	"TestUnifiedSpec/specifications/source/crud/tests/unified/count.json/Count_documents_with_skip_and_limit":                          "Implement GODRIVER-3143",
	"TestUnifiedSpec/specifications/source/crud/tests/unified/findOne.json/FindOne_with_filter,_sort,_and_skip":                        "Implement GODRIVER-3143",

	// TODO(GODRIVER-2125): Allow hint for unacknowledged writes using OP_MSG when
	// supported by the server
	"TestUnifiedSpec/specifications/source/crud/tests/unified/bulkWrite-deleteMany-hint-unacknowledged.json/Unacknowledged_deleteMany_with_hint_string_on_4.4+_server":   "Implement GODRIVER-2125",
	"TestUnifiedSpec/specifications/source/crud/tests/unified/bulkWrite-deleteMany-hint-unacknowledged.json/Unacknowledged_deleteMany_with_hint_document_on_4.4+_server": "Implement GODRIVER-2125",
	"TestUnifiedSpec/specifications/source/crud/tests/unified/bulkWrite-deleteOne-hint-unacknowledged.json/Unacknowledged_deleteOne_with_hint_string_on_4.4+_server":     "Implement GODRIVER-2125",
	"TestUnifiedSpec/specifications/source/crud/tests/unified/bulkWrite-deleteOne-hint-unacknowledged.json/Unacknowledged_deleteOne_with_hint_document_on_4.4+_server":   "Implement GODRIVER-2125",
	"TestUnifiedSpec/specifications/source/crud/tests/unified/bulkWrite-replaceOne-hint-unacknowledged.json/Unacknowledged_replaceOne_with_hint_string_on_4.2+_server":   "Implement GODRIVER-2125",
	"TestUnifiedSpec/specifications/source/crud/tests/unified/bulkWrite-replaceOne-hint-unacknowledged.json/Unacknowledged_replaceOne_with_hint_document_on_4.2+_server": "Implement GODRIVER-2125",
	"TestUnifiedSpec/specifications/source/crud/tests/unified/bulkWrite-updateMany-hint-unacknowledged.json/Unacknowledged_updateMany_with_hint_string_on_4.2+_server":   "Implement GODRIVER-2125",
	"TestUnifiedSpec/specifications/source/crud/tests/unified/bulkWrite-updateMany-hint-unacknowledged.json/Unacknowledged_updateMany_with_hint_document_on_4.2+_server": "Implement GODRIVER-2125",
	"TestUnifiedSpec/specifications/source/crud/tests/unified/bulkWrite-updateOne-hint-unacknowledged.json/Unacknowledged_updateOne_with_hint_string_on_4.2+_server":     "Implement GODRIVER-2125",
	"TestUnifiedSpec/specifications/source/crud/tests/unified/bulkWrite-updateOne-hint-unacknowledged.json/Unacknowledged_updateOne_with_hint_document_on_4.2+_server":   "Implement GODRIVER-2125",

	// TODO(GODRIVER-3407): Allow drivers to set bypassDocumentValidation: false
	// on write commands
	"TestUnifiedSpec/specifications/source/crud/tests/unified/bypassDocumentValidation.json/Aggregate_with_$out_passes_bypassDocumentValidation:_false":                      "Implement GODRIVER-3407",
	"TestUnifiedSpec/specifications/source/crud/tests/unified/bypassDocumentValidation.json/BulkWrite_passes_bypassDocumentValidation:_false":                                "Implement GODRIVER-3407",
	"TestUnifiedSpec/specifications/source/crud/tests/unified/bypassDocumentValidation.json/FindOneAndReplace_passes_bypassDocumentValidation:_false":                        "Implement GODRIVER-3407",
	"TestUnifiedSpec/specifications/source/crud/tests/unified/bypassDocumentValidation.json/FindOneAndUpdate_passes_bypassDocumentValidation:_fals":                          "Implement GODRIVER-3407",
	"TestUnifiedSpec/specifications/source/crud/tests/unified/bypassDocumentValidation.json/FindOneAndUpdate_passes_bypassDocumentValidation:_false":                         "Implement GODRIVER-3407",
	"TestUnifiedSpec/specifications/source/crud/tests/unified/bypassDocumentValidation.json/InsertMany_passes_bypassDocumentValidation:_false":                               "Implement GODRIVER-3407",
	"TestUnifiedSpec/specifications/source/crud/tests/unified/bypassDocumentValidation.json/InsertOne_passes_bypassDocumentValidation:_false":                                "Implement GODRIVER-3407",
	"TestUnifiedSpec/specifications/source/crud/tests/unified/bypassDocumentValidation.json/ReplaceOne_passes_bypassDocumentValidation:_false":                               "Implement GODRIVER-3407",
	"TestUnifiedSpec/specifications/source/crud/tests/unified/bypassDocumentValidation.json/UpdateMany_passes_bypassDocumentValidation:_false":                               "Implement GODRIVER-3407",
	"TestUnifiedSpec/specifications/source/crud/tests/unified/bypassDocumentValidation.json/UpdateOne_passes_bypassDocumentValidation:_false":                                "Implement GODRIVER-3407",
	"TestUnifiedSpec/specifications/source/crud/tests/unified/deleteMany-hint-unacknowledged.json/Unacknowledged_deleteMany_with_hint_string_on_4.4+_server":                 "Implement GODRIVER-3407",
	"TestUnifiedSpec/specifications/source/crud/tests/unified/deleteMany-hint-unacknowledged.json/Unacknowledged_deleteMany_with_hint_document_on_4.4+_server":               "Implement GODRIVER-3407",
	"TestUnifiedSpec/specifications/source/crud/tests/unified/deleteOne-hint-unacknowledged.json/Unacknowledged_deleteOne_with_hint_string_on_4.4+_server":                   "Implement GODRIVER-3407",
	"TestUnifiedSpec/specifications/source/crud/tests/unified/deleteOne-hint-unacknowledged.json/Unacknowledged_deleteOne_with_hint_document_on_4.4+_server":                 "Implement GODRIVER-3407",
	"TestUnifiedSpec/specifications/source/crud/tests/unified/findOneAndDelete-hint-unacknowledged.json/Unacknowledged_findOneAndDelete_with_hint_string_on_4.4+_server":     "Implement GODRIVER-3407",
	"TestUnifiedSpec/specifications/source/crud/tests/unified/findOneAndDelete-hint-unacknowledged.json/Unacknowledged_findOneAndDelete_with_hint_document_on_4.4+_server":   "Implement GODRIVER-3407",
	"TestUnifiedSpec/specifications/source/crud/tests/unified/findOneAndReplace-hint-unacknowledged.json/Unacknowledged_findOneAndReplace_with_hint_string_on_4.4+_server":   "Implement GODRIVER-3407",
	"TestUnifiedSpec/specifications/source/crud/tests/unified/findOneAndReplace-hint-unacknowledged.json/Unacknowledged_findOneAndReplace_with_hint_document_on_4.4+_server": "Implement GODRIVER-3407",
	"TestUnifiedSpec/specifications/source/crud/tests/unified/findOneAndUpdate-hint-unacknowledged.json/Unacknowledged_findOneAndUpdate_with_hint_string_on_4.4+_server":     "Implement GODRIVER-3407",
	"TestUnifiedSpec/specifications/source/crud/tests/unified/findOneAndUpdate-hint-unacknowledged.json/Unacknowledged_findOneAndUpdate_with_hint_document_on_4.4+_server":   "Implement GODRIVER-3407",
	"TestUnifiedSpec/specifications/source/crud/tests/unified/replaceOne-hint-unacknowledged.json/Unacknowledged_replaceOne_with_hint_string_on_4.2+_server":                 "Implement GODRIVER-3407",
	"TestUnifiedSpec/specifications/source/crud/tests/unified/replaceOne-hint-unacknowledged.json/Unacknowledged_replaceOne_with_hint_document_on_4.2+_server":               "Implement GODRIVER-3407",
	"TestUnifiedSpec/specifications/source/crud/tests/unified/updateMany-hint-unacknowledged.json/Unacknowledged_updateMany_with_hint_string_on_4.2+_server":                 "Implement GODRIVER-3407",
	"TestUnifiedSpec/specifications/source/crud/tests/unified/updateMany-hint-unacknowledged.json/Unacknowledged_updateMany_with_hint_document_on_4.2+_server":               "Implement GODRIVER-3407",
	"TestUnifiedSpec/specifications/source/crud/tests/unified/updateOne-hint-unacknowledged.json/Unacknowledged_updateOne_with_hint_string_on_4.2+_server":                   "Implement GODRIVER-3407",
	"TestUnifiedSpec/specifications/source/crud/tests/unified/updateOne-hint-unacknowledged.json/Unacknowledged_updateOne_with_hint_document_on_4.2+_server":                 "Implement GODRIVER-3407",

	// TODO(GODRIVER-3392): Test that inserts and upserts respect null _id values
	"TestUnifiedSpec/specifications/source/crud/tests/unified/create-null-ids.json/inserting__id_with_type_null_via_insertOne": "Implement GODRIVER-3392",

	// TODO(GODRIVER-3395): Ensure findOne does not set batchSize=1
	"TestUnifiedSpec/specifications/source/crud/tests/unified/find.json/Find_with_batchSize_equal_to_limit": "Implement GODRIVER-3395",

	// TODO(GODRIVER-2016): Convert transactions spec tests to unified test format
	"TestUnifiedSpec/specifications/source/transactions/tests/unified/error-labels.json/do_not_add_UnknownTransactionCommitResult_label_to_MaxTimeMSExpired_inside_transaction":                        "Implement GODRIVER-2016",
	"TestUnifiedSpec/specifications/source/transactions/tests/unified/error-labels.json/do_not_add_UnknownTransactionCommitResult_label_to_MaxTimeMSExpired_inside_transactions":                       "Implement GODRIVER-2016",
	"TestUnifiedSpec/specifications/source/transactions/tests/unified/error-labels.json/add_UnknownTransactionCommitResult_label_to_MaxTimeMSExpired":                                                  "Implement GODRIVER-2016",
	"TestUnifiedSpec/specifications/source/transactions/tests/unified/error-labels.json/add_UnknownTransactionCommitResult_label_to_writeConcernError_MaxTimeMSExpired":                                "Implement GODRIVER-2016",
	"TestUnifiedSpec/specifications/source/transactions/tests/unified/retryable-commit.json/commitTransaction_applies_majority_write_concern_on_retries":                                               "Implement GODRIVER-2016",
	"TestUnifiedSpec/specifications/source/transactions/tests/unified/run-command.json/run_command_with_secondary_read_preference_in_client_option_and_primary_read_preference_in_transaction_options": "Implement GODRIVER-2016",
	"TestUnifiedSpec/specifications/source/transactions/tests/unified/transaction-options.json/transaction_options_inherited_from_client":                                                              "Implement GODRIVER-2016",
	"TestUnifiedSpec/specifications/source/transactions/tests/unified/transaction-options.json/transaction_options_inherited_from_defaultTransactionOptions":                                           "Implement GODRIVER-2016",
	"TestUnifiedSpec/specifications/source/transactions/tests/unified/transaction-options.json/startTransaction_options_override_defaults":                                                             "Implement GODRIVER-2016",
	"TestUnifiedSpec/specifications/source/transactions/tests/unified/transaction-options.json/defaultTransactionOptions_override_client_options":                                                      "Implement GODRIVER-2016",
	"TestUnifiedSpec/specifications/source/transactions/tests/unified/transaction-options.json/readConcern_local_in_defaultTransactionOptions":                                                         "Implement GODRIVER-2016",
	"TestUnifiedSpec/specifications/source/transactions/tests/unified/transaction-options.json/readPreference_inherited_from_client":                                                                   "Implement GODRIVER-2016",
	"TestUnifiedSpec/specifications/source/transactions/tests/unified/transaction-options.json/readPreference_inherited_from_defaultTransactionOptions":                                                "Implement GODRIVER-2016",
	"TestUnifiedSpec/specifications/source/transactions/tests/unified/transaction-options.json/startTransaction_overrides_readPreference":                                                              "Implement GODRIVER-2016",

	// GODRIVER-1773: This test runs a "find" with limit=4 and batchSize=3. It
	// expects batchSize values of three for the "find" and one for the
	// "getMore", but we send three for both.
	"TestUnifiedSpec/command-monitoring/find.json/A_successful_find_event_with_a_getmore_and_the_server_kills_the_cursor_(<=_4.4)": "See GODRIVER-1773",

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
	"TestUnifiedSpec/specifications/source/command-logging-and-monitoring/tests/monitoring/find.json/A_successful_find_with_options":                                  "Uses unsupported maxTimeMS",
	"TestUnifiedSpec/specifications/source/crud/tests/unified/estimatedDocumentCount.json/estimatedDocumentCount_with_maxTimeMS":                                      "Uses unsupported maxTimeMS",
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

	// TODO(GODRIVER-3146): Convert retryable reads spec tests to unified test
	// format.
	"TestUnifiedSpec/specifications/source/retryable-reads/tests/unified/listCollectionObjects-serverErrors.json/ListCollectionObjects_succeeds_after_InterruptedAtShutdown":                    "Implement GODRIVER-3146",
	"TestUnifiedSpec/specifications/source/retryable-reads/tests/unified/listCollectionObjects-serverErrors.json/ListCollectionObjects_succeeds_after_InterruptedDueToReplStateChange":          "Implement GODRIVER-3146",
	"TestUnifiedSpec/specifications/source/retryable-reads/tests/unified/listCollectionObjects-serverErrors.json/ListCollectionObjects_succeeds_after_NotWritablePrimary":                       "Implement GODRIVER-3146",
	"TestUnifiedSpec/specifications/source/retryable-reads/tests/unified/listCollectionObjects-serverErrors.json/ListCollectionObjects_succeeds_after_NotPrimaryNoSecondaryOk":                  "Implement GODRIVER-3146",
	"TestUnifiedSpec/specifications/source/retryable-reads/tests/unified/listCollectionObjects-serverErrors.json/ListCollectionObjects_succeeds_after_NotPrimaryOrSecondary":                    "Implement GODRIVER-3146",
	"TestUnifiedSpec/specifications/source/retryable-reads/tests/unified/listCollectionObjects-serverErrors.json/ListCollectionObjects_succeeds_after_PrimarySteppedDown":                       "Implement GODRIVER-3146",
	"TestUnifiedSpec/specifications/source/retryable-reads/tests/unified/listCollectionObjects-serverErrors.json/ListCollectionObjects_succeeds_after_ShutdownInProgress":                       "Implement GODRIVER-3146",
	"TestUnifiedSpec/specifications/source/retryable-reads/tests/unified/listCollectionObjects-serverErrors.json/ListCollectionObjects_succeeds_after_HostNotFound":                             "Implement GODRIVER-3146",
	"TestUnifiedSpec/specifications/source/retryable-reads/tests/unified/listCollectionObjects-serverErrors.json/ListCollectionObjects_succeeds_after_HostUnreachable":                          "Implement GODRIVER-3146",
	"TestUnifiedSpec/specifications/source/retryable-reads/tests/unified/listCollectionObjects-serverErrors.json/ListCollectionObjects_succeeds_after_NetworkTimeout":                           "Implement GODRIVER-3146",
	"TestUnifiedSpec/specifications/source/retryable-reads/tests/unified/listCollectionObjects-serverErrors.json/ListCollectionObjects_succeeds_after_SocketException":                          "Implement GODRIVER-3146",
	"TestUnifiedSpec/specifications/source/retryable-reads/tests/unified/listCollectionObjects-serverErrors.json/ListCollectionObjects_fails_after_two_NotWritablePrimary_errors":               "Implement GODRIVER-3146",
	"TestUnifiedSpec/specifications/source/retryable-reads/tests/unified/listCollectionObjects-serverErrors.json/ListCollectionObjects_fails_after_NotWritablePrimary_when_retryReads_is_false": "Implement GODRIVER-3146",
	"TestUnifiedSpec/specifications/source/retryable-reads/tests/unified/listCollectionObjects.json/ListCollectionObjects_succeeds_on_first_attempt":                                            "Implement GODRIVER-3146",
	"TestUnifiedSpec/specifications/source/retryable-reads/tests/unified/listCollectionObjects.json/ListCollectionObjects_succeeds_on_second_attempt":                                           "Implement GoDRIVER-3146",
	"TestUnifiedSpec/specifications/source/retryable-reads/tests/unified/listCollectionObjects.json/ListCollectionObjects_fails_on_first_attempt":                                               "Implement GoDRIVER-3146",
	"TestUnifiedSpec/specifications/source/retryable-reads/tests/unified/listCollectionObjects.json/ListCollectionObjects_fails_on_second_attempt":                                              "Implement GoDRIVER-3146",
	"TestUnifiedSpec/specifications/source/retryable-reads/tests/unified/listDatabaseObjects-serverErrors.json/ListDatabaseObjects_succeeds_after_InterruptedAtShutdown":                        "Implement GODRIVER-3146",
	"TestUnifiedSpec/specifications/source/retryable-reads/tests/unified/listDatabaseObjects-serverErrors.json/ListDatabaseObjects_succeeds_after_InterruptedDueToReplStateChange":              "Implement GODRIVER-3146",
	"TestUnifiedSpec/specifications/source/retryable-reads/tests/unified/listDatabaseObjects-serverErrors.json/ListDatabaseObjects_succeeds_after_NotWritablePrimary":                           "Implement GODRIVER-3146",
	"TestUnifiedSpec/specifications/source/retryable-reads/tests/unified/listDatabaseObjects-serverErrors.json/ListDatabaseObjects_succeeds_after_NotPrimaryNoSecondaryOk":                      "Implement GODRIVER-3146",
	"TestUnifiedSpec/specifications/source/retryable-reads/tests/unified/listDatabaseObjects-serverErrors.json/ListDatabaseObjects_succeeds_after_NotPrimaryOrSecondary":                        "Implement GODRIVER-3146",
	"TestUnifiedSpec/specifications/source/retryable-reads/tests/unified/listDatabaseObjects-serverErrors.json/ListDatabaseObjects_succeeds_after_PrimarySteppedDown":                           "Implement GODRIVER-3146",
	"TestUnifiedSpec/specifications/source/retryable-reads/tests/unified/listDatabaseObjects-serverErrors.json/ListDatabaseObjects_succeeds_after_ShutdownInProgress":                           "Implement GODRIVER-3146",
	"TestUnifiedSpec/specifications/source/retryable-reads/tests/unified/listDatabaseObjects-serverErrors.json/ListDatabaseObjects_succeeds_after_HostNotFound":                                 "Implement GODRIVER-3146",
	"TestUnifiedSpec/specifications/source/retryable-reads/tests/unified/listDatabaseObjects-serverErrors.json/ListDatabaseObjects_succeeds_after_HostUnreachable":                              "Implement GODRIVER-3146",
	"TestUnifiedSpec/specifications/source/retryable-reads/tests/unified/listDatabaseObjects-serverErrors.json/ListDatabaseObjects_succeeds_after_NetworkTimeout":                               "Implement GODRIVER-3146",
	"TestUnifiedSpec/specifications/source/retryable-reads/tests/unified/listDatabaseObjects-serverErrors.json/ListDatabaseObjects_succeeds_after_SocketException":                              "Implement GODRIVER-3146",
	"TestUnifiedSpec/specifications/source/retryable-reads/tests/unified/listDatabaseObjects-serverErrors.json/ListDatabaseObjects_fails_after_two_NotWritablePrimary_errors":                   "Implement GODRIVER-3146",
	"TestUnifiedSpec/specifications/source/retryable-reads/tests/unified/listDatabaseObjects-serverErrors.json/ListDatabaseObjects_fails_after_NotWritablePrimary_when_retryReads_is_false":     "Implement GODRIVER-3146",
	"TestUnifiedSpec/specifications/source/retryable-reads/tests/unified/listDatabaseObjects.json/ListDatabaseObjects_succeeds_on_first_attempt":                                                "Implement GODRIVER-3146",
	"TestUnifiedSpec/specifications/source/retryable-reads/tests/unified/listDatabaseObjects.json/ListDatabaseObjects_succeeds_on_second_attempt":                                               "Implement GODRIVER-3146",
	"TestUnifiedSpec/specifications/source/retryable-reads/tests/unified/listDatabaseObjects.json/ListDatabaseObjects_fails_on_first_attempt":                                                   "Implement GODRIVER-3146",
	"TestUnifiedSpec/specifications/source/retryable-reads/tests/unified/listDatabaseObjects.json/ListDatabaseObjects_fails_on_second_attempt":                                                  "Implement GODRIVER-3146",
	"TestUnifiedSpec/specifications/source/retryable-reads/tests/unified/mapReduce.json/MapReduce_succeeds_with_retry_on":                                                                       "Implement GODRIVER-3146",
	"TestUnifiedSpec/specifications/source/retryable-reads/tests/unified/mapReduce.json/MapReduce_fails_with_retry_on":                                                                          "Implement GODRIVER-3146",
	"TestUnifiedSpec/specifications/source/retryable-reads/tests/unified/mapReduce.json/MapReduce_fails_with_retry_off":                                                                         "Implement GODRIVER-3146",

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
}

// CheckSkip checks if the fully-qualified test name matches a skipped test
// name. If the test name matches, the reason is logged and the test is skipped.
func CheckSkip(t *testing.T) {
	if reason := skipTests[t.Name()]; reason != "" {
		t.Skipf("Skipping due to known failure: %q", reason)
	}
}
