// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mgocompat

import (
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Registry is the mgo compatible bson.Registry. It contains the default and
// primitive codecs with mgo compatible options.
var Registry = bson.NewMgoRegistry()

// RespectNilValuesRegistry is the bson.Registry compatible with mgo withSetRespectNilValues set to true.
var RespectNilValuesRegistry = bson.NewRespectNilValuesMgoRegistry()
