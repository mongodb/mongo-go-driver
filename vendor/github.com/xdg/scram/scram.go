// Copyright 2018 by David A. Golden. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package scram ...
package scram

import (
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
	"hash"

	"github.com/xdg/stringprep"
)

// HashGeneratorFcn ...
type HashGeneratorFcn func() hash.Hash

// SHA1 ...
var SHA1 HashGeneratorFcn = func() hash.Hash { return sha1.New() }

// SHA256 ...
var SHA256 HashGeneratorFcn = func() hash.Hash { return sha256.New() }

// NewClient ...
func (f HashGeneratorFcn) NewClient(username, password, authID string) (*Client, error) {
	var userprep, passprep, authprep string
	var err error

	if userprep, err = stringprep.SASLprep.Prepare(username); err != nil {
		return nil, fmt.Errorf("Error SASLprepping username '%s': %v", username, err)
	}
	if passprep, err = stringprep.SASLprep.Prepare(password); err != nil {
		return nil, fmt.Errorf("Error SASLprepping password '%s': %v", password, err)
	}
	if authprep, err = stringprep.SASLprep.Prepare(authID); err != nil {
		return nil, fmt.Errorf("Error SASLprepping authID '%s': %v", authID, err)
	}

	return newClient(userprep, passprep, authprep, f), nil
}

// NewClientUnprepped ...
func (f HashGeneratorFcn) NewClientUnprepped(username, password, authID string) (*Client, error) {
	return newClient(username, password, authID, f), nil
}

// NewServer ...
func (f HashGeneratorFcn) NewServer(cl CredentialLookup) (*Server, error) {
	return newServer(cl, f)
}
