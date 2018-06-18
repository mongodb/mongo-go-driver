package benchmark

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/mongodb/mongo-go-driver/bson"
)

func bsonMapDecoding(ctx context.Context, tm TimerManager, iters int, dataSet string) error {
	r, err := loadSourceReader(getProjectRoot(), perfDataDir, bsonDataDir, dataSet)
	if err != nil {
		return err
	}

	tm.ResetTimer()

	for i := 0; i < iters; i++ {
		out := make(map[string]interface{})
		dec := bson.NewDecoder(bytes.NewReader(r))
		err := dec.Decode(out)
		if err != nil {
			return err
		}
		if len(out) == 0 {
			return fmt.Errorf("decoding failed")
		}
	}
	return nil
}

func bsonMapEncoding(ctx context.Context, tm TimerManager, iters int, dataSet string) error {
	r, err := loadSourceReader(getProjectRoot(), perfDataDir, bsonDataDir, dataSet)
	if err != nil {
		return err
	}

	doc := make(map[string]interface{})
	dec := bson.NewDecoder(bytes.NewReader(r))
	if err = dec.Decode(doc); err != nil {
		return err
	}

	buf := bytes.NewBuffer([]byte{})
	tm.ResetTimer()
	for i := 0; i < iters; i++ {
		if err = bson.NewEncoder(buf).Encode(&doc); err != nil {
			return nil
		}

		if buf.Len() == 0 {
			return errors.New("encoding failed")
		}
	}

	return nil
}

func BSONFlatMapDecoding(ctx context.Context, tm TimerManager, iters int) error {
	return bsonMapDecoding(ctx, tm, iters, flatBSONData)
}

func BSONFlatMapEncoding(ctx context.Context, tm TimerManager, iters int) error {
	return bsonMapEncoding(ctx, tm, iters, flatBSONData)
}

func BSONDeepMapDecoding(ctx context.Context, tm TimerManager, iters int) error {
	return bsonMapDecoding(ctx, tm, iters, deepBSONData)
}

func BSONDeepMapEncoding(ctx context.Context, tm TimerManager, iters int) error {
	return bsonMapEncoding(ctx, tm, iters, deepBSONData)
}

func BSONFullMapDecoding(ctx context.Context, tm TimerManager, iters int) error {
	return bsonMapDecoding(ctx, tm, iters, fullBSONData)
}

func BSONFullMapEncoding(ctx context.Context, tm TimerManager, iters int) error {
	return bsonMapEncoding(ctx, tm, iters, fullBSONData)
}
