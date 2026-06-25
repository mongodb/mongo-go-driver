module go.mongodb.org/mongo-driver/ext/awsauth/test

go 1.19

require (
	github.com/aws/aws-sdk-go-v2/config v1.0.0
	go.mongodb.org/mongo-driver/ext/awsauth v0.0.0
	go.mongodb.org/mongo-driver/v2 v2.0.0-beta2
)

require (
	github.com/aws/aws-sdk-go-v2 v1.0.0 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.0.0 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.0.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.0.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.0.0 // indirect
	github.com/aws/smithy-go v1.0.0 // indirect
	github.com/klauspost/compress v1.17.6 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.2.0 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78 // indirect
	golang.org/x/crypto v0.33.0 // indirect
	golang.org/x/sync v0.11.0 // indirect
	golang.org/x/text v0.22.0 // indirect
)

replace (
	go.mongodb.org/mongo-driver/ext/awsauth => ../
	go.mongodb.org/mongo-driver/v2 => ../../..
)
