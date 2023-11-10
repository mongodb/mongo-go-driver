// Copied from https://github.com/stretchr/testify/blob/1333b5d3bda8cf5aedcf3e1aaa95cac28aaab892/require/require.go

// Copyright 2020 Mat Ryer, Tyler Bunnell and all contributors. All rights reserved.
// Use of this source code is governed by an MIT-style license that can be found in
// the THIRD-PARTY-NOTICES file.

package require

import (
	time "time"

	assert "go.mongodb.org/mongo-driver/internal/assert"
)

// TestingT is an interface wrapper around *testing.T
type TestingT interface {
	Errorf(format string, args ...interface{})
	FailNow()
}

type tHelper interface {
	Helper()
}

// Contains asserts that the specified string, list(array, slice...) or map contains the
// specified substring or element.
//
//	assert.Contains(t, "Hello World", "World")
//	assert.Contains(t, ["Hello", "World"], "World")
//	assert.Contains(t, {"Hello": "World"}, "Hello")
func Contains(t TestingT, s interface{}, contains interface{}, msgAndArgs ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.Contains(t, s, contains, msgAndArgs...) {
		return
	}
	t.FailNow()
}

// Containsf asserts that the specified string, list(array, slice...) or map contains the
// specified substring or element.
//
//	assert.Containsf(t, "Hello World", "World", "error message %s", "formatted")
//	assert.Containsf(t, ["Hello", "World"], "World", "error message %s", "formatted")
//	assert.Containsf(t, {"Hello": "World"}, "Hello", "error message %s", "formatted")
func Containsf(t TestingT, s interface{}, contains interface{}, msg string, args ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.Containsf(t, s, contains, msg, args...) {
		return
	}
	t.FailNow()
}

// ElementsMatch asserts that the specified listA(array, slice...) is equal to specified
// listB(array, slice...) ignoring the order of the elements. If there are duplicate elements,
// the number of appearances of each of them in both lists should match.
//
// assert.ElementsMatch(t, [1, 3, 2, 3], [1, 3, 3, 2])
func ElementsMatch(t TestingT, listA interface{}, listB interface{}, msgAndArgs ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.ElementsMatch(t, listA, listB, msgAndArgs...) {
		return
	}
	t.FailNow()
}

// ElementsMatchf asserts that the specified listA(array, slice...) is equal to specified
// listB(array, slice...) ignoring the order of the elements. If there are duplicate elements,
// the number of appearances of each of them in both lists should match.
//
// assert.ElementsMatchf(t, [1, 3, 2, 3], [1, 3, 3, 2], "error message %s", "formatted")
func ElementsMatchf(t TestingT, listA interface{}, listB interface{}, msg string, args ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.ElementsMatchf(t, listA, listB, msg, args...) {
		return
	}
	t.FailNow()
}

// Equal asserts that two objects are equal.
//
//	assert.Equal(t, 123, 123)
//
// Pointer variable equality is determined based on the equality of the
// referenced values (as opposed to the memory addresses). Function equality
// cannot be determined and will always fail.
func Equal(t TestingT, expected interface{}, actual interface{}, msgAndArgs ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.Equal(t, expected, actual, msgAndArgs...) {
		return
	}
	t.FailNow()
}

// EqualError asserts that a function returned an error (i.e. not `nil`)
// and that it is equal to the provided error.
//
//	actualObj, err := SomeFunction()
//	assert.EqualError(t, err,  expectedErrorString)
func EqualError(t TestingT, theError error, errString string, msgAndArgs ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.EqualError(t, theError, errString, msgAndArgs...) {
		return
	}
	t.FailNow()
}

// EqualErrorf asserts that a function returned an error (i.e. not `nil`)
// and that it is equal to the provided error.
//
//	actualObj, err := SomeFunction()
//	assert.EqualErrorf(t, err,  expectedErrorString, "error message %s", "formatted")
func EqualErrorf(t TestingT, theError error, errString string, msg string, args ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.EqualErrorf(t, theError, errString, msg, args...) {
		return
	}
	t.FailNow()
}

// EqualValues asserts that two objects are equal or convertible to the same types
// and equal.
//
//	assert.EqualValues(t, uint32(123), int32(123))
func EqualValues(t TestingT, expected interface{}, actual interface{}, msgAndArgs ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.EqualValues(t, expected, actual, msgAndArgs...) {
		return
	}
	t.FailNow()
}

