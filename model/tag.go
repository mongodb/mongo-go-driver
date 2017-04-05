package model

// Tag is a name/value pair.
type Tag struct {
	Name  string
	Value string
}

// NewTagSet creates a new tag set by taking the entries in pairs.
func NewTagSet(tags ...string) TagSet {
	if len(tags)%2 != 0 {
		panic("desc.NewTagSet: argument count is odd")
	}

	var set TagSet
	for i := 0; i < len(tags); i += 2 {
		set = append(set, Tag{Name: tags[i], Value: tags[i+1]})
	}
	return set
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

// Contains indicates whether the name/value pair
// exists in the tag set.
func (ts TagSet) Contains(name, value string) bool {
	for _, t := range ts {
		if t.Name == name && t.Value == value {
			return true
		}
	}

	return false
}

// ContainsAll indicates whether all the name/value pairs
// exist in the tag set.
func (ts TagSet) ContainsAll(other []Tag) bool {
	for _, ot := range other {
		if !ts.Contains(ot.Name, ot.Value) {
			return false
		}
	}

	return true
}
