module go.mongodb.go/mongo-driver/internal/test/mongodb

go 1.20

replace go.mongodb.org/mongo-driver/v2 => ../../../../../

require github.com/aws/aws-lambda-go v1.41.0

require go.mongodb.org/mongo-driver/v2 v2.0.0-00010101000000-000000000000

require (
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/klauspost/compress v1.13.6 // indirect
	github.com/stretchr/testify v1.8.1 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20181117223130-1be2e3e5546d // indirect
	golang.org/x/crypto v0.22.0 // indirect
	golang.org/x/sync v0.7.0 // indirect
	golang.org/x/text v0.14.0 // indirect
)

replace gopkg.in/yaml.v2 => gopkg.in/yaml.v2 v2.2.8