// EqualValuesf asserts that two objects are equal or convertible to the same types
// and equal.
//
//	assert.EqualValuesf(t, uint32(123), int32(123), "error message %s", "formatted")
func EqualValuesf(t TestingT, expected interface{}, actual interface{}, msg string, args ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.EqualValuesf(t, expected, actual, msg, args...) {
		return
	}
	t.FailNow()
}

// Equalf asserts that two objects are equal.
//
//	assert.Equalf(t, 123, 123, "error message %s", "formatted")
//
// Pointer variable equality is determined based on the equality of the
// referenced values (as opposed to the memory addresses). Function equality
// cannot be determined and will always fail.
func Equalf(t TestingT, expected interface{}, actual interface{}, msg string, args ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.Equalf(t, expected, actual, msg, args...) {
		return
	}
	t.FailNow()
}

// Error asserts that a function returned an error (i.e. not `nil`).
//
//	  actualObj, err := SomeFunction()
//	  if assert.Error(t, err) {
//		   assert.Equal(t, expectedError, err)
//	  }
func Error(t TestingT, err error, msgAndArgs ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.Error(t, err, msgAndArgs...) {
		return
	}
	t.FailNow()
}

// ErrorContains asserts that a function returned an error (i.e. not `nil`)
// and that the error contains the specified substring.
//
//	actualObj, err := SomeFunction()
//	assert.ErrorContains(t, err,  expectedErrorSubString)
func ErrorContains(t TestingT, theError error, contains string, msgAndArgs ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.ErrorContains(t, theError, contains, msgAndArgs...) {
		return
	}
	t.FailNow()
}

// ErrorContainsf asserts that a function returned an error (i.e. not `nil`)
// and that the error contains the specified substring.
//
//	actualObj, err := SomeFunction()
//	assert.ErrorContainsf(t, err,  expectedErrorSubString, "error message %s", "formatted")
func ErrorContainsf(t TestingT, theError error, contains string, msg string, args ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.ErrorContainsf(t, theError, contains, msg, args...) {
		return
	}
	t.FailNow()
}

// Errorf asserts that a function returned an error (i.e. not `nil`).
//
//	  actualObj, err := SomeFunction()
//	  if assert.Errorf(t, err, "error message %s", "formatted") {
//		   assert.Equal(t, expectedErrorf, err)
//	  }
func Errorf(t TestingT, err error, msg string, args ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.Errorf(t, err, msg, args...) {
		return
	}
	t.FailNow()
}

// Eventually asserts that given condition will be met in waitFor time,
// periodically checking target function each tick.
//
//	assert.Eventually(t, func() bool { return true; }, time.Second, 10*time.Millisecond)
func Eventually(t TestingT, condition func() bool, waitFor time.Duration, tick time.Duration, msgAndArgs ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.Eventually(t, condition, waitFor, tick, msgAndArgs...) {
		return
	}
	t.FailNow()
}

// Eventuallyf asserts that given condition will be met in waitFor time,
// periodically checking target function each tick.
//
//	assert.Eventuallyf(t, func() bool { return true; }, time.Second, 10*time.Millisecond, "error message %s", "formatted")
func Eventuallyf(t TestingT, condition func() bool, waitFor time.Duration, tick time.Duration, msg string, args ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.Eventuallyf(t, condition, waitFor, tick, msg, args...) {
		return
	}
	t.FailNow()
}

// Fail reports a failure through
func Fail(t TestingT, failureMessage string, msgAndArgs ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.Fail(t, failureMessage, msgAndArgs...) {
		return
	}
	t.FailNow()
}

// FailNow fails test
func FailNow(t TestingT, failureMessage string, msgAndArgs ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.FailNow(t, failureMessage, msgAndArgs...) {
		return
	}
	t.FailNow()
}

// FailNowf fails test
func FailNowf(t TestingT, failureMessage string, msg string, args ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.FailNowf(t, failureMessage, msg, args...) {
		return
	}
	t.FailNow()
}

// Failf reports a failure through
func Failf(t TestingT, failureMessage string, msg string, args ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.Failf(t, failureMessage, msg, args...) {
		return
	}
	t.FailNow()
}

// False asserts that the specified value is false.
//
//	assert.False(t, myBool)
func False(t TestingT, value bool, msgAndArgs ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.False(t, value, msgAndArgs...) {
		return
	}
	t.FailNow()
}

