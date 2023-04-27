// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson_test

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

// This example uses Raw to skip parsing a nested document in a BSON message.
func ExampleRaw_unmarshal() {
	b, err := bson.Marshal(bson.M{
		"Word":     "beach",
		"Synonyms": bson.A{"coast", "shore", "waterfront"},
	})
	if err != nil {
		panic(err)
	}

	var res struct {
		Word     string
		Synonyms bson.Raw // Don't parse the whole list, we just want to count the elements.
	}

	err = bson.Unmarshal(b, &res)
	if err != nil {
		panic(err)
	}
	elems, err := res.Synonyms.Elements()
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s, synonyms count: %d\n", res.Word, len(elems))

	// Output: beach, synonyms count: 3
}

// This example uses Raw to add a precomputed BSON document during marshal.
func ExampleRaw_marshal() {
	precomputed, err := bson.Marshal(bson.M{"Precomputed": true})
	if err != nil {
		panic(err)
	}

	msg := struct {
		Message  string
		Metadata bson.Raw
	}{
		Message:  "Hello World!",
		Metadata: precomputed,
	}

	b, err := bson.Marshal(msg)
	if err != nil {
		panic(err)
	}
	// Print the Extended JSON by converting BSON to bson.Raw.
	fmt.Println(bson.Raw(b).String())

	// Output: {"message": "Hello World!","metadata": {"Precomputed": true}}
}

// This example uses RawValue to delay parsing a value in a BSON message.
func ExampleRawValue_unmarshal() {
	b1, err := bson.Marshal(bson.M{
		"Format":    "UNIX",
		"Timestamp": 1675282389,
	})
	if err != nil {
		panic(err)
	}

	b2, err := bson.Marshal(bson.M{
		"Format":    "RFC3339",
		"Timestamp": time.Unix(1675282389, 0).Format(time.RFC3339),
	})
	if err != nil {
		panic(err)
	}

	for _, b := range [][]byte{b1, b2} {
		var res struct {
			Format    string
			Timestamp bson.RawValue // Delay parsing until we know the timestamp format.
		}

		err = bson.Unmarshal(b, &res)
		if err != nil {
			panic(err)
		}

		var t time.Time
		switch res.Format {
		case "UNIX":
			t = time.Unix(res.Timestamp.AsInt64(), 0)
		case "RFC3339":
			t, err = time.Parse(time.RFC3339, res.Timestamp.StringValue())
			if err != nil {
				panic(err)
			}
		}
		fmt.Println(res.Format, t.Unix())
	}

	// Output:
	// UNIX 1675282389
	// RFC3339 1675282389
}

// This example uses RawValue to add a precomputed BSON string value during marshal.
func ExampleRawValue_marshal() {
	t, val, err := bson.MarshalValue("Precomputed message!")
	if err != nil {
		panic(err)
	}
	precomputed := bson.RawValue{
		Type:  t,
		Value: val,
	}

	msg := struct {
		Message bson.RawValue
		Time    time.Time
	}{
		Message: precomputed,
		Time:    time.Unix(1675282389, 0),
	}

	b, err := bson.Marshal(msg)
	if err != nil {
		panic(err)
	}
	// Print the Extended JSON by converting BSON to bson.Raw.
	fmt.Println(bson.Raw(b).String())

	// Output: {"message": "Precomputed message!","time": {"$date":{"$numberLong":"1675282389000"}}}
}
