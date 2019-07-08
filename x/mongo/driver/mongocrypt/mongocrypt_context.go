// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongocrypt

// #include <mongocrypt.h>
import "C"
import (
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

// MongoCryptContext represents a mongocrypt_ctx_t handle
type MongoCryptContext struct {
	wrapped *C.mongocrypt_ctx_t
}

// newMongoCryptContext creates a MongoCryptContext wrapper around the given C type.
func newMongoCryptContext(wrapped *C.mongocrypt_ctx_t) *MongoCryptContext {
	return &MongoCryptContext{
		wrapped: wrapped,
	}
}

// State returns the current State of the MongoCryptContext.
func (mc *MongoCryptContext) State() State {
	return State(int(C.mongocrypt_ctx_state(mc.wrapped)))
}

// NextOperation gets the document for the next database operation to run.
func (mc *MongoCryptContext) NextOperation() (bsoncore.Document, error) {
	opDocBinary := newBinary() // out param for mongocrypt_ctx_mongo_op to fill in operation
	defer opDocBinary.close()

	if ok := C.mongocrypt_ctx_mongo_op(mc.wrapped, opDocBinary.wrapped); !ok {
		return nil, mc.createErrorFromStatus()
	}
	return opDocBinary.toBytes(), nil
}

// AddOperationResult feeds the result of a database operation to mongocrypt.
func (mc *MongoCryptContext) AddOperationResult(result bsoncore.Document) error {
	resultBinary := newBinaryFromBytes(result)
	defer resultBinary.close()

	if ok := C.mongocrypt_ctx_mongo_feed(mc.wrapped, resultBinary.wrapped); !ok {
		return mc.createErrorFromStatus()
	}
	return nil
}

// CompleteOperation signals a database operation has been completed.
func (mc *MongoCryptContext) CompleteOperation() error {
	if ok := C.mongocrypt_ctx_mongo_done(mc.wrapped); !ok {
		return mc.createErrorFromStatus()
	}
	return nil
}

// NextKmsContext returns the next MongoCryptKmsContext, or nil if there are no more.
func (mc *MongoCryptContext) NextKmsContext() *MongoCryptKmsContext {
	ctx := C.mongocrypt_ctx_next_kms_ctx(mc.wrapped)
	if ctx == nil {
		return nil
	}
	return newMongoCryptKmsContext(ctx)
}

// FinishKmsContexts signals that all KMS contexts have been completed.
func (mc *MongoCryptContext) FinishKmsContexts() error {
	if ok := C.mongocrypt_ctx_kms_done(mc.wrapped); !ok {
		return mc.createErrorFromStatus()
	}
	return nil
}

// Finish performs the final operations for the context and returns the resulting document.
func (mc *MongoCryptContext) Finish() (bsoncore.Document, error) {
	docBinary := newBinary() // out param for mongocrypt_ctx_finalize to fill in resulting document
	defer docBinary.close()

	if ok := C.mongocrypt_ctx_finalize(mc.wrapped, docBinary.wrapped); !ok {
		return nil, mc.createErrorFromStatus()
	}
	return docBinary.toBytes(), nil
}

// Close cleans up any resources associated with the given MongoCryptContext instance.
func (mc *MongoCryptContext) Close() {
	C.mongocrypt_ctx_destroy(mc.wrapped)
}

// createErrorFromStatus creates a new MongoCryptError based on the status of the MongoCrypt instance.
func (mc *MongoCryptContext) createErrorFromStatus() error {
	status := C.mongocrypt_status_new()
	defer C.mongocrypt_status_destroy(status)
	C.mongocrypt_ctx_status(mc.wrapped, status)
	return errorFromStatus(status)
}
