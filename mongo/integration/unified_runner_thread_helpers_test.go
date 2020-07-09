// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"fmt"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

// Helper functions for the operations in the unified spec test runner that require creating and synchronizing
// background goroutines.

// backgroundRoutine represents a background goroutine that can execute operations. The goroutine reads operations from
// a channel and executes them in order. It exits when an operation with name exitRoutineOperationName is read. If any
// of the operations error, all future operations passed to the routine are skipped. The first error is reported by the
// stop() function.
type backgroundRoutine struct {
	operations chan *operation
	mt         *mtest.T
	testCase   *testCase
	wg         sync.WaitGroup
	err        error
}

func newBackgroundRoutine(mt *mtest.T, testCase *testCase) *backgroundRoutine {
	routine := &backgroundRoutine{
		operations: make(chan *operation, 10),
		mt:         mt,
		testCase:   testCase,
	}

	return routine
}

func (b *backgroundRoutine) start() {
	b.wg.Add(1)

	go func() {
		defer b.wg.Done()

		for op := range b.operations {
			if b.err != nil {
				continue
			}

			if err := runOperation(b.mt, b.testCase, op, nil, nil); err != nil {
				b.err = fmt.Errorf("error running operation %s: %v", op.Name, err)
			}
		}
	}()
}

func (b *backgroundRoutine) stop() error {
	close(b.operations)
	b.wg.Wait()
	return b.err
}

func (b *backgroundRoutine) addOperation(op *operation) bool {
	select {
	case b.operations <- op:
		return true
	default:
		return false
	}
}

func startThread(mt *mtest.T, testCase *testCase, op *operation) {
	routine := newBackgroundRoutine(mt, testCase)
	testCase.routinesMap.Store(getThreadName(op), routine)
	routine.start()
}

func runOnThread(mt *mtest.T, testCase *testCase, op *operation) {
	routineName := getThreadName(op)
	routineVal, ok := testCase.routinesMap.Load(routineName)
	assert.True(mt, ok, "no background routine found with name %s", routineName)
	routine := routineVal.(*backgroundRoutine)

	var routineOperation operation
	operationDoc := op.Arguments.Lookup("operation")
	err := bson.UnmarshalWithRegistry(specTestRegistry, operationDoc.Document(), &routineOperation)
	assert.Nil(mt, err, "error creating operation for runOnThread: %v", err)

	ok = routine.addOperation(&routineOperation)
	assert.True(mt, ok, "failed to add operation %s to routine %s", routineOperation.Name, routineName)
}

func waitForThread(mt *mtest.T, testCase *testCase, op *operation) {
	name := getThreadName(op)
	routineVal, ok := testCase.routinesMap.Load(name)
	assert.True(mt, ok, "no background routine found with name %s", name)

	err := routineVal.(*backgroundRoutine).stop()
	testCase.routinesMap.Delete(name)
	assert.Nil(mt, err, "error on background routine %s: %v", name, err)
}

func getThreadName(op *operation) string {
	return op.Arguments.Lookup("name").StringValue()
}
