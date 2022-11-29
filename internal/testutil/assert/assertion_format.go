// Copied from https://github.com/stretchr/testify/blob/1333b5d3bda8cf5aedcf3e1aaa95cac28aaab892/assert/assertion_format.go

// Copyright 2020 Mat Ryer, Tyler Bunnell and all contributors. All rights reserved.
// Use of this source code is governed by an MIT-style license that can be found in
// the THIRD-PARTY-NOTICES file.

package assert

import (
	time "time"
)

// Containsf asserts that the specified string, list(array, slice...) or map contains the
// specified substring or element.
//
//	assert.Containsf(t, "Hello World", "World", "error message %s", "formatted")
//	assert.Containsf(t, ["Hello", "World"], "World", "error message %s", "formatted")
//	assert.Containsf(t, {"Hello": "World"}, "Hello", "error message %s", "formatted")
func Containsf(t TestingT, s interface{}, contains interface{}, msg string, args ...interface{}) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	return Contains(t, s, contains, append([]interface{}{msg}, args...)...)
}

// ElementsMatchf asserts that the specified listA(array, slice...) is equal to specified
// listB(array, slice...) ignoring the order of the elements. If there are duplicate elements,
// the number of appearances of each of them in both lists should match.
//
// assert.ElementsMatchf(t, [1, 3, 2, 3], [1, 3, 3, 2], "error message %s", "formatted")
func ElementsMatchf(t TestingT, listA interface{}, listB interface{}, msg string, args ...interface{}) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	return ElementsMatch(t, listA, listB, append([]interface{}{msg}, args...)...)
}

// Equalf asserts that two objects are equal.
//
//	assert.Equalf(t, 123, 123, "error message %s", "formatted")
//
// Pointer variable equality is determined based on the equality of the
// referenced values (as opposed to the memory addresses). Function equality
// cannot be determined and will always fail.
func Equalf(t TestingT, expected interface{}, actual interface{}, msg string, args ...interface{}) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	return Equal(t, expected, actual, append([]interface{}{msg}, args...)...)
}

// EqualErrorf asserts that a function returned an error (i.e. not `nil`)
// and that it is equal to the provided error.
//
//	actualObj, err := SomeFunction()
//	assert.EqualErrorf(t, err,  expectedErrorString, "error message %s", "formatted")
func EqualErrorf(t TestingT, theError error, errString string, msg string, args ...interface{}) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	return EqualError(t, theError, errString, append([]interface{}{msg}, args...)...)
}

// EqualValuesf asserts that two objects are equal or convertable to the same types
// and equal.
//
//	assert.EqualValuesf(t, uint32(123), int32(123), "error message %s", "formatted")
func EqualValuesf(t TestingT, expected interface{}, actual interface{}, msg string, args ...interface{}) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	return EqualValues(t, expected, actual, append([]interface{}{msg}, args...)...)
}

// Errorf asserts that a function returned an error (i.e. not `nil`).
//
//	  actualObj, err := SomeFunction()
//	  if assert.Errorf(t, err, "error message %s", "formatted") {
//		   assert.Equal(t, expectedErrorf, err)
//	  }
func Errorf(t TestingT, err error, msg string, args ...interface{}) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	return Error(t, err, append([]interface{}{msg}, args...)...)
}

// ErrorContainsf asserts that a function returned an error (i.e. not `nil`)
// and that the error contains the specified substring.
//
//	actualObj, err := SomeFunction()
//	assert.ErrorContainsf(t, err,  expectedErrorSubString, "error message %s", "formatted")
func ErrorContainsf(t TestingT, theError error, contains string, msg string, args ...interface{}) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	return ErrorContains(t, theError, contains, append([]interface{}{msg}, args...)...)
}

