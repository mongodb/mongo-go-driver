package benchmark

import (
	"context"
	"errors"

	"github.com/mongodb/mongo-go-driver/bson/bsoncodec"
)

func BSONFlatStructDecoding(ctx context.Context, tm TimerManager, iters int) error {
	r, err := loadSourceReader(getProjectRoot(), perfDataDir, bsonDataDir, flatBSONData)
	if err != nil {
		return err
	}

	tm.ResetTimer()

	for i := 0; i < iters; i++ {
		out := flatBSON{}
		err := bsoncodec.Unmarshal(r, &out)
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
	err = bsoncodec.Unmarshal(r, &doc)
	if err != nil {
		return err
	}

	var buf []byte

	tm.ResetTimer()
	for i := 0; i < iters; i++ {
		buf, err = bsoncodec.Marshal(doc)
		if err != nil {
			return err
		}
		if len(buf) == 0 {
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
	err = bsoncodec.Unmarshal(r, &doc)
	if err != nil {
		return err
	}

	var buf []byte

	tm.ResetTimer()
	for i := 0; i < iters; i++ {
		buf, err = bsoncodec.MarshalAppend(buf[:0], doc)
		if err != nil {
			return err
		}
		if len(buf) == 0 {
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
		err := bsoncodec.Unmarshal(r, &out)
		if err != nil {
			return err
		}
	}
	return nil
}
