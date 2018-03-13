// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

import (
	"time"

	"github.com/mongodb/mongo-go-driver/mongo/connstring"
)

// ClientOptioner is the interface implemented by types that can be used as options for constructing a
// mongo.Client.
type ClientOptioner interface {
	ClientOption(*connstring.ConnString)
}

var (
	_ ClientOptioner = (*AppName)(nil)
	_ ClientOptioner = (*AuthMechanism)(nil)
	_ ClientOptioner = (*AuthMechanismProperties)(nil)
	_ ClientOptioner = (*AuthSource)(nil)
	_ ClientOptioner = (*ConnectTimeout)(nil)
	_ ClientOptioner = (*HeartbeatInterval)(nil)
	_ ClientOptioner = (*Hosts)(nil)
	_ ClientOptioner = (*Journal)(nil)
	_ ClientOptioner = (*LocalThreshold)(nil)
	_ ClientOptioner = (*MaxConnIdleTime)(nil)
	_ ClientOptioner = (*MaxConnsPerHost)(nil)
	_ ClientOptioner = (*MaxIdleConnsPerHost)(nil)
	_ ClientOptioner = (*Password)(nil)
	_ ClientOptioner = (*ReadConcernLevel)(nil)
	_ ClientOptioner = (*ReadPreference)(nil)
	_ ClientOptioner = (*ReadPreferenceTagSets)(nil)
	_ ClientOptioner = (*ReplicaSet)(nil)
	_ ClientOptioner = (*ServerSelectionTimeout)(nil)
	_ ClientOptioner = (*Single)(nil)
	_ ClientOptioner = (*SocketTimeout)(nil)
	_ ClientOptioner = (*SSL)(nil)
	_ ClientOptioner = (*SSLClientCertificateKeyFile)(nil)
	_ ClientOptioner = (*SSLInsecure)(nil)
	_ ClientOptioner = (*SSLCaFile)(nil)
	_ ClientOptioner = (*WString)(nil)
	_ ClientOptioner = (*WNumber)(nil)
	_ ClientOptioner = (*Username)(nil)
	_ ClientOptioner = (*WTimeout)(nil)
)

// AppName is for internal use.
type AppName string

// ClientOption implements the ClientOption interface.
func (an AppName) ClientOption(cs *connstring.ConnString) {
	if len(cs.AppName) == 0 {
		cs.AppName = string(an)
	}
}

// AuthMechanism is for internal use.
type AuthMechanism string

// ClientOption implements the ClientOption interface.
func (am AuthMechanism) ClientOption(cs *connstring.ConnString) {
	if len(cs.AuthMechanism) == 0 {
		cs.AuthMechanism = string(am)
	}
}

// AuthMechanismProperties is for internal use.
type AuthMechanismProperties map[string]string

// ClientOption implements the ClientOption interface.
func (amp AuthMechanismProperties) ClientOption(cs *connstring.ConnString) {
	if cs.AuthMechanismProperties == nil {
		cs.AuthMechanismProperties = amp
	}
}

// AuthSource is for internal use.
type AuthSource string

// ClientOption implements the ClientOption interface.
func (as AuthSource) ClientOption(cs *connstring.ConnString) {
	if len(cs.AuthSource) == 0 {
		cs.AuthSource = string(as)
	}
}

// ConnectTimeout is for internal use.
type ConnectTimeout time.Duration

// ClientOption implements the ClientOption interface.
func (ct ConnectTimeout) ClientOption(cs *connstring.ConnString) {
	if !cs.ConnectTimeoutSet {
		cs.ConnectTimeout = time.Duration(ct)
		cs.ConnectTimeoutSet = true
	}
}

// HeartbeatInterval is for internal use.
type HeartbeatInterval time.Duration

// ClientOption implements the ClientOption interface.
func (hi HeartbeatInterval) ClientOption(cs *connstring.ConnString) {
	if !cs.HeartbeatIntervalSet {
		cs.HeartbeatInterval = time.Duration(hi)
		cs.HeartbeatIntervalSet = true
	}
}

// Hosts is for internal use.
type Hosts []string

// ClientOption implements the ClientOption interface.
func (h Hosts) ClientOption(cs *connstring.ConnString) {
	if cs.Hosts == nil {
		cs.Hosts = []string(h)
	}
}

// Journal is for internal use.
type Journal bool

// ClientOption implements the ClientOption interface.
func (j Journal) ClientOption(cs *connstring.ConnString) {
	if !cs.JSet {
		cs.J = bool(j)
		cs.JSet = true
	}
}

// LocalThreshold is for internal use.
type LocalThreshold time.Duration

// ClientOption implements the ClientOption interface.
func (lt LocalThreshold) ClientOption(cs *connstring.ConnString) {
	if !cs.LocalThresholdSet {
		cs.LocalThreshold = time.Duration(lt)
		cs.LocalThresholdSet = true
	}
}

// MaxConnIdleTime is for internal use.
type MaxConnIdleTime time.Duration

// ClientOption implements the ClientOption interface.
func (mcit MaxConnIdleTime) ClientOption(cs *connstring.ConnString) {
	if !cs.MaxConnIdleTimeSet {
		cs.MaxConnIdleTime = time.Duration(mcit)
		cs.MaxConnIdleTimeSet = true
	}
}

// MaxConnsPerHost is for internal use.
type MaxConnsPerHost uint16

// ClientOption implements the ClientOption interface.
func (mcph MaxConnsPerHost) ClientOption(cs *connstring.ConnString) {
	if !cs.MaxConnsPerHostSet {
		cs.MaxConnsPerHost = uint16(mcph)
		cs.MaxConnsPerHostSet = true
	}
}

