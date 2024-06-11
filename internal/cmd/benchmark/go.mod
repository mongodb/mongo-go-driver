module go.mongodb.go/mongo-driver/internal/cmd/benchmark

go 1.22

toolchain go1.22.3

replace go.mongodb.org/mongo-driver => ../../../

// Note that the Go driver version is replaced with the local Go driver code by
// the replace directive above.
//require go.mongodb.org/mongo-driver
require go.mongodb.org/mongo-driver v1.11.7

require github.com/montanaflynn/stats v0.7.1

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/klauspost/compress v1.17.8 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20240424034433-3c2c7870ae76 // indirect
	golang.org/x/crypto v0.24.0 // indirect
	golang.org/x/sync v0.7.0 // indirect
	golang.org/x/text v0.16.0 // indirect
)