// Eventuallyf asserts that given condition will be met in waitFor time,
// periodically checking target function each tick.
//
//	assert.Eventuallyf(t, func() bool { return true; }, time.Second, 10*time.Millisecond, "error message %s", "formatted")
func Eventuallyf(t TestingT, condition func() bool, waitFor time.Duration, tick time.Duration, msg string, args ...interface{}) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	return Eventually(t, condition, waitFor, tick, append([]interface{}{msg}, args...)...)
}

// Failf reports a failure through
func Failf(t TestingT, failureMessage string, msg string, args ...interface{}) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	return Fail(t, failureMessage, append([]interface{}{msg}, args...)...)
}

// FailNowf fails test
func FailNowf(t TestingT, failureMessage string, msg string, args ...interface{}) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	return FailNow(t, failureMessage, append([]interface{}{msg}, args...)...)
}

// Falsef asserts that the specified value is false.
//
//	assert.Falsef(t, myBool, "error message %s", "formatted")
func Falsef(t TestingT, value bool, msg string, args ...interface{}) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	return False(t, value, append([]interface{}{msg}, args...)...)
}

// Greaterf asserts that the first element is greater than the second
//
//	assert.Greaterf(t, 2, 1, "error message %s", "formatted")
//	assert.Greaterf(t, float64(2), float64(1), "error message %s", "formatted")
//	assert.Greaterf(t, "b", "a", "error message %s", "formatted")
func Greaterf(t TestingT, e1 interface{}, e2 interface{}, msg string, args ...interface{}) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	return Greater(t, e1, e2, append([]interface{}{msg}, args...)...)
}

// GreaterOrEqualf asserts that the first element is greater than or equal to the second
//
//	assert.GreaterOrEqualf(t, 2, 1, "error message %s", "formatted")
//	assert.GreaterOrEqualf(t, 2, 2, "error message %s", "formatted")
//	assert.GreaterOrEqualf(t, "b", "a", "error message %s", "formatted")
//	assert.GreaterOrEqualf(t, "b", "b", "error message %s", "formatted")
func GreaterOrEqualf(t TestingT, e1 interface{}, e2 interface{}, msg string, args ...interface{}) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	return GreaterOrEqual(t, e1, e2, append([]interface{}{msg}, args...)...)
}

// InDeltaf asserts that the two numerals are within delta of each other.
//
//	assert.InDeltaf(t, math.Pi, 22/7.0, 0.01, "error message %s", "formatted")
func InDeltaf(t TestingT, expected interface{}, actual interface{}, delta float64, msg string, args ...interface{}) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	return InDelta(t, expected, actual, delta, append([]interface{}{msg}, args...)...)
}

// IsTypef asserts that the specified objects are of the same type.
func IsTypef(t TestingT, expectedType interface{}, object interface{}, msg string, args ...interface{}) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	return IsType(t, expectedType, object, append([]interface{}{msg}, args...)...)
}

// Lenf asserts that the specified object has specific length.
// Lenf also fails if the object has a type that len() not accept.
//
//	assert.Lenf(t, mySlice, 3, "error message %s", "formatted")
func Lenf(t TestingT, object interface{}, length int, msg string, args ...interface{}) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	return Len(t, object, length, append([]interface{}{msg}, args...)...)
}

// Lessf asserts that the first element is less than the second
//
//	assert.Lessf(t, 1, 2, "error message %s", "formatted")
//	assert.Lessf(t, float64(1), float64(2), "error message %s", "formatted")
//	assert.Lessf(t, "a", "b", "error message %s", "formatted")
func Lessf(t TestingT, e1 interface{}, e2 interface{}, msg string, args ...interface{}) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	return Less(t, e1, e2, append([]interface{}{msg}, args...)...)
}

// LessOrEqualf asserts that the first element is less than or equal to the second
//
//	assert.LessOrEqualf(t, 1, 2, "error message %s", "formatted")
//	assert.LessOrEqualf(t, 2, 2, "error message %s", "formatted")
//	assert.LessOrEqualf(t, "a", "b", "error message %s", "formatted")
//	assert.LessOrEqualf(t, "b", "b", "error message %s", "formatted")
func LessOrEqualf(t TestingT, e1 interface{}, e2 interface{}, msg string, args ...interface{}) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	return LessOrEqual(t, e1, e2, append([]interface{}{msg}, args...)...)
}

