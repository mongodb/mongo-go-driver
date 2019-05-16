package operation

//go:generate operationgen insert.toml operation insert.go
//go:generate operationgen find.toml operation find.go
//go:generate operationgen list_collections.toml operation list_collections.go
//go:generate operationgen createIndexes.toml operation createIndexes.go
//go:generate operationgen drop_collection.toml operation drop_collection.go
//go:generate operationgen distinct.toml operation distinct.go
//go:generate operationgen delete.toml operation delete.go
