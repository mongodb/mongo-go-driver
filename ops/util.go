package ops

import "github.com/10gen/mongo-go-driver/readpref"

func slaveOk(rp *readpref.ReadPref) bool {
	if rp == nil {
		// assume primary
		return false
	}

	return rp.Mode() != readpref.PrimaryMode
}
