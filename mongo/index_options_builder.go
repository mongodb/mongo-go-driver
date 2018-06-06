package mongo

import "github.com/mongodb/mongo-go-driver/bson"

// IndexOptionsBuilder constructs a BSON document for index options
type IndexOptionsBuilder struct {
	document *bson.Document
}

// NewIndexOptionsBuilder creates a new instance of IndexOptionsBuilder
func NewIndexOptionsBuilder() *IndexOptionsBuilder {
	var b IndexOptionsBuilder
	b.document = bson.NewDocument()
	return &b
}

func (iob *IndexOptionsBuilder) setBackground(background bool) *IndexOptionsBuilder {
	iob.document.Append(bson.EC.Boolean("background", background))
	return iob
}

func (iob *IndexOptionsBuilder) setExpireAfter(expireAfter int32) *IndexOptionsBuilder {
	iob.document.Append(bson.EC.Int32("expireAfter", expireAfter))
	return iob
}

func (iob *IndexOptionsBuilder) setName(name string) *IndexOptionsBuilder {
	iob.document.Append(bson.EC.String("name", name))
	return iob
}

func (iob *IndexOptionsBuilder) setSparse(sparse bool) *IndexOptionsBuilder {
	iob.document.Append(bson.EC.Boolean("sparse", sparse))
	return iob
}

func (iob *IndexOptionsBuilder) setStorageEngine(storageEngine string) *IndexOptionsBuilder {
	iob.document.Append(bson.EC.String("storageEngine", storageEngine))
	return iob
}

func (iob *IndexOptionsBuilder) setUnique(unique bool) *IndexOptionsBuilder {
	iob.document.Append(bson.EC.Boolean("unique", unique))
	return iob
}

func (iob *IndexOptionsBuilder) setVersion(version int32) *IndexOptionsBuilder {
	iob.document.Append(bson.EC.Int32("version", version))
	return iob
}

func (iob *IndexOptionsBuilder) setDefaultLanguage(defaultLanguage string) *IndexOptionsBuilder {
	iob.document.Append(bson.EC.String("defaultLanguage", defaultLanguage))
	return iob
}

func (iob *IndexOptionsBuilder) setLanguageOverride(languageOverride string) *IndexOptionsBuilder {
	iob.document.Append(bson.EC.String("languageOverride", languageOverride))
	return iob
}

func (iob *IndexOptionsBuilder) setTextVersion(textVersion int32) *IndexOptionsBuilder {
	iob.document.Append(bson.EC.Int32("textVersion", textVersion))
	return iob
}

func (iob *IndexOptionsBuilder) setWeights(weights *bson.Document) *IndexOptionsBuilder {
	iob.document.Append(bson.EC.SubDocument("weights", weights))
	return iob
}

func (iob *IndexOptionsBuilder) setSphereVersion(sphereVersion int32) *IndexOptionsBuilder {
	iob.document.Append(bson.EC.Int32("sphereVersion", sphereVersion))
	return iob
}

func (iob *IndexOptionsBuilder) setBits(bits int32) *IndexOptionsBuilder {
	iob.document.Append(bson.EC.Int32("bits", bits))
	return iob
}

func (iob *IndexOptionsBuilder) setMax(max float64) *IndexOptionsBuilder {
	iob.document.Append(bson.EC.Double("max", max))
	return iob
}

func (iob *IndexOptionsBuilder) setMin(min float64) *IndexOptionsBuilder {
	iob.document.Append(bson.EC.Double("min", min))
	return iob
}

func (iob *IndexOptionsBuilder) setBucketSize(bucketSize int32) *IndexOptionsBuilder {
	iob.document.Append(bson.EC.Int32("bucketSize", bucketSize))
	return iob
}

func (iob *IndexOptionsBuilder) setPartialFilterExpression(partialFilterExpression *bson.Document) *IndexOptionsBuilder {
	iob.document.Append(bson.EC.SubDocument("partialFilterExpression", partialFilterExpression))
	return iob
}

func (iob *IndexOptionsBuilder) setCollation(collation *bson.Document) *IndexOptionsBuilder {
	iob.document.Append(bson.EC.SubDocument("collation", collation))
	return iob
}

func (iob *IndexOptionsBuilder) build() *bson.Document {
	return iob.document
}
