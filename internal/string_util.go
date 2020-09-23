// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package internal

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
)

// StringSliceFromRawElement decodes the provided BSON element into a []string. This internally calls
// StringSliceFromRawValue on the element's value. The error conditions outlined in that function's documentation
// apply for this function as well.
func StringSliceFromRawElement(element bson.RawElement, name string) ([]string, error) {
	return StringSliceFromRawValue(element.Value(), name)
}

// StringSliceFromRawValue decodes the provided BSON value into a []string. This function returns an error if the
// value is not an array or any of the elements in the array are not strings.
func StringSliceFromRawValue(val bson.RawValue, name string) ([]string, error) {
	arr, ok := val.ArrayOK()
	if !ok {
		return nil, fmt.Errorf("expected '%s' to be an array but it's a BSON %s", name, val.Type)
	}
	vals, err := arr.Values()
	if err != nil {
		return nil, err
	}
	var strs []string
	for _, val := range vals {
		str, ok := val.StringValueOK()
		if !ok {
			return nil, fmt.Errorf("expected '%s' to be an array of strings, but found a BSON %s", name, val.Type)
		}
		strs = append(strs, str)
	}
	return strs, nil
}
