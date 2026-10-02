// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package xoptions is intended for internal use only. It is made available to
// facilitate use cases that require access to internal MongoDB driver
// functionality and state. The API of this package is not stable and there is
// no backward compatibility guarantee.
//
// WARNING: THIS PACKAGE IS EXPERIMENTAL AND MAY BE MODIFIED OR REMOVED WITHOUT
// NOTICE! USE WITH EXTREME CAUTION!
//
// The SetInternal*Options functions attach internal options to an options
// builder by key. An unsupported key, or a value whose type does not match the
// key, returns an error. Two keys are defined for the operation setters:
//
//	"rawData"          bool    Sets the rawData command field on servers that
//	                           support it. Ignored on older servers.
//	"addCommandFields" bson.D  Appends the given elements to the top level of
//	                           the command document sent to the server.
//
// Not every setter supports both keys. Call the setter and check the returned
// error rather than assuming a key is available. SetInternalClientOptions is an
// exception to the above; it configures the client itself and accepts a
// different set of keys.
//
// # Risks of addCommandFields
//
// The elements of the supplied bson.D are spliced verbatim into the top level
// of the command document, after the operation has written its own fields and
// before the driver appends the fields it manages. Callers, not the driver, are
// responsible for the correctness of what they add.
//
// The driver does not validate keys. Encoding rejects only keys containing a
// NUL byte; keys containing dots or beginning with "$" are passed through
// unchanged.
//
// The driver does not deduplicate keys. A key that collides with an existing
// command field produces a duplicate element rather than replacing the original
// value. Collisions are possible both with fields the operation itself writes,
// such as the filter, sort, let, or rawData, and with fields the driver appends
// afterward, including readConcern, writeConcern, lsid, txnNumber,
// startTransaction, autocommit, $clusterTime, apiVersion, apiStrict,
// apiDeprecationErrors, maxTimeMS, $db, $readPreference, and the batch array of
// a bulk operation. The behavior of the server when it receives a command with
// duplicate fields is server-defined and is not specified here.
//
// The driver does not redact these fields. Redaction is selected by command
// name, and none of the commands that accept addCommandFields are redacted, so
// the supplied elements appear in full in the command of a CommandStartedEvent
// and in driver command logging. Do not place credentials or other secrets in
// addCommandFields.
//
// Because BSON is a length-prefixed binary format, supplied content cannot
// escape the document it is written into. The residual risk is semantic: keys
// derived from untrusted input can duplicate or shadow fields the driver
// manages, and values carry the same operator-injection considerations as any
// other user-supplied query document. Treat any untrusted input used to build
// an addCommandFields document with the same care as an untrusted filter, and
// prefer constructing keys from a fixed allowlist.
//
// One operation applies the fields narrowly: Collection.Drop on a collection
// with encrypted fields applies them only to the drop of the data collection,
// not to the drops of the associated encryption state collections.
package xoptions
