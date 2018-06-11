package countopt

import (
	"github.com/mongodb/mongo-go-driver/options-design/mongo/readconcern"
	"github.com/mongodb/mongo-go-driver/options-design/option"
)

type Count interface {
	count()
}
type CountBundle struct{}

func (cb *CountBundle) count() {}

func BundleCount(...Count) *CountBundle {
	return nil
}

func (cb *CountBundle) Limit(i int32) *CountBundle {
	return nil
}

func (cb *CountBundle) Skip(i int32) *CountBundle {
	return nil
}

func (cb *CountBundle) Hint(hint interface{}) *CountBundle {
	return nil
}

func (cb *CountBundle) MaxTimeMs(i int32) *CountBundle {
	return nil
}

func (cb *CountBundle) ReadConcern(rc *readconcern.ReadConcern) *CountBundle {
	return nil
}

func (cb *CountBundle) Unbundle(deduplicate bool) []option.CountOptioner {
	return nil
}

func Limit(i int32) OptLimit {
	return 0
}

func Skip(i int32) OptSkip {
	return 0
}

func Hint(hint interface{}) OptHint {
	return OptHint{}
}

func MaxTimeMs(i int32) OptMaxTimeMs {
	return 0
}

func ReadConcern(rc *readconcern.ReadConcern) OptReadConcern {
	return OptReadConcern{}
}

type OptLimit option.OptLimit
type OptSkip option.OptSkip
type OptHint option.OptHint
type OptMaxTimeMs option.OptMaxTime
type OptReadConcern option.OptReadConcern

func (OptLimit) count()       {}
func (OptSkip) count()        {}
func (OptHint) count()        {}
func (OptMaxTimeMs) count()   {}
func (OptReadConcern) count() {}
