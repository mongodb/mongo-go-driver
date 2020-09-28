// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongolog

// Options represents options that can be used to configure a MongoLogger object
type Options struct {
	Logger               Logger
	MaxDocumentLength    interface{}
	OutputFile           *string
	CommandLevel         *Level
	ConnectionLevel      *Level
	SDAMLevel            *Level
	ServerSelectionLevel *Level
}

// NewOptions creates a new Options instance
func NewOptions() *Options {
	return &Options{}
}

// SetLogger sets the underlying logger
func (mlo *Options) SetLogger(logger Logger) *Options {
	mlo.Logger = logger
	return mlo
}

// SetMaxDocumentLength sets maximum length for extended json docments in log messages. If the
// document is longer than that it is truncated. Can be set to either an integer with this method
// or to unlimited with SetMaxDocumentLengthUnlimited. Defaults to 1000.
func (mlo *Options) SetMaxDocumentLength(len int) *Options {
	mlo.MaxDocumentLength = len
	return mlo
}

// SetMaxDocumentLengthUnlimited sets maxDocumentLength to be unlimited, so the logger will print
// entire documents regardless of length. Defaults to 1000.
func (mlo *Options) SetMaxDocumentLengthUnlimited() *Options {
	mlo.MaxDocumentLength = "unlimited"
	return mlo
}

// SetOutputFile sets the output file for the logs. This only applies if SetLogger is not called
func (mlo *Options) SetOutputFile(file string) *Options {
	mlo.OutputFile = &file
	return mlo
}

// SetLevel sets the log level for all logging
func (mlo *Options) SetLevel(level Level) *Options {
	mlo.CommandLevel = &level
	mlo.ConnectionLevel = &level
	mlo.SDAMLevel = &level
	mlo.ServerSelectionLevel = &level
	return mlo
}

// SetCommandLevel sets the log level for commands
func (mlo *Options) SetCommandLevel(level Level) *Options {
	mlo.CommandLevel = &level
	return mlo
}

// SetConnectionLevel sets the log level for connections
func (mlo *Options) SetConnectionLevel(level Level) *Options {
	mlo.ConnectionLevel = &level
	return mlo
}

// SetSDAMLevel sets the log level for SDAM
func (mlo *Options) SetSDAMLevel(level Level) *Options {
	mlo.SDAMLevel = &level
	return mlo
}

// SetServerSelectionLevel sets the log level for server selection
func (mlo *Options) SetServerSelectionLevel(level Level) *Options {
	mlo.ServerSelectionLevel = &level
	return mlo
}

// MergeOptions combines the given Options instances into a single Options in a last-one-wins fashion.
func MergeOptions(opts ...*Options) *Options {
	mlOpts := NewOptions()
	for _, mlo := range opts {
		if mlo == nil {
			continue
		}
		if mlo.Logger != nil {
			mlOpts.Logger = mlo.Logger
		}
		if mlo.MaxDocumentLength != nil {
			mlOpts.MaxDocumentLength = mlo.MaxDocumentLength
		}
		if mlo.OutputFile != nil {
			mlOpts.OutputFile = mlo.OutputFile
		}
		if mlo.CommandLevel != nil {
			mlOpts.CommandLevel = mlo.CommandLevel
		}
		if mlo.ConnectionLevel != nil {
			mlOpts.ConnectionLevel = mlo.ConnectionLevel
		}
		if mlo.SDAMLevel != nil {
			mlOpts.SDAMLevel = mlo.SDAMLevel
		}
		if mlo.ServerSelectionLevel != nil {
			mlOpts.ServerSelectionLevel = mlo.ServerSelectionLevel
		}
	}

	return mlOpts
}
