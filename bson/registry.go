package bson

import (
	"errors"
	"reflect"
	"sync"
	"sync/atomic"
)

// ErrNoCodec is returned when there is no codec available for a type or interface in the registry.
type ErrNoCodec struct {
	Type reflect.Type
}

func (enc ErrNoCodec) Error() string {
	return "no codec found for " + enc.Type.String()
}

// ErrNotInterface is returned when the provided type is not an interface.
var ErrNotInterface = errors.New("The provided typeis not an interface")

var defaultRegistry = NewRegistry()

// ErrFrozenRegistry is returned when an attempt to mutate a frozen Registry is
// made. A Registry is considered frozen when a call to Lookup has been made.
var ErrFrozenRegistry = errors.New("the Registry has been frozen and can no longer be modified")

// A Registry is used to store and retrieve codecs for types and interfaces. This type is the main
// typed passed around and Encoders and Decoders are constructed from it.
//
// TODO: Create a RegistryBuilder type and make the Registry type immutable.
type Registry struct {
	tr *typeRegistry
	ir *interfaceRegistry

	dm   Codec // default map codec
	ds   Codec // default struct codec
	dslc Codec // default slice & array codec

	frozen uint32 // once used the can no longer register new types
	l      sync.RWMutex
}

func NewRegistry() *Registry {
	// TODO: Register codecs.
	tr := &typeRegistry{reg: make(map[reflect.Type]Codec)}
	tr.register(reflect.TypeOf(false), new(BooleanCodec))

	return &Registry{
		tr:   tr,
		ir:   &interfaceRegistry{reg: make([]interfacePair, 0)},
		ds:   defaultStructCodec,
		dslc: defaultSliceCodec,
	}
}

// Register will register the provided Codec to the provided type. If the type is
// an interface, it will be registered in the interface registry. If the type is
// a pointer to or a type that is not an interface, it will be registered in the type
// registry.
func (r *Registry) Register(t reflect.Type, codec Codec) error {
	r.l.Lock()
	defer r.l.Unlock()
	if atomic.LoadUint32(&r.frozen) != 0 {
		return ErrFrozenRegistry
	}

	if t.Kind() == reflect.Interface {
		return r.ir.register(t, codec)
	}

	r.tr.register(t, codec)
	return nil
}

// SetDefaultMapCodec will set the Codec used when encoding or decoding a map that does
// not have another codec registered for it.
func (r *Registry) SetDefaultMapCodec(codec Codec) error {
	r.l.Lock()
	defer r.l.Unlock()
	if atomic.LoadUint32(&r.frozen) != 0 {
		return ErrFrozenRegistry
	}
	r.dm = codec
	return nil
}

// SetDefaultStructCodec will set the Codec used when encoding or decoding a struct that
// does not have another codec registered for it.
func (r *Registry) SetDefaultStructCodec(codec Codec) error {
	r.l.Lock()
	defer r.l.Unlock()
	if atomic.LoadUint32(&r.frozen) != 0 {
		return ErrFrozenRegistry
	}
	r.ds = codec
	return nil
}

// Lookup will inspect the type registry for either the type or a pointer to the type,
// if it doesn't find a codec it will inspect the interface registry for an interface
// that the type satisfies, if it doesn't find a codec there it will attempt to
// return either the default map codec or the default struct codec. If none of those
// apply, an error will be returned.
func (r *Registry) Lookup(t reflect.Type) (Codec, error) {
	r.l.RLock()
	defer r.l.RUnlock()

	// We do this always, it marks used as frozen, preventing mutation of this
	// Registry.
	atomic.CompareAndSwapUint32(&r.frozen, 0, 1)

	codec, err := r.tr.lookup(t)
	switch err {
	case nil:
		return codec, nil
	default:
		if _, ok := err.(ErrNoCodec); ok {
			break // continue
		}
		return nil, err
	}

	codec, err = r.ir.lookup(t)
	switch err {
	case nil:
		return codec, nil
	default:
		if _, ok := err.(ErrNoCodec); ok {
			break // continue
		}
		return nil, err
	}

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() == reflect.Struct {
		return r.ds, nil
	}

	if t.Kind() == reflect.Array || t.Kind() == reflect.Slice {
		return r.dslc, nil
	}

	if t.Kind() == reflect.Map && t.Key().Kind() == reflect.String {
		return r.dm, nil
	}

	return nil, ErrNoCodec{Type: t}
}

// The type registry handles codecs that are for specifics types that are not interfaces.
// This registry will handle both the types themselves and pointers to those types.
type typeRegistry struct {
	reg map[reflect.Type]Codec

	sync.RWMutex
}

// lookup handles finding a codec for the registered type. Will return an error if no codec
// could be found.
func (tr *typeRegistry) lookup(t reflect.Type) (Codec, error) {
	if t.Kind() != reflect.Ptr {
		t = reflect.PtrTo(t)
	}

	tr.RLock()
	defer tr.RUnlock()
	codec, exists := tr.reg[t]
	if !exists {
		return nil, ErrNoCodec{Type: t}
	}
	return codec, nil
}

// register adds a new codec to this registry for the given type. It handles registering the
// codec for both the type and pointer to the type.
func (tr *typeRegistry) register(t reflect.Type, codec Codec) {
	if t.Kind() != reflect.Ptr {
		t = reflect.PtrTo(t)
	}

	tr.Lock()
	tr.reg[t] = codec
	tr.Unlock()
	return
}

type interfacePair struct {
	i reflect.Type
	c Codec
}

// The interface registry handles codecs that are for interface types.
type interfaceRegistry struct {
	reg []interfacePair

	sync.RWMutex
}

// lookup handles finding a codec for the registered interface. Will return an error if no codec
// could be found.
func (ir *interfaceRegistry) lookup(t reflect.Type) (Codec, error) {
	ir.RLock()
	defer ir.RUnlock()
	for _, ip := range ir.reg {
		if !t.Implements(ip.i) {
			continue
		}

		return ip.c, nil
	}
	return nil, ErrNoCodec{Type: t}
}

// register adds a new codec to this registry for the given interface.
func (ir *interfaceRegistry) register(t reflect.Type, codec Codec) error {
	if t.Kind() != reflect.Interface {
		return ErrNotInterface
	}

	ir.Lock()
	defer ir.Unlock()
	for _, ip := range ir.reg {
		if ip.i == t {
			ip.c = codec
			return nil
		}
	}

	ir.reg = append(ir.reg, interfacePair{i: t, c: codec})
	return nil
}
