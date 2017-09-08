package options

// UpdateOption is for internal use.
type UpdateOption interface {
	UpdateName() string
	UpdateValue() interface{}
}

// arrayFilters

// UpdateName is for internal use.
func (opt *OptArrayFilters) UpdateName() string {
	return "arrayFilters"
}

// UpdateValue is for internal use.
func (opt *OptArrayFilters) UpdateValue() interface{} {
	return *opt
}

// bypassDocumentValidation

// UpdateName is for internal use.
func (opt *OptBypassDocumentValidation) UpdateName() string {
	return "bypassDocumentValidation"
}

// UpdateValue is for internal use.
func (opt *OptBypassDocumentValidation) UpdateValue() interface{} {
	return *opt
}

// collation

// UpdateName is for internal use.
func (opt *OptCollation) UpdateName() string {
	return "collation"
}

// UpdateValue is for internal use.
func (opt *OptCollation) UpdateValue() interface{} {
	return opt.Collation
}

// upsert

// UpdateName is for internal use.
func (opt *OptUpsert) UpdateName() string {
	return "upsert"
}

// UpdateValue is for internal use.
func (opt *OptUpsert) UpdateValue() interface{} {
	return *opt
}
