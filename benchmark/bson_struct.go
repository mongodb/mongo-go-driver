package benchmark

import (
	"bytes"
	"context"
	"errors"

	"github.com/mongodb/mongo-go-driver/bson"
)

func BSONFlatStructDecoding(ctx context.Context, tm TimerManager, iters int) error {
	r, err := loadSourceReader(getProjectRoot(), perfDataDir, bsonDataDir, flatBSONData)
	if err != nil {
		return err
	}

	tm.ResetTimer()

	for i := 0; i < iters; i++ {
		out := flatBSON{}
		dec := bson.NewDecoder(bytes.NewReader(r))
		err := dec.Decode(&out)
		if err != nil {
			return err
		}
	}
	return nil
}

func BSONFlatStructEncoding(ctx context.Context, tm TimerManager, iters int) error {
	r, err := loadSourceReader(getProjectRoot(), perfDataDir, bsonDataDir, flatBSONData)
	if err != nil {
		return err
	}

	doc := flatBSON{}
	if err = bson.NewDecoder(bytes.NewReader(r)).Decode(&doc); err != nil {
		return err
	}

	buf := bytes.NewBuffer([]byte{})

	tm.ResetTimer()
	for i := 0; i < iters; i++ {
		if err = bson.NewEncoder(buf).Encode(&doc); err != nil {
			return err
		}
		if buf.Len() == 0 {
			return errors.New("encoding failed")
		}
	}
	return nil
}

func BSONFlatStructTagsEncoding(ctx context.Context, tm TimerManager, iters int) error {
	r, err := loadSourceReader(getProjectRoot(), perfDataDir, bsonDataDir, flatBSONData)
	if err != nil {
		return err
	}

	doc := flatBSONTags{}
	if err = bson.NewDecoder(bytes.NewReader(r)).Decode(&doc); err != nil {
		return err
	}

	buf := bytes.NewBuffer([]byte{})

	tm.ResetTimer()
	for i := 0; i < iters; i++ {
		if err = bson.NewEncoder(buf).Encode(&doc); err != nil {
			return err
		}
		if buf.Len() == 0 {
			return errors.New("encoding failed")
		}
	}
	return nil
}

func BSONFlatStructTagsDecoding(ctx context.Context, tm TimerManager, iters int) error {
	r, err := loadSourceReader(getProjectRoot(), perfDataDir, bsonDataDir, flatBSONData)
	if err != nil {
		return err
	}

	tm.ResetTimer()
	for i := 0; i < iters; i++ {
		out := flatBSONTags{}
		dec := bson.NewDecoder(bytes.NewReader(r))
		err := dec.Decode(&out)
		if err != nil {
			return err
		}
	}
	return nil
}
