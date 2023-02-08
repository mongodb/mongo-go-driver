// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package benchmark

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const text = "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Nunc hendrerit nibh eu lorem scelerisque iaculis. Mauris tincidunt est quis gravida sollicitudin. Vestibulum ante dolor, semper eget finibus accumsan, finibus id nulla. Nam mauris libero, mattis sed libero at, vestibulum vehicula tellus. Nullam in finibus neque. Ut bibendum blandit condimentum. Etiam tortor mauris, tempus et accumsan a, lobortis cursus lacus.  Pellentesque fermentum gravida aliquam. Donec fermentum nisi ut venenatis convallis. Praesent ultrices, justo ac fringilla eleifend, augue libero tristique quam, imperdiet rutrum lectus odio id purus. Curabitur vel metus ullamcorper massa dapibus tincidunt. Nam mollis feugiat sapien ut sagittis. Cras sed ex ligula. Etiam nec nisi diam. Pellentesque id metus molestie, tempus velit et, malesuada eros. Praesent convallis arcu et semper molestie. Morbi eu orci tempor, blandit dui egestas, faucibus diam. Aenean ac tempor justo.  Donec id lectus tristique, ultrices enim sed, suscipit sapien. In arcu mauris, venenatis sed venenatis id, mollis a risus. Phasellus id massa et magna volutpat elementum in quis libero. Aliquam a aliquam erat. Nunc leo massa, sagittis at egestas vel, iaculis vitae quam. Pellentesque placerat metus id velit lobortis euismod in non sapien. Duis a quam sed elit fringilla maximus at et purus. Nullam pharetra accumsan efficitur. Integer a mattis urna. Suspendisse vehicula, nunc euismod luctus suscipit, mi mauris sollicitudin diam, porta rhoncus libero metus vitae erat. Sed lacus sem, feugiat vitae nisi at, malesuada ultricies ipsum. Quisque hendrerit posuere metus. Donec magna erat, facilisis et dictum at, tempor in leo. Maecenas luctus vestibulum quam, eu ornare ex aliquam vitae. Mauris ac mauris posuere, mattis nisl nec, fringilla libero.  Nulla in ipsum ut arcu condimentum mollis. Donec viverra massa nec lacus condimentum vulputate. Nulla at dictum eros, quis sodales ante. Duis condimentum accumsan consectetur. Aenean sodales at turpis vitae efficitur. Vestibulum in diam faucibus, consequat sem ut, auctor lorem. Duis tincidunt non eros pretium rhoncus. Sed quis eros ligula. Donec vulputate ultrices enim, quis interdum dui rhoncus eu. In at mollis ex.  In sed pulvinar risus. Morbi efficitur leo magna, eget bibendum leo consequat id. Pellentesque ultricies ipsum leo, sit amet cursus est bibendum a. Mauris eget porta felis. Vivamus dignissim pellentesque risus eget interdum. Mauris ultrices metus at blandit tincidunt. Duis tempor sapien vel luctus mattis. Vivamus ac orci nibh. Nam eget tempor neque. Proin lacus nibh, porttitor nec pellentesque id, dignissim et eros. Nullam et libero faucibus, iaculis mi sed, faucibus leo. In mollis sem ac porta suscipit.  Ut rutrum, justo a gravida lobortis, neque nibh tincidunt mi, id eleifend dolor dolor vel arcu. Fusce vel egestas ante, eu commodo eros. Donec augue odio, bibendum ut nulla ultricies, cursus eleifend lacus. Nulla viverra ac massa vel placerat. Duis aliquam, ipsum vitae ultricies imperdiet, tellus nisl venenatis mauris, et congue elit nulla vitae est. Suspendisse volutpat ullamcorper diam, et vehicula leo bibendum at. In hac habitasse platea dictumst.  Donec mattis neque a lorem ullamcorper rutrum. Curabitur mattis auctor velit, vitae iaculis mauris convallis in. Donec vulputate sapien non ex pretium semper. Vestibulum ut ligula sit amet arcu pellentesque aliquam. Pellentesque odio diam, pharetra at nunc varius, pharetra consectetur sem. Integer pretium magna pretium, mattis lectus eget, laoreet libero. Morbi dictum et dolor eu finibus. Etiam vestibulum finibus quam, vel eleifend tortor mattis viverra. Donec porttitor est vitae ligula volutpat lobortis. Donec ac semper diam. Maecenas sapien eros, blandit non velit in, faucibus auctor risus. In quam nibh, congue a nisl sit amet, tempor volutpat tortor. Curabitur dignissim auctor orci a varius. Nulla faucibus lacus libero, vitae fringilla elit facilisis id.  Aliquam id elit dui. Cras convallis ligula ac leo bibendum lacinia. Duis interdum ac lectus sed tristique. Maecenas sem magna, gravida quis sapien sit amet, varius luctus ligula. Curabitur eleifend mi nibh. Suspendisse iaculis commodo justo, vitae pretium risus scelerisque non. Sed pulvinar augue nec fermentum feugiat. Nam et ligula tellus. Vestibulum euismod accumsan nibh, at rutrum est tristique sit amet. Duis porttitor ex felis, quis consectetur nunc tempor ut. Nulla vitae consequat velit, id condimentum orci. Sed lacinia velit urna, nec laoreet est varius ac. Integer dapibus libero vel bibendum posuere. Curabitur cursus est vel ante euismod dapibus. Ut hendrerit odio id rhoncus efficitur.  Nam luctus sem orci, in congue ipsum ultrices at. Morbi sed tortor ut metus elementum ultrices. Cras vehicula ante magna, nec faucibus neque placerat et. Vivamus justo lacus, aliquet sit amet semper ac, porta vehicula nibh. Duis et rutrum elit. In nisi eros, fringilla ut odio eget, vehicula laoreet elit. Suspendisse potenti. Vivamus ut ultricies lacus. Integer pellentesque posuere mauris, eget aliquet purus tincidunt in. Suspendisse potenti. Nam quis purus iaculis, cursus mauris a, tempus mi.  Pellentesque ullamcorper lacus vitae lacus volutpat, quis ultricies metus sagittis. Etiam imperdiet libero vitae ante cursus tempus. Nulla eu mi sodales neque scelerisque eleifend id non nisi. Maecenas blandit vitae turpis nec lacinia. Duis posuere cursus metus. Orci varius natoque penatibus et magnis dis parturient montes, nascetur ridaculus mus. Ut ultrices hendrerit quam, sit amet condimentum massa finibus a. Donec et vehicula urna. Pellentesque lorem felis, fermentum vel feugiat et, congue eleifend orci. Suspendisse potenti. Sed dignissim massa at justo tristique, id feugiat neque vulputate.  Pellentesque dui massa, maximus quis consequat quis, fringilla vel turpis. Etiam fermentum ex eget massa varius, a fringilla velit placerat. Sed fringilla convallis urna, ut finibus purus accumsan a. Vestibulum porttitor risus lorem, eu venenatis velit suscipit vel. Ut nec diam egestas, sollicitudin nulla sit amet, porttitor felis. Duis a nisl a ante interdum hendrerit. Integer sollicitudin scelerisque ex, et blandit magna blandit at.  Vivamus a vulputate ante. Nam non tortor a lacus euismod venenatis. Vestibulum libero augue, consequat vitae turpis nec, mattis tristique nibh. Fusce pulvinar dolor vel ipsum eleifend varius. Morbi id ante eget tellus venenatis interdum non sit amet ante. Nulla luctus tempor purus, eget ultrices odio varius eget. Duis commodo eros ac molestie fermentum. Praesent vestibulum est eu massa posuere, et fermentum orci tincidunt. Duis dignissim nunc sit amet elit laoreet mollis. Aenean porttitor et nunc vel venenatis. Nunc viverra ligula nec tincidunt vehicula. Pellentesque in magna volutpat, consequat est eget, varius mauris. Maecenas in tellus eros. Aliquam erat volutpat. Phasellus blandit faucibus velit ac placerat.  Donec luctus hendrerit pretium. Sed mauris purus, lobortis non erat sed, mattis ornare nulla. Fusce eu vulputate lacus. In enim justo, elementum at tortor nec, interdum semper ligula. Donec condimentum erat elit, non luctus augue rhoncus et. Quisque interdum elit dui, in vestibulum lacus aliquet et. Mauris aliquam sed ante id eleifend. Donec velit dolor, blandit et mattis non, bibendum at lorem. Nullam blandit quam sapien. Duis rutrum nunc vitae odio imperdiet condimentum. Nunc vel pellentesque purus. Cras iaculis dui est, quis cursus tortor mattis non. Donec non tincidunt lacus.  Sed eget molestie lacus. Phasellus pharetra ullamcorper purus. Sed sit amet ultricies ligula, aliquam elementum velit. Cras commodo mauris vel sapien rutrum, ac pharetra libero mollis. Donec facilisis sit amet elit ac porttitor. Phasellus rutrum rhoncus sagittis. Interdum et malesuada fames ac ante ipsum primis in faucibus. Etiam iaculis ac odio eu sodales.  Proin blandit fermentum arcu efficitur ornare. Vestibulum pharetra est quis mi lobortis interdum. Proin molestie pretium viverra. Integer pellentesque eros nisi, non rutrum odio interdum ut. Quisque vel ante et mi placerat mollis ut eget eros. Etiam vitae orci lectus. Nulla scelerisque dui in dictum ornare. Aliquam vestibulum fringilla eros, id fermentum dolor euismod eget. Ut vitae massa a augue suscipit bibendum non ac mi. Pellentesque id ligula in sapien fermentum fermentum. In ut sem molestie, consectetur ex tristique, tempor risus.  Maecenas scelerisque, ex eget cursus ornare, dolor nisi condimentum tellus, in venenatis nibh elit rutrum turpis. Sed sed vestibulum ex, molestie sodales leo. Vivamus cursus aliquet consequat. Aliquam et enim eget lorem placerat egestas a at justo. Praesent congue vitae purus vel scelerisque. Praesent faucibus massa felis, non porttitor dolor varius at. Nam fringilla risus sit amet faucibus vestibulum. Aliquam rhoncus ex vel magna blandit, eu dapibus felis tristique. Nam dignissim vestibulum neque vitae suscipit. Nunc a pharetra dui. Etiam nec quam sed mauris pharetra finibus in a tellus. Ut vehicula molestie lectus in pretium. Donec sit amet dui purus.  Nunc in vestibulum sapien. Donec elit quam, mollis luctus gravida ac, ullamcorper quis urna. Vivamus a urna egestas velit tempor interdum non eget dui. Maecenas maximus diam at consequat dictum. Etiam sed metus quis enim faucibus cursus condimentum ut nisi. In et iaculis odio. Curabitur sollicitudin ultrices finibus. Aliquam et nisi porta, vehicula urna id, dictum turpis. Sed id iaculis justo, non semper metus.  Quisque euismod, tellus iaculis sagittis vestibulum, leo magna blandit felis, non pharetra velit lacus sed nunc. Curabitur mollis porttitor odio, sed feugiat leo rhoncus quis. Duis faucibus tellus id venenatis vestibulum. Duis interdum pretium cursus. Integer sed iaculis mi. Phasellus at odio at felis fermentum congue. Morbi at ante ut lacus posuere accumsan quis in orci. Nullam eget sapien eu nibh venenatis malesuada.  Nulla sed ligula et metus mattis placerat sed eget nisl. Nunc cursus et nulla id dictum. Vivamus efficitur aliquam. "

