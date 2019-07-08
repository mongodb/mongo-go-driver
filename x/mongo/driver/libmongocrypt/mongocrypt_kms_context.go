// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package libmongocrypt

// #include <mongocrypt.h>
import "C"

// MongoCryptKmsContext represents a mongocrypt_kms_ctx_t handle.
type MongoCryptKmsContext struct {
	wrapped *C.mongocrypt_kms_ctx_t
}

// newMongoCryptKmsContext creates a MongoCryptKmsContext wrapper around the given C type.
func newMongoCryptKmsContext(wrapped *C.mongocrypt_kms_ctx_t) *MongoCryptKmsContext {
	return &MongoCryptKmsContext{
		wrapped: wrapped,
	}
}

// HostName gets the host name of the KMS.
func (mkc *MongoCryptKmsContext) HostName() (string, error) {
	var hostname *C.char // out param for mongocrypt function to fill in hostname
	if ok := C.mongocrypt_kms_ctx_endpoint(mkc.wrapped, &hostname); !ok {
		return "", mkc.createErrorFromStatus()
	}
	return C.GoString(hostname), nil
}

// Message returns the message to send to the KMS.
func (mkc *MongoCryptKmsContext) Message() ([]byte, error) {
	msgBinary := newBinary()
	defer msgBinary.close()

	if ok := C.mongocrypt_kms_ctx_message(mkc.wrapped, msgBinary.wrapped); !ok {
		return nil, mkc.createErrorFromStatus()
	}
	return msgBinary.toBytes(), nil
}

// BytesNeeded returns the number of bytes that should be received from the KMS.
// After sending the message to the KMS, this message should be called in a loop until the number returned is 0.
func (mkc *MongoCryptKmsContext) BytesNeeded() int32 {
	return int32(C.mongocrypt_kms_ctx_bytes_needed(mkc.wrapped))
}

// FeedResponse feeds the bytes received from the KMS to libmongocrypt.
func (mkc *MongoCryptKmsContext) FeedResponse(response []byte) error {
	responseBinary := newBinaryFromBytes(response)
	defer responseBinary.close()

	if ok := C.mongocrypt_kms_ctx_feed(mkc.wrapped, responseBinary.wrapped); !ok {
		return mkc.createErrorFromStatus()
	}
	return nil
}

// createErrorFromStatus creates a new MongoCryptError from the status of the MongoCryptKmsContext instance.
func (mkc *MongoCryptKmsContext) createErrorFromStatus() error {
	status := C.mongocrypt_status_new()
	defer C.mongocrypt_status_destroy(status)
	C.mongocrypt_kms_ctx_status(mkc.wrapped, status)
	return mongoCryptErrorFromStatus(status)
}
