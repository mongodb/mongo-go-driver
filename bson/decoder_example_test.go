// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
)

func ExampleDecoder() {
	// Marshal a BSON document that contains the name, SKU, and price (in cents)
	// of a product.
	doc := bson.D{
		{Key: "name", Value: "Cereal Rounds"},
		{Key: "sku", Value: "AB12345"},
		{Key: "price_cents", Value: 399},
	}
	data, err := bson.Marshal(doc)
	if err != nil {
		panic(err)
	}

	// Create a Decoder that reads the marshaled BSON document and use it to
	// unmarshal the document into a Product struct.
	decoder, err := bson.NewDecoder(bsonrw.NewBSONDocumentReader(data))
	if err != nil {
		panic(err)
	}

	type Product struct {
		Name  string `bson:"name"`
		SKU   string `bson:"sku"`
		Price int64  `bson:"price_cents"`
	}

	var res Product
	err = decoder.Decode(&res)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%+v\n", res)
	// Output: {Name:Cereal Rounds SKU:AB12345 Price:399}
}

func ExampleDecoder_DefaultDocumentM() {
	// Marshal a BSON document that contains a city name and a nested document
	// with various city properties.
	doc := bson.D{
		{Key: "name", Value: "New York"},
		{Key: "properties", Value: bson.D{
			{Key: "state", Value: "NY"},
			{Key: "population", Value: 8_804_190},
			{Key: "elevation", Value: 10},
		}},
	}
	data, err := bson.Marshal(doc)
	if err != nil {
		panic(err)
	}

	// Create a Decoder that reads the marshaled BSON document and use it to unmarshal the document
	// into a City struct.
	decoder, err := bson.NewDecoder(bsonrw.NewBSONDocumentReader(data))
	if err != nil {
		panic(err)
	}

	type City struct {
		Name       string      `bson:"name"`
		Properties interface{} `bson:"properties"`
	}

	// Configure the Decoder to default to decoding BSON documents as the bson.M
	// type if the decode destination has no type information. The Properties
	// field in the City struct will be decoded as a "bson.M" (i.e. map) instead
	// of the default "bson.D".
	decoder.DefaultDocumentM()

	var res City
	err = decoder.Decode(&res)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%+v\n", res)
	// Output: {Name:New York Properties:map[elevation:10 population:8804190 state:NY]}
}

func ExampleDecoder_UseJSONStructTags() {
	// Marshal a BSON document that contains the name, SKU, and price (in cents)
	// of a product.
	doc := bson.D{
		{Key: "name", Value: "Cereal Rounds"},
		{Key: "sku", Value: "AB12345"},
		{Key: "price_cents", Value: 399},
	}
	data, err := bson.Marshal(doc)
	if err != nil {
		panic(err)
	}

	// Create a Decoder that reads the marshaled BSON document and use it to
	// unmarshal the document into a Product struct.
	decoder, err := bson.NewDecoder(bsonrw.NewBSONDocumentReader(data))
	if err != nil {
		panic(err)
	}

	type Product struct {
		Name  string `json:"name"`
		SKU   string `json:"sku"`
		Price int64  `json:"price_cents"`
	}

	// Configure the Decoder to use "json" struct tags when decoding if "bson"
	// struct tags are not present.
	decoder.UseJSONStructTags()

	var res Product
	err = decoder.Decode(&res)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%+v\n", res)
	// Output: {Name:Cereal Rounds SKU:AB12345 Price:399}
}

func ExampleDecoder_extendedJSON() {
	// Define an Extended JSON document that contains the name, SKU, and price
	// (in cents) of a product.
	data := []byte(`{"name":"Cereal Rounds","sku":"AB12345","price_cents":{"$numberLong":"399"}}`)

	// Create a Decoder that reads the Extended JSON document and use it to
	// unmarshal the document into a Product struct.
	vr, err := bsonrw.NewExtJSONValueReader(bytes.NewReader(data), true)
	if err != nil {
		panic(err)
	}
	decoder, err := bson.NewDecoder(vr)
	if err != nil {
		panic(err)
	}

	type Product struct {
		Name  string `bson:"name"`
		SKU   string `bson:"sku"`
		Price int64  `bson:"price_cents"`
	}

	var res Product
	err = decoder.Decode(&res)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%+v\n", res)
	// Output: {Name:Cereal Rounds SKU:AB12345 Price:399}
}

func ExampleDecoder_multipleExtendedJSONDocuments() {
	// Define a newline-separated sequence of Extended JSON documents that
	// contain X,Y coordinates.
	data := []byte(`
{"x":{"$numberInt":"0"},"y":{"$numberInt":"0"}}
{"x":{"$numberInt":"1"},"y":{"$numberInt":"1"}}
{"x":{"$numberInt":"2"},"y":{"$numberInt":"2"}}
{"x":{"$numberInt":"3"},"y":{"$numberInt":"3"}}
{"x":{"$numberInt":"4"},"y":{"$numberInt":"4"}}
`)

	// Create a Decoder that reads the Extended JSON documents and use it to
	// unmarshal the documents Coordinate structs.
	vr, err := bsonrw.NewExtJSONValueReader(bytes.NewReader(data), true)
	if err != nil {
		panic(err)
	}
	decoder, err := bson.NewDecoder(vr)
	if err != nil {
		panic(err)
	}

	type Coordinate struct {
		X int
		Y int
	}

	// Read and unmarshal each Extended JSON document from the sequence. If
	// Decode returns error io.EOF, that means the Decoder has reached the end
	// of the input, so break the loop.
	for {
		var res Coordinate
		err = decoder.Decode(&res)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			panic(err)
		}

		fmt.Printf("%+v\n", res)
	}
	// Output:
	// {X:0 Y:0}
	// {X:1 Y:1}
	// {X:2 Y:2}
	// {X:3 Y:3}
	// {X:4 Y:4}
}