// MaxIdleConnsPerHost is for internal use.
type MaxIdleConnsPerHost uint16

// ClientOption implements the ClientOption interface.
func (micph MaxIdleConnsPerHost) ClientOption(cs *connstring.ConnString) {
	if !cs.MaxIdleConnsPerHostSet {
		cs.MaxIdleConnsPerHost = uint16(micph)
		cs.MaxIdleConnsPerHostSet = true
	}
}

// Password is for internal use.
type Password string

// ClientOption implements the ClientOption interface.
func (p Password) ClientOption(cs *connstring.ConnString) {
	if !cs.PasswordSet {
		cs.Password = string(p)
		cs.PasswordSet = true
	}
}

// ReadConcernLevel is for internal use.
type ReadConcernLevel string

// ClientOption implements the ClientOption interface.
func (rcl ReadConcernLevel) ClientOption(cs *connstring.ConnString) {
	if len(rcl) == 0 {
		cs.ReadConcernLevel = string(rcl)
	}
}

// ReadPreference is for internal use.
type ReadPreference string

// ClientOption implements the ClientOption interface.
func (rp ReadPreference) ClientOption(cs *connstring.ConnString) {
	if len(rp) == 0 {
		cs.ReadPreference = string(rp)
	}
}

// ReadPreferenceTagSets is for internal use.
type ReadPreferenceTagSets []map[string]string

// ClientOption implements the ClientOption interface.
func (rpts ReadPreferenceTagSets) ClientOption(cs *connstring.ConnString) {
	if cs.ReadPreferenceTagSets == nil {
		cs.ReadPreferenceTagSets = []map[string]string(rpts)
	}
}

// ReplicaSet is for internal use.
type ReplicaSet string

// ClientOption implements the ClientOption interface.
func (rs ReplicaSet) ClientOption(cs *connstring.ConnString) {
	if len(cs.ReplicaSet) == 0 {
		cs.ReplicaSet = string(rs)
	}
}

// ServerSelectionTimeout is for internal use.
type ServerSelectionTimeout time.Duration

// ClientOption implements the ClientOption interface.
func (sst ServerSelectionTimeout) ClientOption(cs *connstring.ConnString) {
	if !cs.ServerSelectionTimeoutSet {
		cs.ServerSelectionTimeout = time.Duration(sst)
		cs.ServerSelectionTimeoutSet = true
	}
}

// Single is for internal use.
type Single bool

// ClientOption implements the ClientOption interface.
func (s Single) ClientOption(cs *connstring.ConnString) {
	if !cs.ConnectSet {
		if s {
			cs.Connect = connstring.SingleConnect
		} else {
			cs.Connect = connstring.AutoConnect
		}

		cs.ConnectSet = true
	}
}

// SocketTimeout is for internal use.
type SocketTimeout time.Duration

// ClientOption implements the ClientOption interface.
func (st SocketTimeout) ClientOption(cs *connstring.ConnString) {
	if !cs.SocketTimeoutSet {
		cs.SocketTimeout = time.Duration(st)
		cs.SocketTimeoutSet = true
	}
}

// SSL is for internal use.
type SSL bool

// ClientOption implements the ClientOption interface.
func (ssl SSL) ClientOption(cs *connstring.ConnString) {
	if !cs.SSLSet {
		cs.SSL = bool(ssl)
		cs.SSLSet = true
	}
}

// SSLClientCertificateKeyFile is for internal use.
type SSLClientCertificateKeyFile string

// ClientOption implements the ClientOption interface.
func (scckf SSLClientCertificateKeyFile) ClientOption(cs *connstring.ConnString) {
	if !cs.SSLClientCertificateKeyFileSet {
		cs.SSLClientCertificateKeyFile = string(scckf)
		cs.SSLClientCertificateKeyFileSet = true
	}
}

// SSLInsecure is for internal use.
type SSLInsecure bool

// ClientOption implements the ClientOption interface.
func (si SSLInsecure) ClientOption(cs *connstring.ConnString) {
	if !cs.SSLInsecureSet {
		cs.SSLInsecure = bool(si)
		cs.SSLInsecureSet = true
	}
}

// SSLCaFile is for internal use.
type SSLCaFile string

// ClientOption implements the ClientOption interface.
func (scckf SSLCaFile) ClientOption(cs *connstring.ConnString) {
	if !cs.SSLCaFileSet {
		cs.SSLCaFile = string(scckf)
		cs.SSLCaFileSet = true
	}
}

// WString is for internal use.
type WString string

// ClientOption implements the ClientOption interface.
func (w WString) ClientOption(cs *connstring.ConnString) {
	if !cs.WNumberSet && len(cs.WString) == 0 {
		cs.WString = string(w)
	}
}

// WNumber is for internal use.
type WNumber int

// ClientOption implements the ClientOption interface.
func (w WNumber) ClientOption(cs *connstring.ConnString) {
	if !cs.WNumberSet && len(cs.WString) == 0 {
		cs.WNumber = int(w)
		cs.WNumberSet = true
	}
}

// Username is for internal use.
type Username string

// ClientOption implements the ClientOption interface.
func (u Username) ClientOption(cs *connstring.ConnString) {
	if len(cs.Username) == 0 {
		cs.Username = string(u)
	}
}

// WTimeout is for internal use.
type WTimeout time.Duration

// ClientOption implements the ClientOption interface.
func (st WTimeout) ClientOption(cs *connstring.ConnString) {
	if !cs.WTimeoutSet && !cs.WTimeoutSetFromOption {
		cs.WTimeout = time.Duration(st)
		cs.WTimeoutSetFromOption = true
	}
}
