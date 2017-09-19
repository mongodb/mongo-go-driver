package options

// AggregateOption is for internal use.
type AggregateOption interface {
	AggregateName() string
	AggregateValue() interface{}
}

// allowDiskUse

// AggregateName is for internal use.
func (opt *OptAllowDiskUse) AggregateName() string {
	return "allowDiskUse"
}

// AggregateValue is for internal use.
func (opt *OptAllowDiskUse) AggregateValue() interface{} {
	return *opt
}

// batchSize

// AggregateName is for internal use.
func (opt *OptBatchSize) AggregateName() string {
	return "batchSize"
}

// AggregateValue is for internal use.
func (opt *OptBatchSize) AggregateValue() interface{} {
	return *opt
}

// bypassDocumentValidation

// AggregateName is for internal use.
func (opt *OptBypassDocumentValidation) AggregateName() string {
	return "bypassDocumentValidation"
}

// AggregateValue is for internal use.
func (opt *OptBypassDocumentValidation) AggregateValue() interface{} {
	return *opt
}

// collation

// AggregateName is for internal use.
func (opt *OptCollation) AggregateName() string {
	return "collation"
}

// AggregateValue is for internal use.
func (opt *OptCollation) AggregateValue() interface{} {
	return opt.Collation
}

// comment

// AggregateName is for internal use.
func (opt *OptComment) AggregateName() string {
	return "comment"
}

// AggregateValue is for internal use.
func (opt *OptComment) AggregateValue() interface{} {
	return *opt
}

// maxTimeMS

// AggregateName is for internal use.
func (opt *OptMaxTime) AggregateName() string {
	return "maxTimeMS"
}

// AggregateValue is for internal use.
func (opt *OptMaxTime) AggregateValue() interface{} {
	return *opt
}
