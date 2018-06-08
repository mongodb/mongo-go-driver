package benchmark

import (
	"context"
	"errors"
)

func bsonReaderDecoding(ctx context.Context, tm TimerManager, iters, keys int, source string) error {
	doc, err := loadSourceReader(getProjectRoot(), perfDataDir, bsonDataDir, source)
	if err != nil {
		return err
	}

	tm.ResetTimer()

	for i := 0; i < iters; i++ {
		keys, err := doc.Keys(true)
		if err != nil {
			return err
		}
		if len(keys) != keys {
			return errors.New("reader parsing error")
		}
	}

	return nil
}

func BSONFlatReaderDecoding(ctx context.Context, tm TimerManager, iters int) error {
	return bsonReaderDecoding(ctx, tm, iters, 145, flatBSONData)
}

func BSONDeepReaderDecoding(ctx context.Context, tm TimerManager, iters int) error {
	return bsonReaderDecoding(ctx, tm, iters, 126, deepBSONData)
}

func BSONFullReaderDecoding(ctx context.Context, tm TimerManager, iters int) error {
	return bsonReaderDecoding(ctx, tm, 145, fullBSONData)
}
