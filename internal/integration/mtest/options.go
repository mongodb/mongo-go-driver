// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mtest

import (
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// TopologyKind describes the topology that a test is run on.
type TopologyKind string

// These constants specify valid values for TopologyKind
const (
	ReplicaSet   TopologyKind = "replicaset"
	Sharded      TopologyKind = "sharded"
	Single       TopologyKind = "single"
	LoadBalanced TopologyKind = "load-balanced"
	// ShardedReplicaSet is a special case of sharded that requires each shard to be a replica set rather than a
	// standalone server.
	ShardedReplicaSet TopologyKind = "sharded-replicaset"
)

// ClientType specifies the type of Client that should be created for a test.
type ClientType int

// These constants specify valid values for ClientType
const (
	// Default specifies a client to the connection string in the MONGODB_URI env variable with command monitoring
	// enabled.
	Default ClientType = iota
	// Pinned specifies a client that is pinned to a single mongos in a sharded cluster.
	Pinned
	// Mock specifies a client that communicates with a mock deployment.
	Mock
	// Proxy specifies a client that proxies messages to the server and also stores parsed copies. The proxied
	// messages can be retrieved via T.GetProxiedMessages or T.GetRawProxiedMessages.
	Proxy
)

var falseBool = false

// CSFLEOptions holds configuration for Client-Side Field Level Encryption
// (CSFLE).
type CSFLEOptions struct {
	MinVer string `bson:"minLibmongocryptVersion"`
}

// CSFLE models the runOnRequirements.csfle field in Unified Test Format tests.
//
// The csfle field accepts either:
//   - a boolean: true enables CSFLE with no options; false disables CSFLE
//     (Options is nil).
//   - an object: Options is populated from the document and Boolean is set to
//     false.
type CSFLE struct {
	Boolean bool
	Options *CSFLEOptions
}

// UnmarshalBSON implements custom BSON unmarshalling for CSFLE, accepting
// either a boolean or an embedded document. If a document is provided, Options
// is set and Boolean is true. If a boolean is provided, Boolean is set and
// Options is nil.
func (csfle *CSFLE) UnmarshalBSON(data []byte) error {
	embRawValue := bson.RawValue{Type: bson.TypeEmbeddedDocument, Value: data}
	if err := embRawValue.Unmarshal(&csfle.Options); err == nil {
		csfle.Boolean = true

		return nil
	}

	rawValue := bson.RawValue{Type: bson.TypeBoolean, Value: data}
	if b, ok := rawValue.BooleanOK(); ok {
		csfle.Boolean = b
		csfle.Options = nil

		return nil
	}

	return fmt.Errorf("error unmarshalling CSFLE: %s", data)
}

// RunOnBlock describes a constraint for a test.
type RunOnBlock struct {
	MinServerVersion string                   `bson:"minServerVersion"`
	MaxServerVersion string                   `bson:"maxServerVersion"`
	Topology         []TopologyKind           `bson:"topology"`
	Serverless       string                   `bson:"serverless"`
	ServerParameters map[string]bson.RawValue `bson:"serverParameters"`
	Auth             *bool                    `bson:"auth"`
	AuthEnabled      *bool                    `bson:"authEnabled"`
	CSFLE            *CSFLE                   `bson:"csfleConfiguration"`
}

// CSFLEEnabled returns true if CSFLE support is explicitly required in the
// "runOnRequirement" block. It returns false if the CSFLE requirement is
// unspecified or explicitly false.
func (r *RunOnBlock) CSFLEEnabled() bool {
	return r.CSFLE != nil && (r.CSFLE.Boolean || r.CSFLE.Options != nil)
}

// CSFLEDisabled returns true if CSFLE support is explicitly forbidden in the
// "runOnRequirement" block. It returns false if the CSFLE requirement is
// unspecified or explicitly true.
func (r *RunOnBlock) CSFLEDisabled() bool {
	return r.CSFLE != nil && !r.CSFLE.Boolean
}

// UnmarshalBSON implements custom BSON unmarshalling behavior for RunOnBlock because some test formats use the
// "topology" key while the unified test format uses "topologies".
func (r *RunOnBlock) UnmarshalBSON(data []byte) error {
	var temp struct {
		MinServerVersion string                   `bson:"minServerVersion"`
		MaxServerVersion string                   `bson:"maxServerVersion"`
		Topology         []TopologyKind           `bson:"topology"`
		Topologies       []TopologyKind           `bson:"topologies"`
		Serverless       string                   `bson:"serverless"`
		ServerParameters map[string]bson.RawValue `bson:"serverParameters"`
		Auth             *bool                    `bson:"auth"`
		AuthEnabled      *bool                    `bson:"authEnabled"`
		CSFLE            *CSFLE                   `bson:"csfle"`
		Extra            map[string]any           `bson:",inline"`
	}
	if err := bson.Unmarshal(data, &temp); err != nil {
		return fmt.Errorf("error unmarshalling to temporary RunOnBlock object: %w", err)
	}
	if len(temp.Extra) > 0 {
		return fmt.Errorf("unrecognized fields for RunOnBlock: %v", temp.Extra)
	}

	r.MinServerVersion = temp.MinServerVersion
	r.MaxServerVersion = temp.MaxServerVersion
	r.Serverless = temp.Serverless
	r.ServerParameters = temp.ServerParameters
	r.Auth = temp.Auth
	r.AuthEnabled = temp.AuthEnabled
	r.CSFLE = temp.CSFLE

	if temp.Topology != nil {
		r.Topology = temp.Topology
	}
	if temp.Topologies != nil {
		if r.Topology != nil {
			return errors.New("both 'topology' and 'topologies' keys cannot be specified for a RunOnBlock")
		}

		r.Topology = temp.Topologies
	}
	return nil
}

// optionFunc is a function type that configures a T instance.
type optionFunc func(*T)

// Options is the type used to configure a new T instance.
type Options struct {
	optFuncs []optionFunc
}

// NewOptions creates an empty Options instance.
func NewOptions() *Options {
	return &Options{}
}

// CollectionCreateOptions sets the options to pass to Database.CreateCollection() when creating a collection for a test.
func (op *Options) CollectionCreateOptions(opts *options.CreateCollectionOptionsBuilder) *Options {
	op.optFuncs = append(op.optFuncs, func(t *T) {
		t.collCreateOpts = opts
	})
	return op
}

// CollectionOptions sets the options to use when creating a collection for a test.
func (op *Options) CollectionOptions(opts *options.CollectionOptionsBuilder) *Options {
	op.optFuncs = append(op.optFuncs, func(t *T) {
		t.collOpts = opts
	})
	return op
}

// ClientOptions sets the options to use when creating a client for a test.
func (op *Options) ClientOptions(opts *options.ClientOptions) *Options {
	op.optFuncs = append(op.optFuncs, func(t *T) {
		t.clientOpts = opts
	})
	return op
}

// CreateClient specifies whether or not a client should be created for a test. This should be set to false when running
// a test that only runs other tests.
func (op *Options) CreateClient(create bool) *Options {
	op.optFuncs = append(op.optFuncs, func(t *T) {
		t.createClient = &create
	})
	return op
}

// CreateCollection specifies whether or not a collection should be created for a test. The default value is true.
func (op *Options) CreateCollection(create bool) *Options {
	op.optFuncs = append(op.optFuncs, func(t *T) {
		t.createCollection = &create
	})
	return op
}

// CollectionName specifies the name for the collection for the test.
func (op *Options) CollectionName(collName string) *Options {
	op.optFuncs = append(op.optFuncs, func(t *T) {
		t.collName = collName
	})
	return op
}

// DatabaseName specifies the name of the database for the test.
func (op *Options) DatabaseName(dbName string) *Options {
	op.optFuncs = append(op.optFuncs, func(t *T) {
		t.dbName = dbName
	})
	return op
}

// ClientType specifies the type of client that should be created for a test. This option will be propagated to all
// sub-tests. If the provided ClientType is Proxy, the SSL(false) option will be also be added because the internal
// proxy dialer and connection types do not support SSL.
func (op *Options) ClientType(ct ClientType) *Options {
	op.optFuncs = append(op.optFuncs, func(t *T) {
		t.clientType = ct

		if ct == Proxy {
			t.ssl = &falseBool
		}
	})
	return op
}

// MockResponses specifies the responses returned by a mock deployment. This should only be used if the current test
// is being run with MockDeployment(true). Responses can also be added after a sub-test has already been created.
func (op *Options) MockResponses(responses ...bson.D) *Options {
	op.optFuncs = append(op.optFuncs, func(t *T) {
		t.mockResponses = responses
	})
	return op
}

// RunOn specifies run-on blocks used to determine if a test should run. If a test's environment meets at least one of the
// given constraints, it will be run. Otherwise, it will be skipped.
func (op *Options) RunOn(blocks ...RunOnBlock) *Options {
	op.optFuncs = append(op.optFuncs, func(t *T) {
		t.runOn = append(t.runOn, blocks...)
	})
	return op
}

// MinServerVersion specifies the minimum server version for the test.
func (op *Options) MinServerVersion(version string) *Options {
	op.optFuncs = append(op.optFuncs, func(t *T) {
		t.minServerVersion = version
	})
	return op
}

// MaxServerVersion specifies the maximum server version for the test.
func (op *Options) MaxServerVersion(version string) *Options {
	op.optFuncs = append(op.optFuncs, func(t *T) {
		t.maxServerVersion = version
	})
	return op
}

// Topologies specifies a list of topologies that the test can run on.
func (op *Options) Topologies(topos ...TopologyKind) *Options {
	op.optFuncs = append(op.optFuncs, func(t *T) {
		t.validTopologies = topos
	})
	return op
}

// Auth specifies whether or not auth should be enabled for this test to run. By default, a test will run regardless
// of whether or not auth is enabled.
func (op *Options) Auth(auth bool) *Options {
	op.optFuncs = append(op.optFuncs, func(t *T) {
		t.auth = &auth
	})
	return op
}

// SSL specifies whether or not SSL should be enabled for this test to run. By default, a test will run regardless
// of whether or not SSL is enabled.
func (op *Options) SSL(ssl bool) *Options {
	op.optFuncs = append(op.optFuncs, func(t *T) {
		t.ssl = &ssl
	})
	return op
}

// Enterprise specifies whether or not this test should only be run on enterprise server variants. Defaults to false.
func (op *Options) Enterprise(ent bool) *Options {
	op.optFuncs = append(op.optFuncs, func(t *T) {
		t.enterprise = &ent
	})
	return op
}

// RequireAPIVersion specifies whether this test should only be run when REQUIRE_API_VERSION is true. Defaults to false.
func (op *Options) RequireAPIVersion(rav bool) *Options {
	op.optFuncs = append(op.optFuncs, func(t *T) {
		t.requireAPIVersion = &rav
	})
	return op
}

// AllowFailPointsOnSharded bypasses the check for failpoints used on sharded
// topologies.
//
// Failpoints are generally unreliable on sharded topologies, but can be used if
// the failpoint is explicitly applied to every mongoS node in the cluster.
//
// TODO(GODRIVER-3328): Remove this option once we set failpoints on every
// mongoS in sharded topologies.
func (op *Options) AllowFailPointsOnSharded() *Options {
	op.optFuncs = append(op.optFuncs, func(t *T) {
		t.allowFailPointsOnSharded = true
	})
	return op
}
