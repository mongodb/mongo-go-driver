package xjson

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"

	"github.com/10gen/mongo-go-driver/bson"
)

// Errors used during encryption/decryption
var (
	ErrNotEncryptedValue      = errors.New("not an encrypted value")
	ErrEncryptedValueBadValue = errors.New("expected encrypted value to be an encoded BSON string")
)

// A Crypter is responsible for traversing whatever value is passed in
// and checking if it can be encrypted/decrypted. If a type is passed in that
// implements the respective decrypt/encrypt methods, that will be used.
// This is useful for structs where only certain information should be
// encrypted/decrypted.
// A Crypter must return errors when a type cannot be explcitly crypted.
type Crypter interface {
	Encrypt(unencrypted interface{}) (interface{}, error)
	EncryptValue(unencrypted interface{}) (string, error)
	EncryptValueAsWrapped(unencrypted interface{}) (EncryptedValue, error)

	Decrypt(encrypted interface{}) (interface{}, error)
	DecryptValue(in string, out interface{}) error
	DecryptParsedValue(in []byte, out interface{}) error
}

// A Decrypter defines a way to decrypt itself and always returns
// a copy of the value
type Decrypter interface {
	Decrypt(crypter Crypter) (interface{}, error)
}

// An Encrypter defines a way to encrypt itself and always returns
// a copy of the value
type Encrypter interface {
	Encrypt(crypter Crypter) (interface{}, error)
}

// NewCrypter creates a new crypter from a given encryption key path
// of an AES key
func NewCrypter(encryptionKeyPath string) (Crypter, error) {
	key, err := ioutil.ReadFile(encryptionKeyPath)
	if err != nil {
		return nil, err
	}
	blockCipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return &SimpleCrypter{blockCipher}, nil
}

// NewTestCrypter is for testing only
func NewTestCrypter() Crypter {
	blockCipher, err := aes.NewCipher([]byte("greatlysecretkey"))
	if err != nil {
		panic(err)
	}
	return &SimpleCrypter{blockCipher}
}

// SimpleCrypter consists of a cipher.Block and crypts values
// in the convention described in its methods
type SimpleCrypter struct {
	blockCipher cipher.Block
}

// CryptMode defines the mode in which a crypter is operating
type CryptMode int

// The set of available crypt modes
const (
	CryptModeEncrypt = iota
	CryptModeDecrypt
)

// Decrypt takes an encrypted value and attempts to decrypt it
func (c *SimpleCrypter) Decrypt(encrypted interface{}) (interface{}, error) {
	return c.crypt(encrypted, CryptModeDecrypt)
}

// Encrypt takes an unencrypted value and attempts to encrypt it
func (c *SimpleCrypter) Encrypt(unencrypted interface{}) (interface{}, error) {
	return c.crypt(unencrypted, CryptModeEncrypt)
}

// EncryptValue is responsible for actual encryption of values. Any value
// passed in here is marshaled down to BSON inside of a wrapped document.
// That wrapped document is then encrypted and placed in a sentinel BSON structure
// as bytes. That structure is encoded to Base64 and used as a string.
func (c *SimpleCrypter) EncryptValue(value interface{}) (string, error) {
	// Marshal the value as wrapped since bson package wont let us marshal
	// individual values
	md, err := bson.Marshal(struct {
		Value interface{} `bson:"value"`
	}{value})
	if err != nil {
		return "", err
	}

	// Encrypt the encoded BSON bytes of the wrapped value
	encrypted, err := c.cryptRawValue(md, CryptModeEncrypt)
	if err != nil {
		return "", err
	}

	// Encode to sentinel BSON value
	md, err = bson.Marshal(struct {
		Value     interface{} `bson:"value"`
		Encrypted bool        `bson:"encrypted_value"`
	}{encrypted, true})
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(md), nil
}