func BenchmarkClientWrite(b *testing.B) {
	benchmarks := []struct {
		name string
		opt  *options.ClientOptions
	}{
		{name: "not compressed", opt: options.Client().ApplyURI("mongodb://localhost:27017")},
		{name: "snappy", opt: options.Client().ApplyURI("mongodb://localhost:27017").SetCompressors([]string{"snappy"})},
		{name: "zlib", opt: options.Client().ApplyURI("mongodb://localhost:27017").SetCompressors([]string{"zlib"})},
		{name: "zstd", opt: options.Client().ApplyURI("mongodb://localhost:27017").SetCompressors([]string{"zstd"})},
	}
	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			client, err := mongo.NewClient(bm.opt)
			if err != nil {
				b.Fatalf("error creating client: %v", err)
			}
			ctx := context.Background()
			err = client.Connect(ctx)
			if err != nil {
				b.Fatalf("error connecting: %v", err)
			}
			defer client.Disconnect(context.Background())
			coll := client.Database("test").Collection("test")
			_, err = coll.DeleteMany(context.Background(), bson.D{})
			if err != nil {
				b.Fatalf("error deleting the document: %v", err)
			}

			b.ResetTimer()
			b.RunParallel(func(p *testing.PB) {
				for p.Next() {
					_, err := coll.InsertOne(context.Background(), bson.D{{"text", text}})
					if err != nil {
						b.Fatalf("error inserting one document: %v", err)
					}
				}
			})
			_, _ = coll.DeleteMany(context.Background(), bson.D{})
		})
	}
}

