// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package auth

import (
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/internal/assert"
)

func TestGetRegion(t *testing.T) {
	longHost := make([]rune, 256)
	emptyErr := errors.New("invalid STS host: empty")
	tooLongErr := errors.New("invalid STS host: too large")
	emptyPartErr := errors.New("invalid STS host: empty part")
	testCases := []struct {
		name   string
		host   string
		err    error
		region string
	}{
		{"success default", "sts.amazonaws.com", nil, "us-east-1"},
		{"success parse", "first.second", nil, "second"},
		{"success no region", "first", nil, "us-east-1"},
		{"error host too long", string(longHost), tooLongErr, ""},
		{"error host empty", "", emptyErr, ""},
		{"error empty middle part", "abc..def", emptyPartErr, ""},
		{"error empty part", "first.", emptyPartErr, ""},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reg, err := getRegion(tc.host)
			if tc.err == nil {
				assert.Nil(t, err, "error getting region: %v", err)
				assert.Equal(t, tc.region, reg, "expected %v, got %v", tc.region, reg)
				return
			}
			assert.NotNil(t, err, "expected error, got nil")
			assert.Equal(t, err, tc.err, "expected error: %v, got: %v", tc.err, err)
		})
	}

}
