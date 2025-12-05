module go.mongodb.go/mongo-driver/v2/internal/cmd/compilecheck

go 1.19

replace go.mongodb.org/mongo-driver/v2 => ../../../

// Note that the Go driver version is replaced with the local Go driver code by
// the replace directive above.
require go.mongodb.org/mongo-driver/v2 v2.0.0-alpha2

require (
	github.com/golang/snappy v1.0.0 // indirect
	github.com/klauspost/compress v1.17.6 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.2.0 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78 // indirect
	golang.org/x/crypto v0.33.0 // indirect
	golang.org/x/sync v0.11.0 // indirect
	golang.org/x/text v0.22.0 // indirect
)
