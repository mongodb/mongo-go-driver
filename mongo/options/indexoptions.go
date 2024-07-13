// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

// CreateIndexesArgs represents arguments that can be used to configure
// IndexView.CreateOne and IndexView.CreateMany operations.
type CreateIndexesArgs struct {
	// The number of data-bearing members of a replica set, including the primary, that must complete the index builds
	// successfully before the primary marks the indexes as ready. This should either be a string or int32 value. The
	// semantics of the values are as follows:
	//
	// 1. String: specifies a tag. All members with that tag must complete the build.
	// 2. int: the number of members that must complete the build.
	// 3. "majority": A special value to indicate that more than half the nodes must complete the build.
	// 4. "votingMembers": A special value to indicate that all voting data-bearing nodes must complete.
	//
	// This option is only available on MongoDB versions >= 4.4. A client-side error will be returned if the option
	// is specified for MongoDB versions <= 4.2. The default value is nil, meaning that the server-side default will be
	// used. See dochub.mongodb.org/core/index-commit-quorum for more information.
	CommitQuorum interface{}
}

// CreateIndexesOptions contains options to create indexes. Each option can be
// set through setter functions. See documentation for each setter function for
// an explanation of the option.
type CreateIndexesOptions struct {
	Opts []func(*CreateIndexesArgs) error
}

// CreateIndexes creates a new CreateIndexesOptions instance.
func CreateIndexes() *CreateIndexesOptions {
	return &CreateIndexesOptions{}
}

// ArgsSetters returns a list of CreateIndexesArgs setter functions.
func (c *CreateIndexesOptions) ArgsSetters() []func(*CreateIndexesArgs) error {
	return c.Opts
}

// SetCommitQuorumInt sets the value for the CommitQuorum field as an int32.
func (c *CreateIndexesOptions) SetCommitQuorumInt(quorum int32) *CreateIndexesOptions {
	c.Opts = append(c.Opts, func(args *CreateIndexesArgs) error {
		args.CommitQuorum = quorum

		return nil
	})

	return c
}

// SetCommitQuorumString sets the value for the CommitQuorum field as a string.
func (c *CreateIndexesOptions) SetCommitQuorumString(quorum string) *CreateIndexesOptions {
	c.Opts = append(c.Opts, func(args *CreateIndexesArgs) error {
		args.CommitQuorum = quorum

		return nil
	})

	return c
}

// SetCommitQuorumMajority sets the value for the CommitQuorum to special "majority" value.
func (c *CreateIndexesOptions) SetCommitQuorumMajority() *CreateIndexesOptions {
	c.Opts = append(c.Opts, func(args *CreateIndexesArgs) error {
		args.CommitQuorum = "majority"

		return nil
	})

	return c
}

// SetCommitQuorumVotingMembers sets the value for the CommitQuorum to special "votingMembers" value.
func (c *CreateIndexesOptions) SetCommitQuorumVotingMembers() *CreateIndexesOptions {
	c.Opts = append(c.Opts, func(args *CreateIndexesArgs) error {
		args.CommitQuorum = "votingMembers"

		return nil
	})

	return c
}

// DropIndexesArgs represents arguments that can be used to configure
// IndexView.DropOne and IndexView.DropAll operations.
type DropIndexesArgs struct{}

// DropIndexesOptions contains options to configure dropping indexes. Each
// option can be set through setter functions. See documentation for each setter
// function for an explanation of the option.
type DropIndexesOptions struct {
	Opts []func(*DropIndexesArgs) error
}

// DropIndexes creates a new DropIndexesOptions instance.
func DropIndexes() *DropIndexesOptions {
	return &DropIndexesOptions{}
}

// ArgsSetters returns a list of DropIndexesArgs setter functions.
func (d *DropIndexesOptions) ArgsSetters() []func(*DropIndexesArgs) error {
	return d.Opts
}

// ListIndexesArgs represents arguments that can be used to configure an
// IndexView.List operation.
type ListIndexesArgs struct {
	// The maximum number of documents to be included in each batch returned by the server.
	BatchSize *int32
}

// ListIndexesOptions contains options to configure count operations. Each
// option can be set through setter functions. See documentation for each setter
// function for an explanation of the option.
type ListIndexesOptions struct {
	Opts []func(*ListIndexesArgs) error
}

// ListIndexes creates a new ListIndexesOptions instance.
func ListIndexes() *ListIndexesOptions {
	return &ListIndexesOptions{}
}

// ArgsSetters returns a list of CountArgs setter functions.
func (l *ListIndexesOptions) ArgsSetters() []func(*ListIndexesArgs) error {
	return l.Opts
}