func BenchmarkClientBulkWrite(b *testing.B) {
	benchmarks := []struct {
		name string
		opt  *options.ClientOptions
	}{
		{name: "not compressed", opt: options.Client().ApplyURI("mongodb://localhost:27017")},
		{name: "snappy", opt: options.Client().ApplyURI("mongodb://localhost:27017").SetCompressors([]string{"snappy"})},
		{name: "zlib", opt: options.Client().ApplyURI("mongodb://localhost:27017").SetCompressors([]string{"zlib"})},
		{name: "zstd", opt: options.Client().ApplyURI("mongodb://localhost:27017").SetCompressors([]string{"zstd"})},
	}
	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			client, err := mongo.NewClient(bm.opt)
			if err != nil {
				b.Fatalf("error creating client: %v", err)
			}
			ctx := context.Background()
			err = client.Connect(ctx)
			if err != nil {
				b.Fatalf("error connecting: %v", err)
			}
			defer client.Disconnect(context.Background())
			coll := client.Database("test").Collection("test")
			_, err = coll.DeleteMany(context.Background(), bson.D{})
			if err != nil {
				b.Fatalf("error deleting the document: %v", err)
			}

			b.ResetTimer()
			b.RunParallel(func(p *testing.PB) {
				for p.Next() {
					_, err := coll.BulkWrite(context.Background(), []mongo.WriteModel{
						mongo.NewInsertOneModel().SetDocument(bson.D{{"text", text}}),
						mongo.NewInsertOneModel().SetDocument(bson.D{{"text", text}}),
						mongo.NewInsertOneModel().SetDocument(bson.D{{"text", text}}),
						mongo.NewInsertOneModel().SetDocument(bson.D{{"text", text}}),
					})
					if err != nil {
						b.Fatalf("error inserting one document: %v", err)
					}
				}
			})
			_, _ = coll.DeleteMany(context.Background(), bson.D{})
		})
	}
}

