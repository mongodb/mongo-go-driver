package distinctopt

import (
	"time"

	"github.com/mongodb/mongo-go-driver/options-design/mongo/mongoopt"
	"github.com/mongodb/mongo-go-driver/options-design/option"
)

type Distinct interface {
	distinct()
}

type DistinctBundle struct{}

func (db *DistinctBundle) distinct() {}

func BundleDistinct(...Distinct) *DistinctBundle {
	return nil
}

func (db *DistinctBundle) Collation(collation *mongoopt.Collation) *DistinctBundle {
	return nil
}

func (db *DistinctBundle) MaxTime(d time.Duration) *DistinctBundle {
	return nil
}

func (db *DistinctBundle) Unbundle(deduplicate bool) []option.Optioner {
	return nil
}

func Collation(collation *mongoopt.Collation) OptCollation {
	return OptCollation{}
}

func MaxTime(d time.Duration) OptMaxTime {
	return 0
}

type OptCollation option.OptCollation

func (OptCollation) distinct() {}

type OptMaxTime option.OptMaxTime

func (OptMaxTime) distinct() {}
