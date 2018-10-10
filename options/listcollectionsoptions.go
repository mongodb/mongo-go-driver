package options

// ListCollectionsOptions represents all possible options for a listCollections command.
type ListCollectionsOptions struct {
	NameOnly *bool // If true, only the collection names will be returned.
}

// ListCollections creates a new *ListCollectionsOptions
func ListCollections() *ListCollectionsOptions {
	return &ListCollectionsOptions{}
}

// SetNameOnly specifies whether to return only the collection names.
func (lc *ListCollectionsOptions) SetNameOnly(b bool) *ListCollectionsOptions {
	lc.NameOnly = &b
	return lc
}

func ToListCollectionsOptions(opts ...*ListCollectionsOptions) *ListCollectionsOptions {
	lc := ListCollections()
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if opt.NameOnly != nil {
			lc.NameOnly = opt.NameOnly
		}
	}

	return lc
}
