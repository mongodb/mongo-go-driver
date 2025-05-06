// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package unified

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// keyPathCtxKey is used as a key for a Context object. The value conveys the BSON key path that is currently being
// compared.
type keyPathCtxKey struct{}

// extraKeysAllowedCtxKey is used as a key for a Context object. The value conveys whether or not the document under
// test can contain extra keys. For example, if the expected document is {x: 1}, the document {x: 1, y: 1} would match
// if the value for this key is true.
type extraKeysAllowedCtxKey struct{}
type extraKeysAllowedRootMatchCtxKey struct{}

func makeMatchContext(ctx context.Context, keyPath string, extraKeysAllowed bool) context.Context {
	ctx = context.WithValue(ctx, keyPathCtxKey{}, keyPath)
	ctx = context.WithValue(ctx, extraKeysAllowedCtxKey{}, extraKeysAllowed)

	// The Root Match Context should be persisted once set.
	if _, ok := ctx.Value(extraKeysAllowedRootMatchCtxKey{}).(bool); !ok {
		ctx = context.WithValue(ctx, extraKeysAllowedRootMatchCtxKey{}, extraKeysAllowed)
	}

	return ctx
}

// verifyValuesMatch compares the provided BSON values and returns an error if they do not match. If the values are
// documents and extraKeysAllowed is true, the actual value will be allowed to have additional keys at the top-level.
// For example, an expected document {x: 1} would match the actual document {x: 1, y: 1}.
func verifyValuesMatch(ctx context.Context, expected, actual bson.RawValue, extraKeysAllowed bool) error {
	return verifyValuesMatchInner(makeMatchContext(ctx, "", extraKeysAllowed), expected, actual)
}

func verifyValuesMatchInner(ctx context.Context, expected, actual bson.RawValue) error {
	keyPath := ctx.Value(keyPathCtxKey{}).(string)
	extraKeysAllowed := ctx.Value(extraKeysAllowedCtxKey{}).(bool)

	if expectedDoc, ok := expected.DocumentOK(); ok {
		// If the root document only has one element and the key is a special matching operator, the actual value might
		// not actually be a document. In this case, evaluate the special operator with the actual value rather than
		// doing an element-wise document comparison.
		if requiresSpecialMatching(expectedDoc) {
			if err := evaluateSpecialComparison(ctx, expectedDoc, actual); err != nil {
				return newMatchingError(keyPath, "error doing special matching assertion: %v", err)
			}
			return nil
		}

		actualDoc, ok := actual.DocumentOK()
		if !ok {
			return newMatchingError(keyPath, "expected value to be a document but got a %s", actual.Type)
		}

		// Perform element-wise comparisons.
		expectedElems, _ := expectedDoc.Elements()
		for _, expectedElem := range expectedElems {
			expectedKey := expectedElem.Key()
			expectedValue := expectedElem.Value()

			fullKeyPath := expectedKey
			if keyPath != "" {
				fullKeyPath = keyPath + "." + expectedKey
			}

			// Get the value from actualDoc here but don't check the error until later because some of the special
			// matching operators can assert that the value isn't present in the document (e.g. $$exists).
			actualValue, err := actualDoc.LookupErr(expectedKey)
			if specialDoc, ok := expectedValue.DocumentOK(); ok && requiresSpecialMatching(specialDoc) {
				// Reset the key path so any errors returned from the function will only have the key path for the
				// target value. Also unconditionally set extraKeysAllowed to false because an assertion like
				// $$unsetOrMatches could recurse back into this function. In that case, the target document is nested
				// and should not have extra keys.
				ctx = makeMatchContext(ctx, "", false)
				if err := evaluateSpecialComparison(ctx, specialDoc, actualValue); err != nil {
					return newMatchingError(fullKeyPath, "error doing special matching assertion: %v", err)
				}
				continue
			}

			// This isn't a special comparison. Assert that the value exists in the actual document.
			if err != nil {
				return newMatchingError(fullKeyPath, "key not found in actual document")
			}

			// Nested documents cannot have extra keys, so we unconditionally pass false for extraKeysAllowed.
			comparisonCtx := makeMatchContext(ctx, fullKeyPath, false)
			if err := verifyValuesMatchInner(comparisonCtx, expectedValue, actualValue); err != nil {
				return err
			}
		}
		// If required, verify that the actual document does not have extra elements. We do this by iterating over the
		// actual and checking for each key in the expected rather than comparing element counts because the presence of
		// special operators can cause incorrect counts. For example, the document {y: {$$exists: false}} has one
		// element, but should match the document {}, which has none.
		if !extraKeysAllowed {
			actualElems, _ := actualDoc.Elements()
			for _, actualElem := range actualElems {
				if _, err := expectedDoc.LookupErr(actualElem.Key()); err != nil {
					return newMatchingError(keyPath, "extra key %q found in actual document %s", actualElem.Key(),
						actualDoc)
				}
			}
		}

		return nil
	}
	if expectedArr, ok := expected.ArrayOK(); ok {
		actualArr, ok := actual.ArrayOK()
		if !ok {
			return newMatchingError(keyPath, "expected value to be an array but got a %s", actual.Type)
		}

		expectedValues, _ := expectedArr.Values()
		actualValues, _ := actualArr.Values()

		// Arrays must always have the same number of elements.
		if len(expectedValues) != len(actualValues) {
			return newMatchingError(keyPath, "expected array length %d, got %d", len(expectedValues),
				len(actualValues))
		}

		for idx, expectedValue := range expectedValues {
			// Use the index as the key to augment the key path.
			fullKeyPath := fmt.Sprintf("%d", idx)
			if keyPath != "" {
				fullKeyPath = keyPath + "." + fullKeyPath
			}

			comparisonCtx := makeMatchContext(ctx, fullKeyPath, extraKeysAllowed)
			err := verifyValuesMatchInner(comparisonCtx, expectedValue, actualValues[idx])
			if err != nil {
				return err
			}
		}

		return nil
	}

	// Numeric values must be considered equal even if their types are different (e.g. if expected is an int32 and
	// actual is an int64).
	if expected.IsNumber() {
		if !actual.IsNumber() {
			return newMatchingError(keyPath, "expected value to be a number but got a %s", actual.Type)
		}

		expectedInt64 := expected.AsInt64()
		actualInt64 := actual.AsInt64()
		if expectedInt64 != actualInt64 {
			return newMatchingError(keyPath, "expected numeric value %d, got %d", expectedInt64, actualInt64)
		}
		return nil
	}

	// If expected is not a recursive or numeric type, we can directly call Equal to do the comparison.
	if !expected.Equal(actual) {
		return newMatchingError(keyPath, "expected value %s, got %s", expected, actual)
	}
	return nil
}