// SetBatchSize sets the value for the BatchSize field.
func (l *ListIndexesOptions) SetBatchSize(i int32) *ListIndexesOptions {
	l.Opts = append(l.Opts, func(args *ListIndexesArgs) error {
		args.BatchSize = &i

		return nil
	})

	return l
}

// IndexArgs represents arguments that can be used to configure a new index
// created through the IndexView.CreateOne or IndexView.CreateMany operations.
type IndexArgs struct {
	// The length of time, in seconds, for documents to remain in the collection. The default value is 0, which means
	// that documents will remain in the collection until they're explicitly deleted or the collection is dropped.
	ExpireAfterSeconds *int32

	// The name of the index. The default value is "[field1]_[direction1]_[field2]_[direction2]...". For example, an
	// index with the specification {name: 1, age: -1} will be named "name_1_age_-1".
	Name *string

	// If true, the index will only reference documents that contain the fields specified in the index. The default is
	// false.
	Sparse *bool

	// Specifies the storage engine to use for the index. The value must be a document in the form
	// {<storage engine name>: <options>}. The default value is nil, which means that the default storage engine
	// will be used. This option is only applicable for MongoDB versions >= 3.0 and is ignored for previous server
	// versions.
	StorageEngine interface{}

	// If true, the collection will not accept insertion or update of documents where the index key value matches an
	// existing value in the index. The default is false.
	Unique *bool

	// The index version number, either 0 or 1.
	Version *int32

	// The language that determines the list of stop words and the rules for the stemmer and tokenizer. This option
	// is only applicable for text indexes and is ignored for other index types. The default value is "english".
	DefaultLanguage *string

	// The name of the field in the collection's documents that contains the override language for the document. This
	// option is only applicable for text indexes and is ignored for other index types. The default value is the value
	// of the DefaultLanguage option.
	LanguageOverride *string

	// The index version number for a text index. See https://www.mongodb.com/docs/manual/core/index-text/#text-versions for
	// information about different version numbers.
	TextVersion *int32

	// A document that contains field and weight pairs. The weight is an integer ranging from 1 to 99,999, inclusive,
	// indicating the significance of the field relative to the other indexed fields in terms of the score. This option
	// is only applicable for text indexes and is ignored for other index types. The default value is nil, which means
	// that every field will have a weight of 1.
	Weights interface{}

	// The index version number for a 2D sphere index. See https://www.mongodb.com/docs/manual/core/2dsphere/#dsphere-v2 for
	// information about different version numbers.
	SphereVersion *int32

	// The precision of the stored geohash value of the location data. This option only applies to 2D indexes and is
	// ignored for other index types. The value must be between 1 and 32, inclusive. The default value is 26.
	Bits *int32

	// The upper inclusive boundary for longitude and latitude values. This option is only applicable to 2D indexes and
	// is ignored for other index types. The default value is 180.0.
	Max *float64

	// The lower inclusive boundary for longitude and latitude values. This option is only applicable to 2D indexes and
	// is ignored for other index types. The default value is -180.0.
	Min *float64

	// The number of units within which to group location values. Location values that are within BucketSize units of
	// each other will be grouped in the same bucket. This option is only applicable to geoHaystack indexes and is
	// ignored for other index types. The value must be greater than 0.
	BucketSize *int32

	// A document that defines which collection documents the index should reference. This option is only valid for
	// MongoDB versions >= 3.2 and is ignored for previous server versions.
	PartialFilterExpression interface{}

	// The collation to use for string comparisons for the index. This option is only valid for MongoDB versions >= 3.4.
	// For previous server versions, the driver will return an error if this option is used.
	Collation *Collation

	// A document that defines the wildcard projection for the index.
	WildcardProjection interface{}

	// If true, the index will exist on the target collection but will not be used by the query planner when executing
	// operations. This option is only valid for MongoDB versions >= 4.4. The default value is false.
	Hidden *bool
}

// IndexOptions contains options to configure index operations. Each option
// can be set through setter functions. See documentation for each setter
// function for an explanation of the option.
type IndexOptions struct {
	Opts []func(*IndexArgs) error
}

// Index creates a new IndexOptions instance.
func Index() *IndexOptions {
	return &IndexOptions{}
}

// ArgsSetters returns a list of IndexArgs setter functions.
func (i *IndexOptions) ArgsSetters() []func(*IndexArgs) error {
	return i.Opts
}

// SetExpireAfterSeconds sets value for the ExpireAfterSeconds field.
func (i *IndexOptions) SetExpireAfterSeconds(seconds int32) *IndexOptions {
	i.Opts = append(i.Opts, func(args *IndexArgs) error {
		args.ExpireAfterSeconds = &seconds

		return nil
	})

	return i
}

// SetName sets the value for the Name field.
func (i *IndexOptions) SetName(name string) *IndexOptions {
	i.Opts = append(i.Opts, func(args *IndexArgs) error {
		args.Name = &name

		return nil
	})

	return i
}

