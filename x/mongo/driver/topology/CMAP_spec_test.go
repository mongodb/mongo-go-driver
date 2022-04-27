// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package topology

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/event"
	testHelpers "go.mongodb.org/mongo-driver/internal/testutil/helpers"
	"go.mongodb.org/mongo-driver/mongo/address"
	"go.mongodb.org/mongo-driver/x/mongo/driver/operation"
)

// skippedTestDescriptions is a collection of test descriptions that the test runner will skip. The
// map format is {"test description": "reason", }.
var skippedTestDescriptions = map[string]string{
	// GODRIVER-1827: These 2 tests assert that in-use connections are not closed until checked
	// back into a closed pool, but the Go connection pool aggressively closes in-use connections.
	// That behavior is currently required by the "Client.Disconnect" API, so skip the tests.
	"When a pool is closed, it MUST first destroy all available connections in that pool": "test requires that close does not aggressively close used connections",
	"must destroy checked in connection if pool has been closed":                          "test requires that close does not aggressively close used connections",
	// GODRIVER-1826: The load-balancer SDAM error handling test "errors during authentication are
	// processed" currently asserts that handshake errors trigger events "pool cleared" then
	// "connection closed". However, the "error during minPoolSize population clears pool" test
	// asserts that handshake errors trigger events "connection closed" then "pool cleared". The Go
	// driver uses the same code path for creating all application connections, so those opposing
	// event orders cannot be satisfied simultaneously.
	// TODO(DRIVERS-1785): Re-enable this test once the spec test is updated to use the same event order as the "errors
	// TODO during authentication are processed" load-balancer SDAM spec test.
	"error during minPoolSize population clears pool": "event ordering is incompatible with load-balancer SDAM spec test (DRIVERS-1785)",
	// GODRIVER-1826: The Go connection pool does not currently always deliver connections created
	// by maintain() to waiting check-outs. There is a race condition between the goroutine started
	// by maintain() to check-in a requested connection and createConnections() picking up the next
	// wantConn created by the waiting check-outs. Most of the time, createConnections() wins and
	// starts creating new connections. That is not a problem for general use cases, but it prevents
	// the "threads blocked by maxConnecting check out minPoolSize connections" test from passing.
	// TODO(DRIVERS-2225): Re-enable this test once the spec test is updated to support the Go pool minPoolSize
	// TODO maintain() behavior.
	"threads blocked by maxConnecting check out minPoolSize connections": "test requires that connections established by minPoolSize are immediately used to satisfy check-out requests (DRIVERS-2225)",
	// GODRIVER-1826: The Go connection pool currently delivers any available connection to the
	// earliest waiting check-out request, independent of if that check-out request already
	// requested a new connection. That behavior is currently incompatible with the "threads blocked
	// by maxConnecting check out returned connections" test, which expects that check-out requests
	// that request a new connection cannot be satisfied by a check-in.
	// TODO(DRIVERS-2223): Re-enable this test once the spec test is updated to support the Go pool check-in behavior.
	"threads blocked by maxConnecting check out returned connections": "test requires a checked-in connections cannot satisfy a check-out waiting on a new connection (DRIVERS-2223)",
}

type cmapEvent struct {
	EventType    string      `json:"type"`
	Address      interface{} `json:"address"`
	ConnectionID uint64      `json:"connectionId"`
	Options      interface{} `json:"options"`
	Reason       string      `json:"reason"`
}

type poolOptions struct {
	MaxPoolSize                int32 `json:"maxPoolSize"`
	MinPoolSize                int32 `json:"minPoolSize"`
	MaxConnecting              int32 `json:"maxConnecting"`
	MaxIdleTimeMS              int32 `json:"maxIdleTimeMS"`
	WaitQueueTimeoutMS         int32 `json:"waitQueueTimeoutMS"`
	BackgroundThreadIntervalMS int32 `json:"backgroundThreadIntervalMS"`
}

