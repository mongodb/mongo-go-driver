module go.mongodb.go/mongo-driver/examples/logger/logrus

go 1.20

replace go.mongodb.org/mongo-driver => ../../../

require (
	github.com/bombsimon/logrusr/v4 v4.0.0
	github.com/sirupsen/logrus v1.9.0
	// Note that the Go driver version is replaced with the local Go driver code
	// by the replace directive above.
	go.mongodb.org/mongo-driver v1.11.7
)

require (
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/klauspost/compress v1.13.6 // indirect
	github.com/montanaflynn/stats v0.7.1 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78 // indirect
	golang.org/x/crypto v0.26.0 // indirect
	golang.org/x/sync v0.8.0 // indirect
	golang.org/x/sys v0.23.0 // indirect
	golang.org/x/text v0.17.0 // indirect
)