// SetSparse sets the value of the Sparse field.
func (i *IndexOptions) SetSparse(sparse bool) *IndexOptions {
	i.Opts = append(i.Opts, func(args *IndexArgs) error {
		args.Sparse = &sparse

		return nil
	})

	return i
}

// SetStorageEngine sets the value for the StorageEngine field.
func (i *IndexOptions) SetStorageEngine(engine interface{}) *IndexOptions {
	i.Opts = append(i.Opts, func(args *IndexArgs) error {
		args.StorageEngine = engine

		return nil
	})

	return i
}

// SetUnique sets the value for the Unique field.
func (i *IndexOptions) SetUnique(unique bool) *IndexOptions {
	i.Opts = append(i.Opts, func(args *IndexArgs) error {
		args.Unique = &unique

		return nil
	})

	return i
}

// SetVersion sets the value for the Version field.
func (i *IndexOptions) SetVersion(version int32) *IndexOptions {
	i.Opts = append(i.Opts, func(args *IndexArgs) error {
		args.Version = &version

		return nil
	})

	return i
}

// SetDefaultLanguage sets the value for the DefaultLanguage field.
func (i *IndexOptions) SetDefaultLanguage(language string) *IndexOptions {
	i.Opts = append(i.Opts, func(args *IndexArgs) error {
		args.DefaultLanguage = &language

		return nil
	})

	return i
}

// SetLanguageOverride sets the value of the LanguageOverride field.
func (i *IndexOptions) SetLanguageOverride(override string) *IndexOptions {
	i.Opts = append(i.Opts, func(args *IndexArgs) error {
		args.LanguageOverride = &override

		return nil
	})

	return i
}

// SetTextVersion sets the value for the TextVersion field.
func (i *IndexOptions) SetTextVersion(version int32) *IndexOptions {
	i.Opts = append(i.Opts, func(args *IndexArgs) error {
		args.TextVersion = &version

		return nil
	})

	return i
}

// SetWeights sets the value for the Weights field.
func (i *IndexOptions) SetWeights(weights interface{}) *IndexOptions {
	i.Opts = append(i.Opts, func(args *IndexArgs) error {
		args.Weights = weights

		return nil
	})

	return i
}

// SetSphereVersion sets the value for the SphereVersion field.
func (i *IndexOptions) SetSphereVersion(version int32) *IndexOptions {
	i.Opts = append(i.Opts, func(args *IndexArgs) error {
		args.SphereVersion = &version

		return nil
	})

	return i
}

// SetBits sets the value for the Bits field.
func (i *IndexOptions) SetBits(bits int32) *IndexOptions {
	i.Opts = append(i.Opts, func(args *IndexArgs) error {
		args.Bits = &bits

		return nil
	})

	return i
}

// SetMax sets the value for the Max field.
func (i *IndexOptions) SetMax(max float64) *IndexOptions {
	i.Opts = append(i.Opts, func(args *IndexArgs) error {
		args.Max = &max

		return nil
	})

	return i
}

// SetMin sets the value for the Min field.
func (i *IndexOptions) SetMin(min float64) *IndexOptions {
	i.Opts = append(i.Opts, func(args *IndexArgs) error {
		args.Min = &min

		return nil
	})

	return i
}

// SetBucketSize sets the value for the BucketSize field
func (i *IndexOptions) SetBucketSize(bucketSize int32) *IndexOptions {
	i.Opts = append(i.Opts, func(args *IndexArgs) error {
		args.BucketSize = &bucketSize

		return nil
	})

	return i
}

// SetPartialFilterExpression sets the value for the PartialFilterExpression field.
func (i *IndexOptions) SetPartialFilterExpression(expression interface{}) *IndexOptions {
	i.Opts = append(i.Opts, func(args *IndexArgs) error {
		args.PartialFilterExpression = expression

		return nil
	})

	return i
}

// SetCollation sets the value for the Collation field.
func (i *IndexOptions) SetCollation(collation *Collation) *IndexOptions {
	i.Opts = append(i.Opts, func(args *IndexArgs) error {
		args.Collation = collation

		return nil
	})

	return i
}

// SetWildcardProjection sets the value for the WildcardProjection field.
func (i *IndexOptions) SetWildcardProjection(wildcardProjection interface{}) *IndexOptions {
	i.Opts = append(i.Opts, func(args *IndexArgs) error {
		args.WildcardProjection = wildcardProjection

		return nil
	})

	return i
}

// SetHidden sets the value for the Hidden field.
func (i *IndexOptions) SetHidden(hidden bool) *IndexOptions {
	i.Opts = append(i.Opts, func(args *IndexArgs) error {
		args.Hidden = &hidden

		return nil
	})

	return i
}
