package bson

// Flex represents a flexible BSON document containing ordered elements. It can be used to create
// concise BSON representations.
//
// 		bson.Flex{{"hello", "world"}, {"foo", true}}
//
// This type should generally be used for the encoding path, such as providing a document to be
// inserted into MongoDB. On the decode path, the FlexElement Value property will be filled with the
// default primitive types.
type Flex []FlexElement

// FlexElement is an element of a Flex BSON document.
type FlexElement struct {
	Name  string
	Value interface{}
}

// FlexArray represents a BSON array. It serves as the primitive type for arrays when decoding into
// a FlexElement.
type FlexArray []interface{}

// FlexCodeWithScope represents a BSON code with scope element. It serves as the primitive type for
// code with scope when decoding into a FlexElement.
type FlexCodeWithScope struct {
	Code  string
	Scope Flex
}
