// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"fmt"
	"testing"
)

func ExampleDocument() {
	internalVersion := "1234567"

	f := func(appName string) *Documentv2 {
		doc := NewDocumentv2().Append(
			"driver", EmbedElements([]Elementv2{
				{Key: "name", Value: String("mongo-go-driver")},
				{Key: "version", Value: String(internalVersion)}},
			)).Append(
			"os", EmbedElements([]Elementv2{
				{Key: "type", Value: String("darwin")},
				{Key: "architecture", Value: String("amd64")}},
			)).Append(
			"platform", String("go1.9.2"),
		)
		if appName != "" {
			doc.Append("application", EmbedElement("name", String(appName)))
		}

		return doc
	}
	buf, err := f("hello-world").MarshalBSON()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(buf)

	// Output: [177 0 0 0 3 100 114 105 118 101 114 0 52 0 0 0 2 110 97 109 101 0 16 0 0 0 109 111 110 103 111 45 103 111 45 100 114 105 118 101 114 0 2 118 101 114 115 105 111 110 0 8 0 0 0 49 50 51 52 53 54 55 0 0 3 111 115 0 46 0 0 0 2 116 121 112 101 0 7 0 0 0 100 97 114 119 105 110 0 2 97 114 99 104 105 116 101 99 116 117 114 101 0 6 0 0 0 97 109 100 54 52 0 0 2 112 108 97 116 102 111 114 109 0 8 0 0 0 103 111 49 46 57 46 50 0 3 97 112 112 108 105 99 97 116 105 111 110 0 27 0 0 0 2 110 97 109 101 0 12 0 0 0 104 101 108 108 111 45 119 111 114 108 100 0 0 0]
}

func BenchmarkDocument(b *testing.B) {
	b.ReportAllocs()
	internalVersion := "1234567"
	for i := 0; i < b.N; i++ {
		doc := NewDocumentv2().Append(
			"driver", EmbedElements([]Elementv2{
				{Key: "name", Value: String("mongo-go-driver")},
				{Key: "version", Value: String(internalVersion)}},
			)).Append(
			"os", EmbedElements([]Elementv2{
				{Key: "type", Value: String("darwin")},
				{Key: "architecture", Value: String("amd64")}},
			)).Append(
			"platform", String("go1.9.2"),
		)
		_, _ = doc.MarshalBSON()
	}
}

func valueEqual(v1, v2 Valuev2) bool { return v1.Equal(v2) }

func elementEqual(e1, e2 Elementv2) bool { return e1.Equal(e2) }

func documentComparer(d1, d2 *Documentv2) bool { return d1.Equal(d2) }
