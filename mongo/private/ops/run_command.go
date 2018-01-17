// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ops

import (
	"context"

	"github.com/skriptble/wilson/bson"
)

// RunCommand runs a command on the database.
func RunCommand(ctx context.Context, s *SelectedServer, db string, command interface{},
	result interface{}) (bson.Reader, error) {

	return runMustUsePrimary(ctx, s, db, command, result)
}