// Falsef asserts that the specified value is false.
//
//	assert.Falsef(t, myBool, "error message %s", "formatted")
func Falsef(t TestingT, value bool, msg string, args ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.Falsef(t, value, msg, args...) {
		return
	}
	t.FailNow()
}

// Greater asserts that the first element is greater than the second
//
//	assert.Greater(t, 2, 1)
//	assert.Greater(t, float64(2), float64(1))
//	assert.Greater(t, "b", "a")
func Greater(t TestingT, e1 interface{}, e2 interface{}, msgAndArgs ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.Greater(t, e1, e2, msgAndArgs...) {
		return
	}
	t.FailNow()
}

// GreaterOrEqual asserts that the first element is greater than or equal to the second
//
//	assert.GreaterOrEqual(t, 2, 1)
//	assert.GreaterOrEqual(t, 2, 2)
//	assert.GreaterOrEqual(t, "b", "a")
//	assert.GreaterOrEqual(t, "b", "b")
func GreaterOrEqual(t TestingT, e1 interface{}, e2 interface{}, msgAndArgs ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.GreaterOrEqual(t, e1, e2, msgAndArgs...) {
		return
	}
	t.FailNow()
}

// GreaterOrEqualf asserts that the first element is greater than or equal to the second
//
//	assert.GreaterOrEqualf(t, 2, 1, "error message %s", "formatted")
//	assert.GreaterOrEqualf(t, 2, 2, "error message %s", "formatted")
//	assert.GreaterOrEqualf(t, "b", "a", "error message %s", "formatted")
//	assert.GreaterOrEqualf(t, "b", "b", "error message %s", "formatted")
func GreaterOrEqualf(t TestingT, e1 interface{}, e2 interface{}, msg string, args ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.GreaterOrEqualf(t, e1, e2, msg, args...) {
		return
	}
	t.FailNow()
}

// Greaterf asserts that the first element is greater than the second
//
//	assert.Greaterf(t, 2, 1, "error message %s", "formatted")
//	assert.Greaterf(t, float64(2), float64(1), "error message %s", "formatted")
//	assert.Greaterf(t, "b", "a", "error message %s", "formatted")
func Greaterf(t TestingT, e1 interface{}, e2 interface{}, msg string, args ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.Greaterf(t, e1, e2, msg, args...) {
		return
	}
	t.FailNow()
}

// InDelta asserts that the two numerals are within delta of each other.
//
//	assert.InDelta(t, math.Pi, 22/7.0, 0.01)
func InDelta(t TestingT, expected interface{}, actual interface{}, delta float64, msgAndArgs ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.InDelta(t, expected, actual, delta, msgAndArgs...) {
		return
	}
	t.FailNow()
}

// InDeltaf asserts that the two numerals are within delta of each other.
//
//	assert.InDeltaf(t, math.Pi, 22/7.0, 0.01, "error message %s", "formatted")
func InDeltaf(t TestingT, expected interface{}, actual interface{}, delta float64, msg string, args ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.InDeltaf(t, expected, actual, delta, msg, args...) {
		return
	}
	t.FailNow()
}

// IsType asserts that the specified objects are of the same type.
func IsType(t TestingT, expectedType interface{}, object interface{}, msgAndArgs ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.IsType(t, expectedType, object, msgAndArgs...) {
		return
	}
	t.FailNow()
}

// IsTypef asserts that the specified objects are of the same type.
func IsTypef(t TestingT, expectedType interface{}, object interface{}, msg string, args ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.IsTypef(t, expectedType, object, msg, args...) {
		return
	}
	t.FailNow()
}

// Len asserts that the specified object has specific length.
// Len also fails if the object has a type that len() not accept.
//
//	assert.Len(t, mySlice, 3)
func Len(t TestingT, object interface{}, length int, msgAndArgs ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.Len(t, object, length, msgAndArgs...) {
		return
	}
	t.FailNow()
}

// Lenf asserts that the specified object has specific length.
// Lenf also fails if the object has a type that len() not accept.
//
//	assert.Lenf(t, mySlice, 3, "error message %s", "formatted")
func Lenf(t TestingT, object interface{}, length int, msg string, args ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.Lenf(t, object, length, msg, args...) {
		return
	}
	t.FailNow()
}

// Less asserts that the first element is less than the second
//
//	assert.Less(t, 1, 2)
//	assert.Less(t, float64(1), float64(2))
//	assert.Less(t, "a", "b")
func Less(t TestingT, e1 interface{}, e2 interface{}, msgAndArgs ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.Less(t, e1, e2, msgAndArgs...) {
		return
	}
	t.FailNow()
}

