module go.mongodb.go/mongo-driver/v2/internal/cmd/faas/awslambda/mongodb

go 1.25

replace go.mongodb.org/mongo-driver/v2 => ../../../../../

require github.com/aws/aws-lambda-go v1.41.0

require go.mongodb.org/mongo-driver/v2 v2.0.0-00010101000000-000000000000

require (
	github.com/klauspost/compress v1.17.6 // indirect
	github.com/stretchr/testify v1.8.1 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.2.0 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78 // indirect
	golang.org/x/crypto v0.45.0 // indirect
	golang.org/x/sync v0.18.0 // indirect
	golang.org/x/text v0.31.0 // indirect
)

replace gopkg.in/yaml.v2 => gopkg.in/yaml.v2 v2.2.8