// Negativef asserts that the specified element is negative
//
//	assert.Negativef(t, -1, "error message %s", "formatted")
//	assert.Negativef(t, -1.23, "error message %s", "formatted")
func Negativef(t TestingT, e interface{}, msg string, args ...interface{}) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	return Negative(t, e, append([]interface{}{msg}, args...)...)
}

// Nilf asserts that the specified object is nil.
//
//	assert.Nilf(t, err, "error message %s", "formatted")
func Nilf(t TestingT, object interface{}, msg string, args ...interface{}) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	return Nil(t, object, append([]interface{}{msg}, args...)...)
}

// NoErrorf asserts that a function returned no error (i.e. `nil`).
//
//	  actualObj, err := SomeFunction()
//	  if assert.NoErrorf(t, err, "error message %s", "formatted") {
//		   assert.Equal(t, expectedObj, actualObj)
//	  }
func NoErrorf(t TestingT, err error, msg string, args ...interface{}) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	return NoError(t, err, append([]interface{}{msg}, args...)...)
}

// NotContainsf asserts that the specified string, list(array, slice...) or map does NOT contain the
// specified substring or element.
//
//	assert.NotContainsf(t, "Hello World", "Earth", "error message %s", "formatted")
//	assert.NotContainsf(t, ["Hello", "World"], "Earth", "error message %s", "formatted")
//	assert.NotContainsf(t, {"Hello": "World"}, "Earth", "error message %s", "formatted")
func NotContainsf(t TestingT, s interface{}, contains interface{}, msg string, args ...interface{}) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	return NotContains(t, s, contains, append([]interface{}{msg}, args...)...)
}

// NotEqualf asserts that the specified values are NOT equal.
//
//	assert.NotEqualf(t, obj1, obj2, "error message %s", "formatted")
//
// Pointer variable equality is determined based on the equality of the
// referenced values (as opposed to the memory addresses).
func NotEqualf(t TestingT, expected interface{}, actual interface{}, msg string, args ...interface{}) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	return NotEqual(t, expected, actual, append([]interface{}{msg}, args...)...)
}

// NotEqualValuesf asserts that two objects are not equal even when converted to the same type
//
//	assert.NotEqualValuesf(t, obj1, obj2, "error message %s", "formatted")
func NotEqualValuesf(t TestingT, expected interface{}, actual interface{}, msg string, args ...interface{}) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	return NotEqualValues(t, expected, actual, append([]interface{}{msg}, args...)...)
}

// NotNilf asserts that the specified object is not nil.
//
//	assert.NotNilf(t, err, "error message %s", "formatted")
func NotNilf(t TestingT, object interface{}, msg string, args ...interface{}) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	return NotNil(t, object, append([]interface{}{msg}, args...)...)
}

// Positivef asserts that the specified element is positive
//
//	assert.Positivef(t, 1, "error message %s", "formatted")
//	assert.Positivef(t, 1.23, "error message %s", "formatted")
func Positivef(t TestingT, e interface{}, msg string, args ...interface{}) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	return Positive(t, e, append([]interface{}{msg}, args...)...)
}

// Truef asserts that the specified value is true.
//
//	assert.Truef(t, myBool, "error message %s", "formatted")
func Truef(t TestingT, value bool, msg string, args ...interface{}) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	return True(t, value, append([]interface{}{msg}, args...)...)
}

// WithinDurationf asserts that the two times are within duration delta of each other.
//
//	assert.WithinDurationf(t, time.Now(), time.Now(), 10*time.Second, "error message %s", "formatted")
func WithinDurationf(t TestingT, expected time.Time, actual time.Time, delta time.Duration, msg string, args ...interface{}) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	return WithinDuration(t, expected, actual, delta, append([]interface{}{msg}, args...)...)
}