// EncryptValueAsWrapped is similar to EncryptValue but wraps the value in a structure that allows
// it to be decrypted/marshaled later.
func (c *SimpleCrypter) EncryptValueAsWrapped(value interface{}) (EncryptedValue, error) {

	encrypted, err := c.EncryptValue(value)
	if err != nil {
		return EncryptedValue{}, err
	}

	return EncryptedValue{value: encrypted}, nil
}

// DecryptValue is responsible for decrypting actual encrypted values.
// It first decodes the encoded Base64 string for the BSON bytes
// of the sentinel structure. That structure is then validated. The
// encrypted bytes are then decrypted and unmarshaled back into a wrapper
// structure. The true value is then unmarshaled onto `out`.
func (c *SimpleCrypter) DecryptValue(in string, out interface{}) error {
	if in == "" {
		return ErrNotEncryptedValue
	}

	// Get unwrapped value
	encrypted, err := ParseEncryptedValueFromEncoded(in)
	if err != nil {
		return err
	}

	return c.DecryptParsedValue(encrypted, out)
}

// DecryptParsedValue is like DecryptValue but assumes the encryped bytes have
// already been pulled out from the parser.
func (c *SimpleCrypter) DecryptParsedValue(encryptedIn []byte, out interface{}) error {
	if len(encryptedIn) == 0 {
		return ErrNotEncryptedValue
	}

	// Decrypt the wrapped value
	decrypted, err := c.cryptRawValue(encryptedIn, CryptModeDecrypt)
	if err != nil {
		return err
	}

	// Pull out the wrapped value
	var wrapped struct {
		Value bson.Raw `bson:"value"`
	}
	if err := bson.Unmarshal(decrypted, &wrapped); err != nil {
		return ErrEncryptedValueBadValue
	}

	// Use the actual wrapped value to unmarshal onto the desired type
	if err := wrapped.Value.Unmarshal(out); err != nil {
		return err
	}

	return nil
}

func (c *SimpleCrypter) crypt(value interface{}, mode CryptMode) (interface{}, error) {
	if value == nil {
		if mode == CryptModeDecrypt {
			return value, nil
		}
		return c.cryptBSON(nil, mode)
	}

	// Check special types first since they may satisfy
	// kinds below.
	switch mode {
	case CryptModeDecrypt:
		dec, ok := value.(Decrypter)
		if !ok {
			break
		}
		decrypted, err := dec.Decrypt(c)
		if err != nil {
			return nil, err
		}
		return decrypted, nil
	case CryptModeEncrypt:
		enc, ok := value.(Encrypter)
		if !ok {
			break
		}
		encrypted, err := enc.Encrypt(c)
		if err != nil {
			return nil, err
		}
		return encrypted, nil
	default:
		return nil, errors.New("unknown crypt mode")
	}

	switch x := value.(type) {
	case bson.ObjectId, bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return c.cryptBSON(value, mode)
	case string:
		return c.cryptString(x, mode)
	case bson.D:
		return c.cryptBSOND(x, mode)
	}

	switch reflect.ValueOf(value).Kind() {
	case reflect.Slice:
		return c.cryptSlice(value, mode)
	case reflect.Map:
		return c.cryptMap(value, mode)
	}

	if mode == CryptModeEncrypt {
		return nil, fmt.Errorf("do not know how to encrypt a %T %#v", value, value)
	}

	// Last effort
	return c.cryptBSON(value, mode)
}

// Crypting BSON generically tries to crypt the given value
func (c *SimpleCrypter) cryptBSON(value interface{}, mode CryptMode) (interface{}, error) {
	switch mode {
	case CryptModeEncrypt:
		return c.EncryptValue(value)
	case CryptModeDecrypt:
		return nil, fmt.Errorf("do not know how to decrypt a %q %#v", value, value)
	}

	panic("unknown crypt mode")
}

