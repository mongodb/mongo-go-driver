package benchmark

import (
	"context"
	"errors"
)

func BSONFlatReaderDecoding(ctx context.Context, iters int) error {
	doc, err := loadSourceReader(getProjectRoot(), perfDataDir, bsonDataDir, flatBSONData)
	if err != nil {
		return err
	}

	for i := 0; i < iters; i++ {
		keys, err := doc.Keys(true)
		if err != nil {
			return err
		}
		if len(keys) != 145 {
			return errors.New("reader parsing error")
		}
	}
	return nil
}

func BSONDeepReaderDecoding(ctx context.Context, iters int) error {
	doc, err := loadSourceReader(getProjectRoot(), perfDataDir, bsonDataDir, deepBSONData)
	if err != nil {
		return err
	}

	for i := 0; i < iters; i++ {
		keys, err := doc.Keys(true)
		if err != nil {
			return err
		}
		if len(keys) != 126 {
			return errors.New("incomplete bson")
		}
	}
	return nil
}

func BSONFullReaderDecoding(ctx context.Context, iters int) error {
	doc, err := loadSourceReader(getProjectRoot(), perfDataDir, bsonDataDir, fullBSONData)
	if err != nil {
		return err
	}

	for i := 0; i < iters; i++ {
		keys, err := doc.Keys(true)
		if err != nil {
			return err
		}
		if len(keys) != 145 {
			return errors.New("incomplete bson")
		}
	}
	return nil
}
