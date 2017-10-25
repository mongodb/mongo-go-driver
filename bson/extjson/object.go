package extjson

import "github.com/10gen/mongo-go-driver/bson"


// MapAsBsonD returns a shallow copy of the given document.
func MapAsBsonD(m map[string]interface{}) bson.D {
	doc := make(bson.D, 0, len(m))
	for k, v := range m {
		doc = append(doc, bson.DocElem{k, v})
	}
	return doc
}

// DeepMapAsBsonD converts the given map into a bson.D. It assumes an
// encountered bson.D does not contain maps within it.
func DeepMapAsBsonD(m map[string]interface{}) bson.D {
	return deepMapAsBsonD(m).(bson.D)
}

func deepMapAsBsonD(in interface{}) interface{} {
	switch x := in.(type) {
	case map[string]interface{}:
		doc := make(bson.D, 0, len(x))
		for k, v := range x {
			v = deepMapAsBsonD(v)
			doc = append(doc, bson.DocElem{k, v})
		}
		return doc
	case []interface{}:
		o := make([]interface{}, len(x))
		for i, v := range x {
			o[i] = deepMapAsBsonD(v)
		}
		return o
	}
	return in
}