// Crypting a string is special in that our output format is also a string. This provides
// a nice way to always call Encrypt/Decrypt on strings and get some generic value back.
func (c *SimpleCrypter) cryptString(str string, mode CryptMode) (interface{}, error) {
	switch mode {
	case CryptModeEncrypt:
		return c.EncryptValue(str)
	case CryptModeDecrypt:
		// This may not be a string underneath, so let the type be derived
		var iface interface{}
		if err := c.DecryptValue(str, &iface); err != nil {
			return nil, err
		}
		return iface, nil
	}

	panic("unknown crypt mode")
}

// Maps are treated specially in that we do not encrypt the whole map, but its
// values recursively.
func (c *SimpleCrypter) cryptMap(m interface{}, mode CryptMode) (interface{}, error) {
	value := reflect.ValueOf(m)
	out := reflect.MakeMap(value.Type())
	for _, key := range value.MapKeys() {
		v := value.MapIndex(key)
		crypted, err := c.crypt(v.Interface(), mode)
		if err != nil {
			return nil, err
		}
		cryptedV := reflect.ValueOf(crypted)
		if crypted == nil {
			switch out.Type().Elem().Kind() {
			case reflect.Interface, reflect.Ptr, reflect.Map, reflect.Slice:
			default:
				return nil, fmt.Errorf("cannot assign a nil value to a %q", out.Type().Elem().Kind().String())
			}
			cryptedV = reflect.Zero(v.Type())
		}
		if !cryptedV.Type().AssignableTo(out.Type().Elem()) {
			return nil, fmt.Errorf("cannot assign a %T to a %q", crypted, out.Type().Elem().String())
		}

		out.SetMapIndex(key, cryptedV)
	}
	return out.Interface(), nil
}

// Slices are treated specially in that we do not encrypt the whole slice, but
// its values recursively.
func (c *SimpleCrypter) cryptSlice(arr interface{}, mode CryptMode) (interface{}, error) {
	value := reflect.ValueOf(arr)
	size := value.Len()
	out := reflect.MakeSlice(value.Type(), size, size)
	for i := 0; i < size; i++ {
		v := value.Index(i)
		crypted, err := c.crypt(v.Interface(), mode)
		if err != nil {
			return nil, err
		}
		var cryptedV reflect.Value
		if crypted == nil {
			cryptedV = reflect.Zero(v.Type())
		} else {
			cryptedV = reflect.ValueOf(crypted)
		}
		out.Index(i).Set(cryptedV)
	}
	return out.Interface(), nil
}

// bson.Ds are treated specially in that we do not encrypt the whole objects, but its
// values recursively.
func (c *SimpleCrypter) cryptBSOND(d bson.D, mode CryptMode) (interface{}, error) {
	newD := make(bson.D, 0, len(d))
	for _, elem := range d {
		crypted, err := c.crypt(elem.Value, mode)
		if err != nil {
			return nil, err
		}
		newD = append(newD, bson.DocElem{elem.Name, crypted})
	}
	return newD, nil
}

// Crypting raw values is responsible for taking actual bytes and doing the
// real crypto on them. This MUST not be used outside of this package and should only
// be called by EncryptValue/DecryptValue since those methods define the encrypted
// storage format.
func (c *SimpleCrypter) cryptRawValue(value []byte, mode CryptMode) ([]byte, error) {
	switch mode {
	case CryptModeEncrypt:
		plaintext := value
		paddingNeeded := aes.BlockSize - ((len(plaintext) + 1) % aes.BlockSize)

		if paddingNeeded > 255 {
			return nil, errors.New("cannot pad properly")
		}

		paddedText := make([]byte, 1+len(plaintext)+paddingNeeded)
		paddedText[0] = uint8(paddingNeeded)
		copy(paddedText[1:], plaintext)
		for i := 0; i < paddingNeeded; i++ {
			paddedText[1+len(plaintext)+i] = 0
		}

		cipherText := make([]byte, aes.BlockSize+len(paddedText))
		iv := cipherText[:aes.BlockSize]
		if _, err := io.ReadFull(rand.Reader, iv); err != nil {
			return nil, err
		}

		mode := cipher.NewCBCEncrypter(c.blockCipher, iv)
		mode.CryptBlocks(cipherText[aes.BlockSize:], paddedText)

		return cipherText, nil
	case CryptModeDecrypt:
		cipherText := make([]byte, len(value))
		copy(cipherText, value)
		if len(cipherText) < aes.BlockSize {
			return nil, errors.New("cipher text shorter than block size")
		}
		iv := cipherText[:aes.BlockSize]
		cipherText = cipherText[aes.BlockSize:]

		if len(cipherText)%aes.BlockSize != 0 {
			return nil, errors.New("cipher text is not a multiple of the block size")
		}

		mode := cipher.NewCBCDecrypter(c.blockCipher, iv)
		mode.CryptBlocks(cipherText, cipherText)

		padding := cipherText[0]
		if int(padding) >= len(cipherText) {
			return nil, errors.New("bad padding")
		}

		plainText := cipherText[1 : len(cipherText)-int(padding)]

		return plainText, nil
	}

	panic("unknown crypt mode")
}