func evaluateSpecialComparison(ctx context.Context, assertionDoc bson.Raw, actual bson.RawValue) error {
	assertionElem := assertionDoc.Index(0)
	assertion := assertionElem.Key()
	assertionVal := assertionElem.Value()
	extraKeysAllowed := ctx.Value(extraKeysAllowedCtxKey{}).(bool)
	extraKeysRootMatchAllowed := ctx.Value(extraKeysAllowedRootMatchCtxKey{}).(bool)

	switch assertion {
	case "$$exists":
		shouldExist := assertionVal.Boolean()
		exists := actual.Validate() == nil
		if shouldExist != exists {
			return fmt.Errorf("expected value to exist: %v; value actually exists: %v", shouldExist, exists)
		}
	case "$$type":
		possibleTypes, err := getTypesArray(assertionVal)
		if err != nil {
			return fmt.Errorf("error getting possible types for a $$type assertion: %v", err)
		}

		for _, possibleType := range possibleTypes {
			if actual.Type == possibleType {
				return nil
			}
		}
		return fmt.Errorf("expected type to be one of %v but was %s", possibleTypes, actual.Type)
	case "$$matchesEntity":
		expected, err := entities(ctx).BSONValue(assertionVal.StringValue())
		if err != nil {
			return err
		}

		// $$matchesEntity doesn't modify the nesting level of the key path so we can propagate ctx without changes.
		return verifyValuesMatchInner(ctx, expected, actual)
	case "$$matchesHexBytes":
		expectedBytes, err := hex.DecodeString(assertionVal.StringValue())
		if err != nil {
			return fmt.Errorf("error converting $$matcesHexBytes value to bytes: %v", err)
		}

		_, actualBytes, ok := actual.BinaryOK()
		if !ok {
			return fmt.Errorf("expected binary value for a $$matchesHexBytes assertion, but got a %s", actual.Type)
		}
		if !bytes.Equal(expectedBytes, actualBytes) {
			return fmt.Errorf("expected bytes %v, got %v", expectedBytes, actualBytes)
		}
	case "$$unsetOrMatches":
		if actual.Validate() != nil {
			return nil
		}

		// $$unsetOrMatches doesn't modify the nesting level or the key path so we can propagate the context to the
		// comparison function without changing anything.
		return verifyValuesMatchInner(ctx, assertionVal, actual)
	case "$$sessionLsid":
		sess, err := entities(ctx).session(assertionVal.StringValue())
		if err != nil {
			return err
		}

		expectedID := sess.ID()
		actualID, ok := actual.DocumentOK()
		if !ok {
			return fmt.Errorf("expected document value for a $$sessionLsid assertion, but got a %s", actual.Type)
		}
		if !bytes.Equal(expectedID, actualID) {
			return fmt.Errorf("expected lsid %v, got %v", expectedID, actualID)
		}
	case "$$lte":
		if assertionVal.Type != bson.TypeInt32 && assertionVal.Type != bson.TypeInt64 {
			return fmt.Errorf("expected assertionVal to be an Int32 or Int64 but got a %s", assertionVal.Type)
		}
		if actual.Type != bson.TypeInt32 && actual.Type != bson.TypeInt64 {
			return fmt.Errorf("expected value to be an Int32 or Int64 but got a %s", actual.Type)
		}

		// Numeric values can be compared even if their types are different (e.g. if expected is an int32 and actual
		// is an int64).
		expectedInt64 := assertionVal.AsInt64()
		actualInt64 := actual.AsInt64()
		if actualInt64 > expectedInt64 {
			return fmt.Errorf("expected numeric value %d to be less than or equal %d", actualInt64, expectedInt64)
		}
		return nil
	case "$$matchAsDocument":
		var actualDoc bson.Raw
		str, ok := actual.StringValueOK()
		if !ok {
			return fmt.Errorf("expected value to be a string but got a %s", actual.Type)
		}

		if err := bson.UnmarshalExtJSON([]byte(str), true, &actualDoc); err != nil {
			return fmt.Errorf("error unmarshalling string as document: %v", err)
		}

		if err := verifyValuesMatch(ctx, assertionVal, documentToRawValue(actualDoc), extraKeysAllowed); err != nil {
			return fmt.Errorf("error matching $$matchAsRoot assertion: %v", err)
		}
	case "$$matchAsRoot":
		// Treat the actual value as a root-level document that can have extra keys that are not subject to
		// the matching rules.
		if err := verifyValuesMatch(ctx, assertionVal, actual, extraKeysRootMatchAllowed); err != nil {
			return fmt.Errorf("error matching $$matchAsRoot assertion: %v", err)
		}
	default:
		return fmt.Errorf("unrecognized special matching assertion %q", assertion)
	}

	return nil
}

