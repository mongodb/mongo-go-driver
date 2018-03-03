package description

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
