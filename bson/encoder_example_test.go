// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson_test

import (
	"bytes"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
)

func ExampleEncoder() {
	// Create an Encoder that writes BSON values to a bytes.Buffer.
	buf := new(bytes.Buffer)
	vw, err := bsonrw.NewBSONValueWriter(buf)
	if err != nil {
		panic(err)
	}
	encoder, err := bson.NewEncoder(vw)
	if err != nil {
		panic(err)
	}

	type Product struct {
		Name  string `bson:"name"`
		SKU   string `bson:"sku"`
		Price int64  `bson:"price_cents"`
	}

	// Use the Encoder to marshal a BSON document that contains the name, SKU,
	// and price (in cents) of a product.
	product := Product{
		Name:  "Cereal Rounds",
		SKU:   "AB12345",
		Price: 399,
	}
	err = encoder.Encode(product)
	if err != nil {
		panic(err)
	}

	// Print the BSON document as Extended JSON by converting it to bson.Raw.
	fmt.Println(bson.Raw(buf.Bytes()).String())
	// Output: {"name": "Cereal Rounds","sku": "AB12345","price_cents": {"$numberLong":"399"}}
}

type CityState struct {
	City  string
	State string
}

func (k CityState) String() string {
	return fmt.Sprintf("%s, %s", k.City, k.State)
}

func ExampleEncoder_StringifyMapKeysWithFmt() {
	// Create an Encoder that writes BSON values to a bytes.Buffer.
	buf := new(bytes.Buffer)
	vw, err := bsonrw.NewBSONValueWriter(buf)
	if err != nil {
		panic(err)
	}
	encoder, err := bson.NewEncoder(vw)
	if err != nil {
		panic(err)
	}

	// Configure the Encoder to convert Go map keys to BSON document field names
	// using fmt.Sprintf instead of the default string conversion logic.
	encoder.StringifyMapKeysWithFmt()

	// Use the Encoder to marshal a BSON document that contains is a map of
	// city and state to a list of zip codes in that city.
	zipCodes := map[CityState][]int{
		{City: "New York", State: "NY"}: {10001, 10301, 10451},
	}
	err = encoder.Encode(zipCodes)
	if err != nil {
		panic(err)
	}

	// Print the BSON document as Extended JSON by converting it to bson.Raw.
	fmt.Println(bson.Raw(buf.Bytes()).String())
	// Output: {"New York, NY": [{"$numberInt":"10001"},{"$numberInt":"10301"},{"$numberInt":"10451"}]}
}

func ExampleEncoder_UseJSONStructTags() {
	// Create an Encoder that writes BSON values to a bytes.Buffer.
	buf := new(bytes.Buffer)
	vw, err := bsonrw.NewBSONValueWriter(buf)
	if err != nil {
		panic(err)
	}
	encoder, err := bson.NewEncoder(vw)
	if err != nil {
		panic(err)
	}

	type Product struct {
		Name  string `json:"name"`
		SKU   string `json:"sku"`
		Price int64  `json:"price_cents"`
	}

	// Configure the Encoder to use "json" struct tags when decoding if "bson"
	// struct tags are not present.
	encoder.UseJSONStructTags()

	// Use the Encoder to marshal a BSON document that contains the name, SKU,
	// and price (in cents) of a product.
	product := Product{
		Name:  "Cereal Rounds",
		SKU:   "AB12345",
		Price: 399,
	}
	err = encoder.Encode(product)
	if err != nil {
		panic(err)
	}

	// Print the BSON document as Extended JSON by converting it to bson.Raw.
	fmt.Println(bson.Raw(buf.Bytes()).String())
	// Output: {"name": "Cereal Rounds","sku": "AB12345","price_cents": {"$numberLong":"399"}}
}