type cmapTestFile struct {
	Version     uint64                   `json:"version"`
	Style       string                   `json:"style"`
	Description string                   `json:"description"`
	SkipReason  string                   `json:"skipReason"`
	FailPoint   map[string]interface{}   `json:"failPoint"`
	PoolOptions poolOptions              `json:"poolOptions"`
	Operations  []map[string]interface{} `json:"operations"`
	Error       *cmapTestError           `json:"error"`
	Events      []cmapEvent              `json:"events"`
	Ignore      []string                 `json:"ignore"`
}

type cmapTestError struct {
	ErrorType string `json:"type"`
	Message   string `json:"message"`
	Address   string `json:"address"`
}

type simThread struct {
	JobQueue      chan func()
	JobsAssigned  int32
	JobsCompleted int32
}

type testInfo struct {
	objects                map[string]interface{}
	originalEventChan      chan *event.PoolEvent
	finalEventChan         chan *event.PoolEvent
	threads                map[string]*simThread
	backgroundThreadErrors chan error
	eventCounts            map[string]uint64
	sync.Mutex
}

const cmapTestDir = "../../../../data/connection-monitoring-and-pooling/"

func TestCMAPSpec(t *testing.T) {
	for _, testFileName := range testHelpers.FindJSONFilesInDir(t, cmapTestDir) {
		t.Run(testFileName, func(t *testing.T) {
			runCMAPTest(t, testFileName)
		})
	}
}

