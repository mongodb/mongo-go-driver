// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package prose

import (
	"context"
	"testing"
)

func TestCSE(t *testing.T) {
	if !*cseFlag {
		t.Skip("pass -cse to run CSE tests")
	}

	exit, out, err := execCSE(context.Background(), "go test -tags cse ./x/mongo/driver/mongocrypt")
	if err != nil {
		t.Fatalf("failed to run CSE tests: %v", err)
	}
	if exit != 0 {
		t.Fatalf("go test failed with exit code %d:\n%s", exit, out)
	}
}
