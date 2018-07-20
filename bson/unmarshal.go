package bson

// Unmarshal parses the BSON-encoded data and stores the result in the value
// pointed to by val. If val is nil or not a pointer, Unmarshal returns
// InvalidUnmarshalError.
func Unmarshalv2(data []byte, val interface{}) error { return nil }

// UnmarshalWithRegistry parses the BSON-encoded data using Registry r and
// stores the result in the value pointed to by val. If val is nil or not
// a pointer, UnmarshalWithRegistry returns InvalidUnmarshalError.
func UnmarshalWithRegistry(r *Registry, data []byte, val interface{}) error { return nil }

// UnmarshalDocument parses the *Document and stores the result in the value pointed to by val. If
// val is nil or not a pointer, UnmarshalDocument returns INvalidUnmarshalError.
func UnmarshalDocumentv2(d *Document, val interface{}) error { return nil }

// UnmarshalDocumentWithRegistry behaves the same as UnmarshalDocument but uses r as the *Registry.
func UnmarshalDocumentWithRegistry(r *Registry, d *Document, val interface{}) error { return nil }