func runCMAPTest(t *testing.T, testFileName string) {
	content, err := ioutil.ReadFile(path.Join(cmapTestDir, testFileName))
	testHelpers.RequireNil(t, err, "unable to read content of test file")

	var test cmapTestFile
	err = json.Unmarshal(content, &test)
	testHelpers.RequireNil(t, err, "error unmarshalling testFile %v", err)

	if test.SkipReason != "" {
		t.Skip(test.SkipReason)
	}
	if msg, ok := skippedTestDescriptions[test.Description]; ok {
		t.Skip(msg)
	}

	l, err := net.Listen("tcp", "localhost:0")
	testHelpers.RequireNil(t, err, "unable to create listener: %v", err)

	testInfo := &testInfo{
		objects:                make(map[string]interface{}),
		originalEventChan:      make(chan *event.PoolEvent, 200),
		finalEventChan:         make(chan *event.PoolEvent, 200),
		threads:                make(map[string]*simThread),
		eventCounts:            make(map[string]uint64),
		backgroundThreadErrors: make(chan error, 100),
	}

	sOpts := []ServerOption{
		WithMaxConnections(func(uint64) uint64 {
			return uint64(test.PoolOptions.MaxPoolSize)
		}),
		WithMinConnections(func(uint64) uint64 {
			return uint64(test.PoolOptions.MinPoolSize)
		}),
		WithMaxConnecting(func(uint64) uint64 {
			return uint64(test.PoolOptions.MaxConnecting)
		}),
		WithConnectionPoolMaxIdleTime(func(time.Duration) time.Duration {
			return time.Duration(test.PoolOptions.MaxIdleTimeMS) * time.Millisecond
		}),
		WithConnectionPoolMaintainInterval(func(time.Duration) time.Duration {
			return time.Duration(test.PoolOptions.BackgroundThreadIntervalMS) * time.Millisecond
		}),
		WithConnectionPoolMonitor(func(*event.PoolMonitor) *event.PoolMonitor {
			return &event.PoolMonitor{
				Event: func(evt *event.PoolEvent) { testInfo.originalEventChan <- evt },
			}
		}),
	}

	// If there's a failpoint configured in the test, use a dialer that returns connections that
	// mock the configured failpoint. If "blockConnection" is true and "blockTimeMS" is specified,
	// use a mock connection that delays reads by the configured amount. If "closeConnection" is
	// true, close the connection so it always returns an error on read and write.
	if test.FailPoint != nil {
		data, ok := test.FailPoint["data"].(map[string]interface{})
		if !ok {
			t.Fatalf("expected to find \"data\" map in failPoint (%v)", test.FailPoint)
		}

		var delay time.Duration
		blockConnection, _ := data["blockConnection"].(bool)
		if blockTimeMS, ok := data["blockTimeMS"].(float64); ok && blockConnection {
			delay = time.Duration(blockTimeMS) * time.Millisecond
		}

		closeConnection, _ := data["closeConnection"].(bool)

		sOpts = append(sOpts, WithConnectionOptions(func(...ConnectionOption) []ConnectionOption {
			return []ConnectionOption{
				WithDialer(func(Dialer) Dialer {
					return DialerFunc(func(_ context.Context, _, _ string) (net.Conn, error) {
						msc := newMockSlowConn(makeHelloReply(), delay)
						if closeConnection {
							msc.Close()
						}
						return msc, nil
					})
				}),
				WithHandshaker(func(h Handshaker) Handshaker {
					return operation.NewHello()
				}),
			}
		}))
	}

	s := NewServer(address.Address(l.Addr().String()), primitive.NewObjectID(), sOpts...)
	s.state = serverConnected
	testHelpers.RequireNil(t, err, "error connecting connection pool: %v", err)
	defer s.pool.close(context.Background())

	for _, op := range test.Operations {
		if tempErr := runOperation(t, op, testInfo, s, test.PoolOptions.WaitQueueTimeoutMS); tempErr != nil {
			if err != nil {
				t.Fatalf("received multiple errors in primary thread: %v and %v", err, tempErr)
			}
			err = tempErr
		}
	}

	// make sure all threads have finished
	testInfo.Lock()
	threadNames := make([]string, 0)
	for threadName := range testInfo.threads {
		threadNames = append(threadNames, threadName)
	}
	testInfo.Unlock()

	for _, threadName := range threadNames {
	WAIT:
		for {
			testInfo.Lock()
			thread, ok := testInfo.threads[threadName]
			if !ok {
				t.Fatalf("thread was unexpectedly ended: %v", threadName)
			}
			if len(thread.JobQueue) == 0 && atomic.LoadInt32(&thread.JobsCompleted) == atomic.LoadInt32(&thread.JobsAssigned) {
				break WAIT
			}
			testInfo.Unlock()
		}
		close(testInfo.threads[threadName].JobQueue)
		testInfo.Unlock()
	}

	if test.Error != nil {
		if err == nil || strings.ToLower(test.Error.Message) != err.Error() {
			var erroredCorrectly bool
			errs := make([]error, 0, len(testInfo.backgroundThreadErrors)+1)
			errs = append(errs, err)
			for len(testInfo.backgroundThreadErrors) > 0 {
				bgErr := <-testInfo.backgroundThreadErrors
				errs = append(errs, bgErr)
				if bgErr != nil && strings.Contains(bgErr.Error(), strings.ToLower(test.Error.Message)) {
					erroredCorrectly = true
					break
				}
			}
			if !erroredCorrectly {
				t.Fatalf("error differed from expected error, expected: %v, actual errors received: %v", test.Error.Message, errs)
			}
		}
	}

	testInfo.Lock()
	defer testInfo.Unlock()
	for len(testInfo.originalEventChan) > 0 {
		temp := <-testInfo.originalEventChan
		testInfo.finalEventChan <- temp
	}

	checkEvents(t, test.Events, testInfo.finalEventChan, test.Ignore)

}