func BenchmarkClientRead(b *testing.B) {
	benchmarks := []struct {
		name string
		opt  *options.ClientOptions
	}{
		{name: "not compressed", opt: options.Client().ApplyURI("mongodb://localhost:27017")},
		{name: "snappy", opt: options.Client().ApplyURI("mongodb://localhost:27017").SetCompressors([]string{"snappy"})},
		{name: "zlib", opt: options.Client().ApplyURI("mongodb://localhost:27017").SetCompressors([]string{"zlib"})},
		{name: "zstd", opt: options.Client().ApplyURI("mongodb://localhost:27017").SetCompressors([]string{"zstd"})},
	}
	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			client, err := mongo.NewClient(bm.opt)
			if err != nil {
				b.Fatalf("error creating client: %v", err)
			}
			ctx := context.Background()
			err = client.Connect(ctx)
			if err != nil {
				b.Fatalf("error connecting: %v", err)
			}
			defer client.Disconnect(context.Background())
			coll := client.Database("test").Collection("test")
			_, err = coll.DeleteMany(context.Background(), bson.D{})
			if err != nil {
				b.Fatalf("error deleting the document: %v", err)
			}
			_, err = coll.InsertOne(context.Background(), bson.D{{"text", text}})
			if err != nil {
				b.Fatalf("error inserting one document: %v", err)
			}

			b.ResetTimer()
			b.RunParallel(func(p *testing.PB) {
				for p.Next() {
					var res bson.D
					err := coll.FindOne(context.Background(), bson.D{}).Decode(&res)
					if err != nil {
						b.Errorf("error finding one document: %v", err)
					}
				}
			})
		})
	}
}
