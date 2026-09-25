module go.mongodb.org/mongo-driver/ext/awsauth

go 1.26

// v1.28.0 is the earliest release whose aws.Credentials field layout matches
// AWSCredentials, which is what lets Retrieve convert between them with a plain
// type conversion. This is a minimum, not a pin: Minimal Version Selection
// ensures users can build against any newer version they need. See doc.go for
// more information.
require github.com/aws/aws-sdk-go-v2 v1.28.0

require github.com/aws/smithy-go v1.27.3 // indirect
