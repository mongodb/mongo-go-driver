module go.mongodb.go/mongo-driver/internal/test/compilecheck

go 1.13

replace go.mongodb.org/mongo-driver => ../../../

// Note that the Go driver version is replaced with the local Go driver code by
// the replace directive above.
require go.mongodb.org/mongo-driver v1.11.7