func checkEvents(t *testing.T, expectedEvents []cmapEvent, actualEvents chan *event.PoolEvent, ignoreEvents []string) {
	for _, expectedEvent := range expectedEvents {
		validEvent := nextValidEvent(t, actualEvents, ignoreEvents)

		if expectedEvent.EventType != validEvent.Type {
			var reason string
			if validEvent.Type == "ConnectionCheckOutFailed" {
				reason = ": " + validEvent.Reason
			}
			t.Errorf("unexpected event occurred: expected: %v, actual: %v%v", expectedEvent.EventType, validEvent.Type, reason)
		}

		if expectedEvent.Address != nil {

			if expectedEvent.Address == float64(42) { // can be any address
				if validEvent.Address == "" {
					t.Errorf("expected address in event, instead received none in %v", expectedEvent.EventType)
				}
			} else { // must be specific address
				addr, ok := expectedEvent.Address.(string)
				if !ok {
					t.Errorf("received non string address: %v", expectedEvent.Address)
				}
				if addr != validEvent.Address {
					t.Errorf("received unexpected address: %v, expected: %v", validEvent.Address, expectedEvent.Address)
				}
			}
		}

		if expectedEvent.ConnectionID != 0 {
			if expectedEvent.ConnectionID == 42 {
				if validEvent.ConnectionID == 0 {
					t.Errorf("expected a connectionId but found none in %v", validEvent.Type)
				}
			} else if expectedEvent.ConnectionID != validEvent.ConnectionID {
				t.Errorf("expected and actual connectionIds differed: expected: %v, actual: %v for event: %v", expectedEvent.ConnectionID, validEvent.ConnectionID, expectedEvent.EventType)
			}
		}

		if expectedEvent.Reason != "" && expectedEvent.Reason != validEvent.Reason {
			t.Errorf("event reason differed from expected: expected: %v, actual: %v for %v", expectedEvent.Reason, validEvent.Reason, expectedEvent.EventType)
		}

		if expectedEvent.Options != nil {
			if expectedEvent.Options == float64(42) {
				if validEvent.PoolOptions == nil {
					t.Errorf("expected poolevent options but found none")
				}
			} else {
				opts, ok := expectedEvent.Options.(map[string]interface{})
				if !ok {
					t.Errorf("event options were unexpected type: %T for %v", expectedEvent.Options, expectedEvent.EventType)
				}

				if maxSize, ok := opts["maxPoolSize"]; ok && validEvent.PoolOptions.MaxPoolSize != uint64(maxSize.(float64)) {
					t.Errorf("event's max pool size differed from expected: %v, actual: %v", maxSize, validEvent.PoolOptions.MaxPoolSize)
				}

				if minSize, ok := opts["minPoolSize"]; ok && validEvent.PoolOptions.MinPoolSize != uint64(minSize.(float64)) {
					t.Errorf("event's min pool size differed from expected: %v, actual: %v", minSize, validEvent.PoolOptions.MinPoolSize)
				}

				if waitQueueTimeoutMS, ok := opts["waitQueueTimeoutMS"]; ok && validEvent.PoolOptions.WaitQueueTimeoutMS != uint64(waitQueueTimeoutMS.(float64)) {
					t.Errorf("event's min pool size differed from expected: %v, actual: %v", waitQueueTimeoutMS, validEvent.PoolOptions.WaitQueueTimeoutMS)
				}
			}
		}
	}
}

func nextValidEvent(t *testing.T, events chan *event.PoolEvent, ignoreEvents []string) *event.PoolEvent {
	t.Helper()
NextEvent:
	for {
		if len(events) == 0 {
			t.Fatalf("unable to get next event. too few events occurred")
		}

		event := <-events
		for _, Type := range ignoreEvents {
			if event.Type == Type {
				continue NextEvent
			}
		}
		return event
	}
}

func runOperation(t *testing.T, operation map[string]interface{}, testInfo *testInfo, s *Server, checkOutTimeout int32) error {
	threadName, ok := operation["thread"]
	if ok { // to be run in background thread
		testInfo.Lock()
		thread, ok := testInfo.threads[threadName.(string)]
		if !ok {
			thread = &simThread{
				JobQueue: make(chan func(), 200),
			}
			testInfo.threads[threadName.(string)] = thread

			go func() {
				for {
					job, more := <-thread.JobQueue
					if !more {
						break
					}
					job()
					atomic.AddInt32(&thread.JobsCompleted, 1)
				}
			}()
		}
		testInfo.Unlock()

		atomic.AddInt32(&thread.JobsAssigned, 1)
		thread.JobQueue <- func() {
			err := runOperationInThread(t, operation, testInfo, s, checkOutTimeout)
			testInfo.backgroundThreadErrors <- err
		}

		return nil // since we don't care about errors occurring in non primary threads
	}
	return runOperationInThread(t, operation, testInfo, s, checkOutTimeout)
}

