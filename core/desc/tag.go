package desc

// Tag is a name/value pair.
type Tag struct {
	Name  string
	Value string
}

// TagSet is an ordered list of Tags.
type TagSet []Tag