func requiresSpecialMatching(doc bson.Raw) bool {
	elems, _ := doc.Elements()
	return len(elems) == 1 && strings.HasPrefix(elems[0].Key(), "$$")
}

func getTypesArray(val bson.RawValue) ([]bson.Type, error) {
	switch val.Type {
	case bson.TypeString:
		convertedType, err := convertStringToBSONType(val.StringValue())
		if err != nil {
			return nil, err
		}

		return []bson.Type{convertedType}, nil
	case bson.TypeArray:
		var typeStrings []string
		if err := val.Unmarshal(&typeStrings); err != nil {
			return nil, fmt.Errorf("error unmarshalling to slice of strings: %v", err)
		}

		var types []bson.Type
		for _, typeStr := range typeStrings {
			convertedType, err := convertStringToBSONType(typeStr)
			if err != nil {
				return nil, err
			}

			types = append(types, convertedType)
		}
		return types, nil
	default:
		return nil, fmt.Errorf("invalid type to convert to bson.Type slice: %s", val.Type)
	}
}

func convertStringToBSONType(typeStr string) (bson.Type, error) {
	switch typeStr {
	case "double":
		return bson.TypeDouble, nil
	case "string":
		return bson.TypeString, nil
	case "object":
		return bson.TypeEmbeddedDocument, nil
	case "array":
		return bson.TypeArray, nil
	case "binData":
		return bson.TypeBinary, nil
	case "undefined":
		return bson.TypeUndefined, nil
	case "objectId":
		return bson.TypeObjectID, nil
	case "bool":
		return bson.TypeBoolean, nil
	case "date":
		return bson.TypeDateTime, nil
	case "null":
		return bson.TypeNull, nil
	case "regex":
		return bson.TypeRegex, nil
	case "dbPointer":
		return bson.TypeDBPointer, nil
	case "javascript":
		return bson.TypeJavaScript, nil
	case "symbol":
		return bson.TypeSymbol, nil
	case "javascriptWithScope":
		return bson.TypeCodeWithScope, nil
	case "int":
		return bson.TypeInt32, nil
	case "timestamp":
		return bson.TypeTimestamp, nil
	case "long":
		return bson.TypeInt64, nil
	case "decimal":
		return bson.TypeDecimal128, nil
	case "minKey":
		return bson.TypeMinKey, nil
	case "maxKey":
		return bson.TypeMaxKey, nil
	default:
		return bson.Type(0), fmt.Errorf("unrecognized BSON type string %q", typeStr)
	}
}

// newMatchingError creates an error to convey that BSON value comparison failed at the provided key path. If the
// key path is empty (e.g. because the values being compared were not documents), the error message will contain the
// phrase "top-level" instead of the path.
func newMatchingError(keyPath, msg string, args ...interface{}) error {
	fullMsg := fmt.Sprintf(msg, args...)
	if keyPath == "" {
		return fmt.Errorf("comparison error at top-level: %s", fullMsg)
	}
	return fmt.Errorf("comparison error at key %q: %s", keyPath, fullMsg)
}
