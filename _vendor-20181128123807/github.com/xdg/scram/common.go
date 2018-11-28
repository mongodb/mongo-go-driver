// Copyright 2018 by David A. Golden. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package scram ...
package scram

import (
	"crypto/hmac"
	"crypto/rand"
	"encoding/base64"
	"strings"
)

// NonceGeneratorFcn defines a function that returns a string high-quality
// random printable ASCII characters EXCLUDING the comma (',') character.
// The default nonce generator provides Base64 encoding of 24 bytes from
// crypt.rand.
type NonceGeneratorFcn func() string

// DerivedKeys ...
type DerivedKeys struct {
	ClientKey []byte
	StoredKey []byte
	ServerKey []byte
}

// KeyFactors ...
// Salt is base64 encoded so that KeyFactors can be used as a map key
// for cached credentials.
type KeyFactors struct {
	Salt  string
	Iters int
}

// StoredCredentials ...
type StoredCredentials struct {
	KeyFactors
	StoredKey []byte
	ServerKey []byte
}

// CredentialLookup ...
type CredentialLookup func(string) (StoredCredentials, error)

// AuthProxyLookup ...
type AuthProxyLookup func(string, string) bool

func defaultNonceGenerator() string {
	raw := make([]byte, 24)
	nonce := make([]byte, base64.StdEncoding.EncodedLen(len(raw)))
	rand.Read(raw)
	base64.StdEncoding.Encode(nonce, raw)
	return string(nonce)
}

func encodeName(s string) string {
	return strings.Replace(strings.Replace(s, "=", "=3D", -1), ",", "=2C", -1)
}

func decodeName(s string) (string, error) {
	// TODO Check for = not followed by 2C or 3D
	return strings.Replace(strings.Replace(s, "=2C", ",", -1), "=3D", "=", -1), nil
}

func computeHash(hg HashGeneratorFcn, b []byte) []byte {
	h := hg()
	h.Write(b)
	return h.Sum(nil)
}

func computeHMAC(hg HashGeneratorFcn, key, data []byte) []byte {
	mac := hmac.New(hg, key)
	mac.Write(data)
	return mac.Sum(nil)
}

func xorBytes(a, b []byte) []byte {
	// TODO check a & b are same length, or just xor to smallest
	xor := make([]byte, len(a))
	for i := range a {
		xor[i] = a[i] ^ b[i]
	}
	return xor
}