// An EncryptedValue is a simple wrapper that can capture the
// encrypted storage format and decrypt the values at a later stage.
type EncryptedValue struct {
	value    string
	rawValue []byte
}

// DecryptValue attempts toe decrypt the underlying value, whether it be
// raw or parsed
func (ev EncryptedValue) DecryptValue(crypter Crypter, out interface{}) error {
	if ev.value != "" {
		return crypter.DecryptValue(ev.value, out)
	}
	return crypter.DecryptParsedValue(ev.rawValue, out)
}

// GetBSON marshals a CryptedValue to a string encoding
// a BSON document of the value and a sentinel
func (ev EncryptedValue) GetBSON() (interface{}, error) {
	if ev.value == "" {
		return nil, ErrNotEncryptedValue
	}
	return ev.value, nil
}

// SetBSON attempts to parse the BSON for an encrypted value
// but does not decrypt it
func (ev *EncryptedValue) SetBSON(raw bson.Raw) error {
	temp, err := NewEncryptedValueFromBSON(raw)
	if err != nil {
		return err
	}
	*ev = temp
	return nil
}

// NewEncryptedValueFromBSON attempts to pull out the expected BSON
// string from the raw BSON and parse the string into an encrypted
// value
func NewEncryptedValueFromBSON(raw bson.Raw) (EncryptedValue, error) {
	// Should be a string
	var bsonStr string
	if err := raw.Unmarshal(&bsonStr); err != nil {
		return EncryptedValue{}, ErrNotEncryptedValue
	}

	rawValue, err := ParseEncryptedValueFromEncoded(bsonStr)
	if err != nil {
		return EncryptedValue{}, err
	}

	return EncryptedValue{rawValue: rawValue}, nil
}

const (
	// EncryptedValueValueField is the field at which the actual
	// encrypted value is stored as encrypted BSON bytes
	EncryptedValueValueField = "value"

	// EncryptedValueSentinelField is a sentinel that indicates to us
	// that this BSON object is indeed an encrypted value
	EncryptedValueSentinelField = "encrypted_value"
)

// ParseEncryptedValueFromEncoded parses the storage encrypted format
// into the encrypted bytes for processing.
func ParseEncryptedValueFromEncoded(encoded string) ([]byte, error) {
	// Decode to BSON
	md, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, ErrNotEncryptedValue
	}

	// Ensure this is actually an encrypted value
	var rawM bson.M
	if err := bson.Unmarshal(md, &rawM); err != nil {
		return nil, ErrNotEncryptedValue
	}

	if len(rawM) != 2 {
		return nil, ErrNotEncryptedValue
	}

	if flag, ok := rawM[EncryptedValueSentinelField].(bool); !ok || !flag {
		return nil, ErrNotEncryptedValue
	}

	value, ok := rawM[EncryptedValueValueField].([]byte)
	if !ok || len(value) == 0 {
		return nil, ErrNotEncryptedValue
	}

	return value, nil
}
