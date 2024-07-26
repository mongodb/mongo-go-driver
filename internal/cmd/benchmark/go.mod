module go.mongodb.go/mongo-driver/internal/cmd/benchmark

go 1.18

replace go.mongodb.org/mongo-driver/v2 => ../../../

// Note that the Go driver version is replaced with the local Go driver code by
// the replace directive above.

require (
	github.com/montanaflynn/stats v0.7.1
	github.com/stretchr/testify v1.9.0
	go.mongodb.org/mongo-driver/v2 v2.0.0-alpha2
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/klauspost/compress v1.13.6 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20181117223130-1be2e3e5546d // indirect
	golang.org/x/crypto v0.24.0 // indirect
	golang.org/x/sync v0.7.0 // indirect
	golang.org/x/text v0.16.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
