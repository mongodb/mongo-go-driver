// Copyright (C) MongoDB, Inc. 2021-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// finder is an object that implements FindOne and Find.
type finder interface {
	FindOne(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) *mongo.SingleResult
	Find(context.Context, interface{}, ...*options.FindOptions) (*mongo.Cursor, error)
}

// mockFinder implements finder.
type mockFinder struct {
	docs     []interface{}
	err      error
	registry *bsoncodec.Registry
}

// FindOne mocks a findOne operation using NewSingleResultFromDocument.
func (mf *mockFinder) FindOne(_ context.Context, _ interface{}, _ ...*options.FindOneOptions) *mongo.SingleResult {
	return mongo.NewSingleResultFromDocument(mf.docs[0], mf.err, mf.registry)
}

// Find mocks a find operation using NewCursorFromDocuments.
func (mf *mockFinder) Find(context.Context, interface{}, ...*options.FindOptions) (*mongo.Cursor, error) {
	return mongo.NewCursorFromDocuments(mf.docs, mf.err, mf.registry)
}

// ShopItem is an item with an associated ID and price.
type ShopItem struct {
	ID    int     `bson:"id"`
	Price float64 `bson:"price"`
}

// getItem is an example function using the interface finder to test the mocking of SingleResult.
func getItem(f finder, id int) (*ShopItem, error) {
	res := f.FindOne(context.Background(), bson.D{{"id", id}})
	var item ShopItem
	err := res.Decode(&item)
	return &item, err
}

// getItems is an example function using the interface finder to test the mocking of Cursor.
func getItems(f finder) ([]ShopItem, error) {
	cur, err := f.Find(context.Background(), bson.D{})
	if err != nil {
		return nil, err
	}

	var items []ShopItem
	err = cur.All(context.Background(), &items)
	return items, err
}

func TestMockFind(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().CreateClient(false))
	defer mt.Close()

	insertItems := []interface{}{
		ShopItem{ID: 0, Price: 1.5},
		ShopItem{ID: 1, Price: 5.7},
		ShopItem{ID: 2, Price: 0.25},
	}
	insertItem := []interface{}{ShopItem{ID: 1, Price: 5.7}}

	mt.Run("mongo.Collection can be passed as interface", func(mt *mtest.T) {
		// Actually insert documents to collection.
		_, err := mt.Coll.InsertMany(context.Background(), insertItems)
		assert.Nil(mt, err, "InsertMany error: %v", err)

		// Assert that FindOne behaves as expected.
		shopItem, err := getItem(mt.Coll, 1)
		assert.Nil(mt, err, "getItem error: %v", err)
		assert.Equal(mt, 1, shopItem.ID)
		assert.Equal(mt, 5.7, shopItem.Price)

		// Assert that Find behaves as expected.
		shopItems, err := getItems(mt.Coll)
		assert.Nil(mt, err, "getItems error: %v", err)
		for i, shopItem := range shopItems {
			expectedItem := insertItems[i].(ShopItem)
			assert.Equal(mt, expectedItem.ID, shopItem.ID)
			assert.Equal(mt, expectedItem.Price, shopItem.Price)
		}
	})

	mt.Run("FindOne can be mocked", func(mt *mtest.T) {
		// Mock a FindOne result with mockFinder.
		mf := &mockFinder{docs: insertItem, err: nil, registry: nil}

		// Assert that FindOne behaves as expected.
		shopItem, err := getItem(mf, 1)
		assert.Nil(mt, err, "getItem error: %v", err)
		assert.Equal(mt, 1, shopItem.ID)
		assert.Equal(mt, 5.7, shopItem.Price)
	})

	mt.Run("Find can be mocked", func(mt *mtest.T) {
		// Mock a Find result with mockFinder.
		mf := &mockFinder{docs: insertItems, err: nil, registry: nil}

		// Assert that Find behaves as expected.
		shopItems, err := getItems(mf)
		assert.Nil(mt, err, "getItems error: %v", err)
		for i, shopItem := range shopItems {
			expectedItem := insertItems[i].(ShopItem)
			assert.Equal(mt, expectedItem.ID, shopItem.ID)
			assert.Equal(mt, expectedItem.Price, shopItem.Price)
		}
	})
}
