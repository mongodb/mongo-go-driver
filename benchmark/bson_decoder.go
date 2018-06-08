package benchmark

import (
	"bytes"
	"context"
	"errors"

	"github.com/mongodb/mongo-go-driver/bson"
)

func BSONFlatMapDecoding(ctx context.Context, iters int) error {
	r, err := loadSourceReader(getProjectRoot(), perfDataDir, bsonDataDir, flatBSONData)
	if err != nil {
		return err
	}

	out := make(map[string]interface{})
	for i := 0; i < iters; i++ {
		dec := bson.NewDecoder(bytes.NewReader(r))
		err := dec.Decode(out)
		if err != nil {
			return err
		}
		if len(out) == 0 {
			return errors.New("decoding failed")
		}
	}
	return nil
}

func BSONFlatStructDecoding(ctx context.Context, iters int) error {
	r, err := loadSourceReader(getProjectRoot(), perfDataDir, bsonDataDir, flatBSONData)
	if err != nil {
		return err
	}

	out := flatBSON{}
	for i := 0; i < iters; i++ {
		dec := bson.NewDecoder(bytes.NewReader(r))
		err := dec.Decode(&out)
		if err != nil {
			return err
		}
	}
	return nil
}

func BSONFlatStructEncoding(ctx context.Context, iters int) error {
	r, err := loadSourceReader(getProjectRoot(), perfDataDir, bsonDataDir, flatBSONData)
	if err != nil {
		return err
	}

	doc := flatBSON{}
	if err = bson.NewDecoder(bytes.NewReader(r)).Decode(&doc); err != nil {
		return err
	}

	for i := 0; i < iters; i++ {
		buf := bytes.NewBuffer([]byte{})
		if err = bson.NewEncoder(buf).Encode(&doc); err != nil {
			return err
		}
		if buf.Len() == 0 {
			return errors.New("encoding failed")
		}
	}
	return nil
}

func BSONFlatStructTagsEncoding(ctx context.Context, iters int) error {
	r, err := loadSourceReader(getProjectRoot(), perfDataDir, bsonDataDir, flatBSONData)
	if err != nil {
		return err
	}

	doc := flatBSONTags{}
	if err = bson.NewDecoder(bytes.NewReader(r)).Decode(&doc); err != nil {
		return err
	}

	for i := 0; i < iters; i++ {
		buf := bytes.NewBuffer([]byte{})
		if err = bson.NewEncoder(buf).Encode(&doc); err != nil {
			return err
		}
		if buf.Len() == 0 {
			return errors.New("encoding failed")
		}
	}
	return nil
}

func BSONFlatStructTagsDecoding(ctx context.Context, iters int) error {
	r, err := loadSourceReader(getProjectRoot(), perfDataDir, bsonDataDir, flatBSONData)
	if err != nil {
		return err
	}

	out := flatBSONTags{}
	for i := 0; i < iters; i++ {
		dec := bson.NewDecoder(bytes.NewReader(r))
		err := dec.Decode(&out)
		if err != nil {
			return err
		}
	}
	return nil
}

func BSONDeepMapDecoding(ctx context.Context, iters int) error {
	r, err := loadSourceReader(getProjectRoot(), perfDataDir, bsonDataDir, deepBSONData)
	if err != nil {
		return err
	}

	out := make(map[string]interface{})
	for i := 0; i < iters; i++ {
		dec := bson.NewDecoder(bytes.NewReader(r))
		err := dec.Decode(out)
		if err != nil {
			return err
		}
		if len(out) == 0 {
			return errors.New("decoding failed")
		}
	}
	return nil
}

func BSONFullMapDecoding(ctx context.Context, iters int) error {
	r, err := loadSourceReader(getProjectRoot(), perfDataDir, bsonDataDir, fullBSONData)
	if err != nil {
		return err
	}

	out := make(map[string]interface{})
	for i := 0; i < iters; i++ {
		dec := bson.NewDecoder(bytes.NewReader(r))
		err := dec.Decode(out)
		if err != nil {
			return err
		}
		if len(out) == 0 {
			return errors.New("decoding failed")
		}
	}
	return nil
}
