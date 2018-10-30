package bsoncodec

// Proxy is an interface implemented by types that cannot themselves be directly encoded. Types
// that implement this interface with have ProxyBSON called during the encoding process and that
// value will be encoded in place for the implementer.
type Proxy interface {
	ProxyBSON() (interface{}, error)
}
