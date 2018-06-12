package bson

// Type represents a BSON type.
type Type byte

// These constants uniquely refer to each BSON type.
const (
	TypeDouble        Type = 0x01
	TypeString        Type = 0x02
	TypeDocument      Type = 0x03
	TypeArray         Type = 0x04
	TypeBinary        Type = 0x05
	TypeUndefined     Type = 0x06
	TypeObjectID      Type = 0x07
	TypeBoolean       Type = 0x08
	TypeDateTime      Type = 0x09
	TypeNull          Type = 0x0A
	TypeRegex         Type = 0x0B
	TypeDBPointer     Type = 0x0C
	TypeJavaScript    Type = 0x0D
	TypeSymbol        Type = 0x0E
	TypeCodeWithScope Type = 0x0F
	TypeInt32         Type = 0x10
	TypeTimestamp     Type = 0x11
	TypeInt64         Type = 0x12
	TypeDecimal128    Type = 0x13
	TypeMinKey        Type = 0xFF
	TypeMaxKey        Type = 0x7F
)
