package unified

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// setCreateKeyDKO will parse an options document and set the data on a options.DataKeyOptions instance.
func setCreateKeyDKO(dko *options.DataKeyOptions, optsDocElem bson.RawElement) error {
	optsDoc, err := optsDocElem.Value().Document().Elements()
	if err != nil {
		return err
	}
	for _, elem := range optsDoc {
		key := elem.Key()
		val := elem.Value()
		switch key {
		case "masterKey":
			masterKey := make(map[string]interface{})
			if err := val.Unmarshal(&masterKey); err != nil {
				return fmt.Errorf("error unmarshaling 'masterKey': %v", err)
			}
			dko.SetMasterKey(masterKey)
		case "keyAltNames":
			keyAltNames := []string{}
			if err := val.Unmarshal(&keyAltNames); err != nil {
				return fmt.Errorf("error unmarshaling 'keyAltNames': %v", err)
			}
			dko.SetKeyAltNames(keyAltNames)
		case "keyMaterial":
			bin := primitive.Binary{}
			if err := val.Unmarshal(&bin); err != nil {
				return fmt.Errorf("error unmarshaling 'keyMaterial': %v", err)
			}
			dko.SetKeyMaterial(bin.Data)
		default:
			return fmt.Errorf("unrecognized DataKeyOptions arg: %q", key)
		}
	}
	return nil
}

// executeCreateKey will attempt to create a client-encrypted key for a unified operation.
func executeCreateKey(ctx context.Context, operation *operation) (*operationResult, error) {
	cee, err := entities(ctx).clientEncryption(operation.Object)
	if err != nil {
		return nil, err
	}

	var kmsProvider string
	dko := options.DataKey()

	elems, err := operation.Arguments.Elements()
	if err != nil {
		return nil, err
	}
	for _, elem := range elems {
		key := elem.Key()
		val := elem.Value()

		switch key {
		case "kmsProvider":
			kmsProvider = val.StringValue()
		case "opts":
			setCreateKeyDKO(dko, elem)
		default:
			return nil, fmt.Errorf("unrecognized CreateKey arg: %q", key)
		}
	}
	if kmsProvider == "" {
		return nil, newMissingArgumentError("kmsProvider")
	}

	bin, err := cee.CreateKey(ctx, kmsProvider, dko)
	return newValueResult(bsontype.Binary, bin.Data, err), nil
}
