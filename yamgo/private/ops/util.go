package ops

import (
	"github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/mongo-go-driver/yamgo/model"
	"github.com/10gen/mongo-go-driver/yamgo/readpref"
)

func slaveOk(selectedServer *SelectedServer) bool {
	if selectedServer.ClusterKind == model.Single && selectedServer.Model().Kind != model.Mongos {
		return true
	}

	if selectedServer.ReadPref == nil {
		// assume primary
		return false
	}

	return selectedServer.ReadPref.Mode() != readpref.PrimaryMode
}

func readPrefMeta(rp *readpref.ReadPref, kind model.ServerKind) interface{} {
	if kind != model.Mongos || rp == nil {
		return nil
	}

	// simple Primary or SecondaryPreferred is communicated via slaveOk to Mongos.
	if rp.Mode() == readpref.PrimaryMode || rp.Mode() == readpref.SecondaryPreferredMode {
		if _, ok := rp.MaxStaleness(); !ok && len(rp.TagSets()) == 0 {
			return nil
		}
	}

	var doc struct {
		Mode                string   `bson:"mode,omitempty"`
		Tags                []bson.D `bson:"tags,omitempty"`
		MaxStalenessSeconds uint32   `bson:"maxStalenessSeconds,omitempty"`
	}

	switch rp.Mode() {
	case readpref.PrimaryMode:
		doc.Mode = "primary"
	case readpref.PrimaryPreferredMode:
		doc.Mode = "primaryPreferred"
	case readpref.SecondaryPreferredMode:
		doc.Mode = "secondaryPreferred"
	case readpref.SecondaryMode:
		doc.Mode = "secondary"
	case readpref.NearestMode:
		doc.Mode = "nearest"
	}

	for _, ts := range rp.TagSets() {
		set := bson.D{}
		for _, t := range ts {
			set = append(set, bson.DocElem{Name: t.Name, Value: t.Value})
		}
		if len(set) > 0 {
			doc.Tags = append(doc.Tags, set)
		}
	}

	if d, ok := rp.MaxStaleness(); ok {
		doc.MaxStalenessSeconds = uint32(d.Seconds())
	}

	return doc
}
