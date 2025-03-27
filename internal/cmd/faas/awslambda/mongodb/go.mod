module go.mongodb.go/mongo-driver/internal/test/mongodb

go 1.23

replace go.mongodb.org/mongo-driver/v2 => ../../../../../

require github.com/aws/aws-lambda-go v1.41.0

require go.mongodb.org/mongo-driver/v2 v2.0.0-00010101000000-000000000000

require (
	github.com/golang/snappy v1.0.0 // indirect
	github.com/klauspost/compress v1.16.7 // indirect
	github.com/stretchr/testify v1.8.1 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78 // indirect
	golang.org/x/crypto v0.33.0 // indirect
	golang.org/x/sync v0.11.0 // indirect
	golang.org/x/text v0.22.0 // indirect
)

replace gopkg.in/yaml.v2 => gopkg.in/yaml.v2 v2.2.8