func runOperationInThread(t *testing.T, operation map[string]interface{}, testInfo *testInfo, s *Server, checkOutTimeout int32) error {
	name, ok := operation["name"]
	if !ok {
		t.Fatalf("unable to find name in operation")
	}

	switch name {
	case "start":
		return nil // we dont need to start another thread since this has already been done in runOperation
	case "wait":
		timeMs, ok := operation["ms"]
		if !ok {
			t.Fatalf("unable to find ms in wait operation")
		}
		dur := time.Duration(int64(timeMs.(float64))) * time.Millisecond
		time.Sleep(dur)
	case "waitForThread":
		threadName, ok := operation["target"]
		if !ok {
			t.Fatalf("unable to waitForThread without specified threadName")
		}

		testInfo.Lock()
		thread, ok := testInfo.threads[threadName.(string)]
		testInfo.Unlock()
		if !ok {
			t.Fatalf("unable to find thread to wait for: %v", threadName)
		}

		for {
			if atomic.LoadInt32(&thread.JobsCompleted) == atomic.LoadInt32(&thread.JobsAssigned) {
				break
			}
		}
	case "waitForEvent":
		var targetCount int
		{
			f, ok := operation["count"].(float64)
			if !ok {
				t.Fatalf("count is required to waitForEvent")
			}
			targetCount = int(f)
		}

		targetEventName, ok := operation["event"].(string)
		if !ok {
			t.Fatalf("event is require to waitForEvent")
		}

		// If there is a timeout specified in the "waitForEvent" operation, then use that timeout.
		// Otherwise, use a default timeout of 10s when waiting for events. Using a default timeout
		// prevents the Go test runner from timing out, which just prints a stack trace and no
		// information about what event the test was waiting for.
		timeout := 10 * time.Second
		if timeoutMS, ok := operation["timeout"].(float64); ok {
			timeout = time.Duration(timeoutMS) * time.Millisecond
		}

		originalChan := testInfo.originalEventChan
		finalChan := testInfo.finalEventChan

		for {
			var event *event.PoolEvent
			{
				timer := time.NewTimer(timeout)
				select {
				case event = <-originalChan:
				case <-timer.C:
					t.Fatalf("timed out waiting for %d %q events", targetCount, targetEventName)
				}
				timer.Stop()
			}
			finalChan <- event

			testInfo.Lock()
			_, ok = testInfo.eventCounts[event.Type]
			if !ok {
				testInfo.eventCounts[event.Type] = 0
			}
			testInfo.eventCounts[event.Type]++
			count := testInfo.eventCounts[event.Type]
			testInfo.Unlock()

			if event.Type == targetEventName && count == uint64(targetCount) {
				break
			}
		}
	case "checkOut":
		checkoutContext := context.Background()
		if checkOutTimeout != 0 {
			var cancel context.CancelFunc
			checkoutContext, cancel = context.WithTimeout(context.Background(), time.Duration(checkOutTimeout)*time.Millisecond)
			defer cancel()
		}

		c, err := s.Connection(checkoutContext)
		if label, ok := operation["label"]; ok {
			testInfo.Lock()
			testInfo.objects[label.(string)] = c
			testInfo.Unlock()
		}

		return err
	case "checkIn":
		cName, ok := operation["connection"]
		if !ok {
			t.Fatalf("unable to find connection to checkin")
		}

		var cEmptyInterface interface{}
		testInfo.Lock()
		cEmptyInterface, ok = testInfo.objects[cName.(string)]
		delete(testInfo.objects, cName.(string))
		testInfo.Unlock()
		if !ok {
			t.Fatalf("was unable to find %v in objects when expected", cName)
		}

		c, ok := cEmptyInterface.(*Connection)
		if !ok {
			t.Fatalf("object in objects was expected to be a connection, but was instead a %T", cEmptyInterface)
		}
		return c.Close()
	case "clear":
		s.pool.clear(nil, nil)
	case "close":
		s.pool.close(context.Background())
	case "ready":
		return s.pool.ready()
	default:
		t.Fatalf("unknown operation: %v", name)
	}

	return nil
}