// LessOrEqual asserts that the first element is less than or equal to the second
//
//	assert.LessOrEqual(t, 1, 2)
//	assert.LessOrEqual(t, 2, 2)
//	assert.LessOrEqual(t, "a", "b")
//	assert.LessOrEqual(t, "b", "b")
func LessOrEqual(t TestingT, e1 interface{}, e2 interface{}, msgAndArgs ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.LessOrEqual(t, e1, e2, msgAndArgs...) {
		return
	}
	t.FailNow()
}

// LessOrEqualf asserts that the first element is less than or equal to the second
//
//	assert.LessOrEqualf(t, 1, 2, "error message %s", "formatted")
//	assert.LessOrEqualf(t, 2, 2, "error message %s", "formatted")
//	assert.LessOrEqualf(t, "a", "b", "error message %s", "formatted")
//	assert.LessOrEqualf(t, "b", "b", "error message %s", "formatted")
func LessOrEqualf(t TestingT, e1 interface{}, e2 interface{}, msg string, args ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.LessOrEqualf(t, e1, e2, msg, args...) {
		return
	}
	t.FailNow()
}

// Lessf asserts that the first element is less than the second
//
//	assert.Lessf(t, 1, 2, "error message %s", "formatted")
//	assert.Lessf(t, float64(1), float64(2), "error message %s", "formatted")
//	assert.Lessf(t, "a", "b", "error message %s", "formatted")
func Lessf(t TestingT, e1 interface{}, e2 interface{}, msg string, args ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.Lessf(t, e1, e2, msg, args...) {
		return
	}
	t.FailNow()
}

// Negative asserts that the specified element is negative
//
//	assert.Negative(t, -1)
//	assert.Negative(t, -1.23)
func Negative(t TestingT, e interface{}, msgAndArgs ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.Negative(t, e, msgAndArgs...) {
		return
	}
	t.FailNow()
}

// Negativef asserts that the specified element is negative
//
//	assert.Negativef(t, -1, "error message %s", "formatted")
//	assert.Negativef(t, -1.23, "error message %s", "formatted")
func Negativef(t TestingT, e interface{}, msg string, args ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.Negativef(t, e, msg, args...) {
		return
	}
	t.FailNow()
}

// Nil asserts that the specified object is nil.
//
//	assert.Nil(t, err)
func Nil(t TestingT, object interface{}, msgAndArgs ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.Nil(t, object, msgAndArgs...) {
		return
	}
	t.FailNow()
}

// Nilf asserts that the specified object is nil.
//
//	assert.Nilf(t, err, "error message %s", "formatted")
func Nilf(t TestingT, object interface{}, msg string, args ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.Nilf(t, object, msg, args...) {
		return
	}
	t.FailNow()
}

// NoError asserts that a function returned no error (i.e. `nil`).
//
//	  actualObj, err := SomeFunction()
//	  if assert.NoError(t, err) {
//		   assert.Equal(t, expectedObj, actualObj)
//	  }
func NoError(t TestingT, err error, msgAndArgs ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.NoError(t, err, msgAndArgs...) {
		return
	}
	t.FailNow()
}

// NoErrorf asserts that a function returned no error (i.e. `nil`).
//
//	  actualObj, err := SomeFunction()
//	  if assert.NoErrorf(t, err, "error message %s", "formatted") {
//		   assert.Equal(t, expectedObj, actualObj)
//	  }
func NoErrorf(t TestingT, err error, msg string, args ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.NoErrorf(t, err, msg, args...) {
		return
	}
	t.FailNow()
}

// NotContains asserts that the specified string, list(array, slice...) or map does NOT contain the
// specified substring or element.
//
//	assert.NotContains(t, "Hello World", "Earth")
//	assert.NotContains(t, ["Hello", "World"], "Earth")
//	assert.NotContains(t, {"Hello": "World"}, "Earth")
func NotContains(t TestingT, s interface{}, contains interface{}, msgAndArgs ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.NotContains(t, s, contains, msgAndArgs...) {
		return
	}
	t.FailNow()
}

