package main

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"go.mongodb.org/mongo-driver/ext/awsauth"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	_ options.AWSCredentialsProvider = (*awsauth.CredentialsProvider)(nil)
	_ options.AWSCredentials         = (awsauth.AWSCredentials)(aws.Credentials{})
)

func main() {}
