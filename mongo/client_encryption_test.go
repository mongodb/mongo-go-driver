package mongo

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestClientEncryption(t *testing.T) {
	ce := &ClientEncryption{closed: true}

	t.Run("CreateEncryptedCollection", func(t *testing.T) {
		// used options.CreateCollection() to avoid catching the nil CreateCollectionOptions error
		_, _, err := ce.CreateEncryptedCollection(context.TODO(), nil, "", options.CreateCollection(), "", nil)
		assert.Equal(t, ErrClientDisconnected, err, "expected error %v, got %v", ErrClientDisconnected, err)
	})
	t.Run("AddKeyAltName", func(t *testing.T) {
		err := ce.AddKeyAltName(context.TODO(), bson.Binary{}, "").err
		assert.Equal(t, ErrClientDisconnected, err, "expected error %v, got %v", ErrClientDisconnected, err)
	})
	t.Run("CreateDataKey", func(t *testing.T) {
		_, err := ce.CreateDataKey(context.TODO(), "", options.DataKey())
		assert.Equal(t, ErrClientDisconnected, err, "expected error %v, got %v", ErrClientDisconnected, err)
	})
	t.Run("Encrypt", func(t *testing.T) {
		_, err := ce.Encrypt(context.TODO(), bson.RawValue{}, options.Encrypt())
		assert.Equal(t, ErrClientDisconnected, err, "expected error %v, got %v", ErrClientDisconnected, err)
	})
	t.Run("EncryptExpression", func(t *testing.T) {
		err := ce.EncryptExpression(context.TODO(), nil, nil, options.Encrypt())
		assert.Equal(t, ErrClientDisconnected, err, "expected error %v, got %v", ErrClientDisconnected, err)
	})
	t.Run("Decrypt", func(t *testing.T) {
		_, err := ce.Decrypt(context.TODO(), bson.Binary{})
		assert.Equal(t, ErrClientDisconnected, err, "expected error %v, got %v", ErrClientDisconnected, err)
	})
	t.Run("Close", func(t *testing.T) {
		err := ce.Close(context.TODO())
		assert.Equal(t, ErrClientDisconnected, err, "expected error %v, got %v", ErrClientDisconnected, err)
	})
	t.Run("DeleteKey", func(t *testing.T) {
		_, err := ce.DeleteKey(context.TODO(), bson.Binary{})
		assert.Equal(t, ErrClientDisconnected, err, "expected error %v, got %v", ErrClientDisconnected, err)
	})
	t.Run("GetKeyByAltName", func(t *testing.T) {
		err := ce.GetKeyByAltName(context.TODO(), "").err
		assert.Equal(t, ErrClientDisconnected, err, "expected error %v, got %v", ErrClientDisconnected, err)
	})
	t.Run("GetKey", func(t *testing.T) {
		err := ce.GetKey(context.TODO(), bson.Binary{}).err
		assert.Equal(t, ErrClientDisconnected, err, "expected error %v, got %v", ErrClientDisconnected, err)
	})
	t.Run("GetKeys", func(t *testing.T) {
		_, err := ce.GetKeys(context.TODO())
		assert.Equal(t, ErrClientDisconnected, err, "expected error %v, got %v", ErrClientDisconnected, err)
	})
	t.Run("RemoveKeyAltName", func(t *testing.T) {
		err := ce.RemoveKeyAltName(context.TODO(), bson.Binary{}, "").err
		assert.Equal(t, ErrClientDisconnected, err, "expected error %v, got %v", ErrClientDisconnected, err)
	})
	t.Run("RewrapManyDataKey", func(t *testing.T) {
		_, err := ce.RewrapManyDataKey(context.TODO(), nil, options.RewrapManyDataKey())
		assert.Equal(t, ErrClientDisconnected, err, "expected error %v, got %v", ErrClientDisconnected, err)
	})
}
