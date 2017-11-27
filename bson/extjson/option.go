package extjson

// OptionalValue is a wrapper around Value which has special meaning when the
// value is not present.
type OptionalValue struct {
	v    Value
	none bool
}

// OptionalValueNone is the None value.
var OptionalValueNone = OptionalValue{none: true}

// IsNone returns whether the value is not present.
func (o OptionalValue) IsNone() bool {
	return o.none
}

// IsSome returns whether the value is present.
func (o OptionalValue) IsSome() bool {
	return !o.none
}

// Get returns the value and whether it was present.
func (o OptionalValue) Get() (Value, bool) {
	return o.v, !o.none
}

// Unwrap returns the value, and panics when called on None.
func (o OptionalValue) Unwrap() Value {
	if o.none {
		panic("unwrap on OptionalValue None")
	}
	return o.v
}

// GetDocItem tries to fetch a value by key from a supposed document.
func (o OptionalValue) GetDocItem(key string) OptionalValue {
	if o.none {
		return OptionalValue{none: true}
	}
	return o.v.GetDocItem(key)
}
