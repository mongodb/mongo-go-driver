package mongo

type BulkWriteResult struct {
	acknowledged bool
	inserted     int64
	insertIDs    map[int64]interface{}
	matched      int64
	modified     int64
	deleted      int64
	upserted     int64
	upsertedIDs  map[int64]interface{}
}

type InsertOneResult struct {
	acknowldged bool
	insertedID  interface{}
}

type InsertManyResult struct {
	acknowledged bool
	insertedIDs  map[int64]interface{}
}

type DeleteResult struct {
	acknowledged bool
	deleted      int64
}

type UpdateResult struct {
	acknowledged bool
	matched      int64
	modified     int64
	upsertID     interface{}
}

type DocumentResult struct {
	err error
	cur Cursor
}

func (dr *DocumentResult) Decode(interface{}) error { return nil }