// NotContainsf asserts that the specified string, list(array, slice...) or map does NOT contain the
// specified substring or element.
//
//	assert.NotContainsf(t, "Hello World", "Earth", "error message %s", "formatted")
//	assert.NotContainsf(t, ["Hello", "World"], "Earth", "error message %s", "formatted")
//	assert.NotContainsf(t, {"Hello": "World"}, "Earth", "error message %s", "formatted")
func NotContainsf(t TestingT, s interface{}, contains interface{}, msg string, args ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.NotContainsf(t, s, contains, msg, args...) {
		return
	}
	t.FailNow()
}

// NotEqual asserts that the specified values are NOT equal.
//
//	assert.NotEqual(t, obj1, obj2)
//
// Pointer variable equality is determined based on the equality of the
// referenced values (as opposed to the memory addresses).
func NotEqual(t TestingT, expected interface{}, actual interface{}, msgAndArgs ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.NotEqual(t, expected, actual, msgAndArgs...) {
		return
	}
	t.FailNow()
}

// NotEqualValues asserts that two objects are not equal even when converted to the same type
//
//	assert.NotEqualValues(t, obj1, obj2)
func NotEqualValues(t TestingT, expected interface{}, actual interface{}, msgAndArgs ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.NotEqualValues(t, expected, actual, msgAndArgs...) {
		return
	}
	t.FailNow()
}

// NotEqualValuesf asserts that two objects are not equal even when converted to the same type
//
//	assert.NotEqualValuesf(t, obj1, obj2, "error message %s", "formatted")
func NotEqualValuesf(t TestingT, expected interface{}, actual interface{}, msg string, args ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.NotEqualValuesf(t, expected, actual, msg, args...) {
		return
	}
	t.FailNow()
}

// NotEqualf asserts that the specified values are NOT equal.
//
//	assert.NotEqualf(t, obj1, obj2, "error message %s", "formatted")
//
// Pointer variable equality is determined based on the equality of the
// referenced values (as opposed to the memory addresses).
func NotEqualf(t TestingT, expected interface{}, actual interface{}, msg string, args ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.NotEqualf(t, expected, actual, msg, args...) {
		return
	}
	t.FailNow()
}

// NotNil asserts that the specified object is not nil.
//
//	assert.NotNil(t, err)
func NotNil(t TestingT, object interface{}, msgAndArgs ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.NotNil(t, object, msgAndArgs...) {
		return
	}
	t.FailNow()
}

// NotNilf asserts that the specified object is not nil.
//
//	assert.NotNilf(t, err, "error message %s", "formatted")
func NotNilf(t TestingT, object interface{}, msg string, args ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.NotNilf(t, object, msg, args...) {
		return
	}
	t.FailNow()
}

// Positive asserts that the specified element is positive
//
//	assert.Positive(t, 1)
//	assert.Positive(t, 1.23)
func Positive(t TestingT, e interface{}, msgAndArgs ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.Positive(t, e, msgAndArgs...) {
		return
	}
	t.FailNow()
}

// Positivef asserts that the specified element is positive
//
//	assert.Positivef(t, 1, "error message %s", "formatted")
//	assert.Positivef(t, 1.23, "error message %s", "formatted")
func Positivef(t TestingT, e interface{}, msg string, args ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.Positivef(t, e, msg, args...) {
		return
	}
	t.FailNow()
}

// True asserts that the specified value is true.
//
//	assert.True(t, myBool)
func True(t TestingT, value bool, msgAndArgs ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.True(t, value, msgAndArgs...) {
		return
	}
	t.FailNow()
}

// Truef asserts that the specified value is true.
//
//	assert.Truef(t, myBool, "error message %s", "formatted")
func Truef(t TestingT, value bool, msg string, args ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.Truef(t, value, msg, args...) {
		return
	}
	t.FailNow()
}

// WithinDuration asserts that the two times are within duration delta of each other.
//
//	assert.WithinDuration(t, time.Now(), time.Now(), 10*time.Second)
func WithinDuration(t TestingT, expected time.Time, actual time.Time, delta time.Duration, msgAndArgs ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.WithinDuration(t, expected, actual, delta, msgAndArgs...) {
		return
	}
	t.FailNow()
}

// WithinDurationf asserts that the two times are within duration delta of each other.
//
//	assert.WithinDurationf(t, time.Now(), time.Now(), 10*time.Second, "error message %s", "formatted")
func WithinDurationf(t TestingT, expected time.Time, actual time.Time, delta time.Duration, msg string, args ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if assert.WithinDurationf(t, expected, actual, delta, msg, args...) {
		return
	}
	t.FailNow()
}
