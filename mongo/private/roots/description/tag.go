package description

import "github.com/mongodb/mongo-go-driver/mongo/model"

// Tag is a name/vlaue pair.
type Tag struct {
	Name  string
	Value string
}

// NewTagSetFromMap creates a new tag set from a map.
func NewTagSetFromMap(m map[string]string) TagSet {
	var set TagSet
	for k, v := range m {
		set = append(set, Tag{Name: k, Value: v})
	}

	return set
}

// NewTagSetsFromMaps creates new tag sets from maps.
func NewTagSetsFromMaps(maps []map[string]string) []TagSet {
	sets := make([]TagSet, 0, len(maps))
	for _, m := range maps {
		sets = append(sets, NewTagSetFromMap(m))
	}
	return sets
}

// TagSet is an ordered list of Tags.
type TagSet []Tag

// Contains indicates whether the name/value pair exists in the tagset.
func (ts TagSet) Contains(name, value string) bool {
	for _, t := range ts {
		if t.Name == name && t.Value == value {
			return true
		}
	}

	return false
}

// ContainsAll indicates whether all the name/value pairs exist in the tagset.
func (ts TagSet) ContainsAll(other []Tag) bool {
	for _, ot := range other {
		if !ts.Contains(ot.Name, ot.Value) {
			return false
		}
	}

	return true
}

// ConvertTagSets converts a []model.TagSet to a []TagSet.
func ConvertTagSets(ts []model.TagSet) []TagSet {
	result := make([]TagSet, 0, len(ts))
	for _, set := range ts {
		result = append(result, ConvertTagSet(set))
	}
	return result
}

// ConvertTagSet converts a model.TagSet to a TagSet.
func ConvertTagSet(ts model.TagSet) TagSet {
	result := make(TagSet, 0, len(ts))
	for _, t := range ts {
		result = append(result, ConvertTag(t))
	}
	return result
}

//ConvertTag converts a model.Tag to a Tag.
func ConvertTag(t model.Tag) Tag {
	return Tag{Name: t.Name, Value: t.Value}
}
