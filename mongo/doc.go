// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package mongo provides a MongoDB Driver API for Go.
//
// Basic usage of the driver starts with creating a Client from a connection
// string. To do so, call the NewClient and Connect functions:
//
//    client, err := mongo.NewClient("mongodb://foo:bar@localhost:27017")
//    if err != nil { log.Fatal(err) }
//    err = client.Connect(context.TODO())
//    if err != nil { log.Fatal(err) }
//
// This will create a new client and start monitoring the MongoDB server on localhost.
// The Database and Collection types can be used to access the database:
//
//    collection := client.Database("baz").Collection("qux")
//
// A Collection can be used to query the database or insert documents:
//
//    res, err := collection.InsertOne(context.Background(), map[string]string{"hello": "world"})
//    if err != nil { log.Fatal(err) }
//    id := res.InsertedID
//
// Several methods return a cursor, which can be used like this:
//
//    cur, err := collection.Find(context.Background(), nil)
//    if err != nil { log.Fatal(err) }
//    defer cur.Close(context.Background())
//    for cur.Next(context.Background()) {
//       elem := bson.NewDocument()
//       err := cur.Decode(elem)
//       if err != nil { log.Fatal(err) }
//       // do something with elem....
//    }
//    if err := cur.Err(); err != nil {
//        log.Fatal(err)
//    }
//
// Methods that only return a single document will return a *DocumentResult, which works
// like a *sql.Row:
//
//    result := bson.NewDocument()
//    filter := bson.NewDocument(bson.EC.String("hello", "world"))
//    err := collection.FindOne(context.Background(), filter).Decode(result)
//    if err != nil { log.Fatal(err) }
//    // do something with result...
//
// Additional examples can be found under the examples directory in the driver's repository and
// on the MongoDB website.
package mongo
