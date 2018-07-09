// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package dropopt

import (
	"github.com/mongodb/mongo-go-driver/core/session"
)

// Drop represents all possible params for the drop() function.
type Drop interface {
	drop()
}

type DropSession interface {
	Drop
	ConvertDropSession() *session.Client
}

// DropSessionOpt is a drop session option.
type DropSessionOpt struct{}

func (DropSessionOpt) drop() {}

// ConvertDropSession implements the DropSession interface.
func (DropSessionOpt) ConvertDropSession() *session.Client {
	return nil
}
