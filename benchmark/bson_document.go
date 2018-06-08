package benchmark

import (
	"context"
	"errors"

	"github.com/mongodb/mongo-go-driver/bson"
)

func BSONFlatDocumentEncoding(ctx context.Context, tm TimerManager, iters int) error {
	doc, err := loadSourceDocument(getProjectRoot(), perfDataDir, bsonDataDir, flatBSONData)
	if err != nil {
		return err
	}

	tm.ResetTimer()

	for i := 0; i < iters; i++ {
		out, err := doc.MarshalBSON()
		if err != nil {
			return err
		}
		if len(out) == 0 {
			return errors.New("marshaling error")
		}
	}

	return nil
}

func BSONFlatDocumentDecodingLazy(ctx context.Context, tm TimerManager, iters int) error {
	doc, err := loadSourceDocument(getProjectRoot(), perfDataDir, bsonDataDir, flatBSONData)
	if err != nil {
		return err
	}

	raw, err := doc.MarshalBSON()
	if err != nil {
		return err
	}

	tm.ResetTimer()

	for i := 0; i < iters; i++ {
		out, err := bson.ReadDocument(raw)
		if err != nil {
			return err
		}
		if out.Len() == 0 {
			return errors.New("marshaling error")
		}
	}
	return nil
}

func BSONFlatDocumentDecoding(ctx context.Context, tm TimerManager, iters int) error {
	doc, err := loadSourceDocument(getProjectRoot(), perfDataDir, bsonDataDir, flatBSONData)
	if err != nil {
		return err
	}

	raw, err := doc.MarshalBSON()
	if err != nil {
		return err
	}

	tm.ResetTimer()

	for i := 0; i < iters; i++ {
		out, err := bson.ReadDocument(raw)
		if err != nil {
			return err
		}

		keys, err := out.Keys(true)
		if err != nil {
			return err
		}
		if len(keys) != 145 {
			return errors.New("document parsing error")
		}
	}
	return nil
}

func BSONDeepDocumentEncoding(ctx context.Context, tm TimerManager, iters int) error {
	doc, err := loadSourceDocument(getProjectRoot(), perfDataDir, bsonDataDir, deepBSONData)
	if err != nil {
		return err
	}

	tm.ResetTimer()

	for i := 0; i < iters; i++ {
		out, err := doc.MarshalBSON()
		if err != nil {
			return err
		}
		if len(out) == 0 {
			return errors.New("marshaling error")
		}
	}

	return nil
}

func BSONDeepDocumentDecodingLazy(ctx context.Context, tm TimerManager, iters int) error {
	doc, err := loadSourceDocument(getProjectRoot(), perfDataDir, bsonDataDir, deepBSONData)
	if err != nil {
		return err
	}

	raw, err := doc.MarshalBSON()
	if err != nil {
		return err
	}

	tm.ResetTimer()

	for i := 0; i < iters; i++ {
		out, err := bson.ReadDocument(raw)
		if err != nil {
			return err
		}
		if out.Len() == 0 {
			return errors.New("marshaling error")
		}
	}
	return nil
}

func BSONDeepDocumentDecoding(ctx context.Context, tm TimerManager, iters int) error {
	doc, err := loadSourceDocument(getProjectRoot(), perfDataDir, bsonDataDir, deepBSONData)
	if err != nil {
		return err
	}

	raw, err := doc.MarshalBSON()
	if err != nil {
		return err
	}

	tm.ResetTimer()

	for i := 0; i < iters; i++ {
		out, err := bson.ReadDocument(raw)
		if err != nil {
			return err
		}

		keys, err := out.Keys(true)
		if err != nil {
			return err
		}
		if len(keys) != 126 {
			return errors.New("incomplete bson")
		}
	}
	return nil
}

func BSONFullDocumentEncoding(ctx context.Context, tm TimerManager, iters int) error {
	doc, err := loadSourceDocument(getProjectRoot(), perfDataDir, bsonDataDir, fullBSONData)
	if err != nil {
		return err
	}

	tm.ResetTimer()

	for i := 0; i < iters; i++ {
		out, err := doc.MarshalBSON()
		if err != nil {
			return err
		}
		if len(out) == 0 {
			return errors.New("marshaling error")
		}
	}

	return nil
}

func BSONFullDocumentDecodingLazy(ctx context.Context, tm TimerManager, iters int) error {
	doc, err := loadSourceDocument(getProjectRoot(), perfDataDir, bsonDataDir, fullBSONData)
	if err != nil {
		return err
	}

	raw, err := doc.MarshalBSON()
	if err != nil {
		return err
	}

	tm.ResetTimer()

	for i := 0; i < iters; i++ {
		out, err := bson.ReadDocument(raw)
		if err != nil {
			return err
		}
		if out.Len() == 0 {
			return errors.New("marshaling error")
		}
	}
	return nil
}

func BSONFullDocumentDecoding(ctx context.Context, tm TimerManager, iters int) error {
	doc, err := loadSourceDocument(getProjectRoot(), perfDataDir, bsonDataDir, fullBSONData)
	if err != nil {
		return err
	}

	raw, err := doc.MarshalBSON()
	if err != nil {
		return err
	}

	tm.ResetTimer()

	for i := 0; i < iters; i++ {
		out, err := bson.ReadDocument(raw)
		if err != nil {
			return err
		}

		keys, err := out.Keys(true)
		if err != nil {
			return err
		}
		if len(keys) != 145 {
			return errors.New("incomplete bson")
		}
	}
	return nil
}
