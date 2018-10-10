package options

// ListDatabasesOptions represents all possible options for a listDatabases command.
type ListDatabasesOptions struct {
	NameOnly *bool // If true, only the database names will be returned.
}

// ListDatabases creates a new *ListDatabasesOptions
func ListDatabases() *ListDatabasesOptions {
	return &ListDatabasesOptions{}
}

// SetNameOnly specifies whether to return only the database names.
func (ld *ListDatabasesOptions) SetNameOnly(b bool) *ListDatabasesOptions {
	ld.NameOnly = &b
	return ld
}

func ToListDatabasesOptions(opts ...*ListDatabasesOptions) *ListDatabasesOptions {
	ld := ListDatabases()
	for _, opt := range opts {
		if opts == nil {
			continue
		}
		if opt.NameOnly != nil {
			ld.NameOnly = opt.NameOnly
		}
	}

	return ld
}
