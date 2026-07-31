// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

//go:build cse

package integration

import (
	"context"
	"os"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/integration/mtest"
	"go.mongodb.org/mongo-driver/v2/internal/integtest"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type awsCredentialsProvider struct {
	cnt int
}

func (p *awsCredentialsProvider) Retrieve(context.Context) (options.AWSCredentials, error) {
	p.cnt++
	return options.AWSCredentials{
		AccessKeyID:     awsAccessKeyID,
		SecretAccessKey: awsSecretAccessKey,
	}, nil
}

func TestClientSideEncryptionProse_26_CustomAWS_Case1_CE_WithCredProvidersAndIncorrectKMSProviders(t *testing.T) {
	opts := options.Client().ApplyURI(mtest.ClusterURI())
	integtest.AddTestServerAPIVersion(opts)
	keyVaultClient, err := mongo.Connect(opts)
	require.NoErrorf(t, err, "error on Connect: %v", err)

	var provider awsCredentialsProvider
	ceo := options.ClientEncryption().
		SetKeyVaultNamespace("keyvault.datakeys").
		SetKmsProviders(map[string]map[string]any{
			"aws": {
				"accessKeyId":     awsAccessKeyID,
				"secretAccessKey": awsSecretAccessKey,
			},
		}).
		SetAWSCredentialsProvider(&provider)
	_, err = mongo.NewClientEncryption(keyVaultClient, ceo)
	require.ErrorContains(t, err, "can only provide a custom AWS credential provider",
		"unexpected error: %v", err)
}

func TestClientSideEncryptionProse_26_CustomAWS_Case2_CE_WithCredProviders(t *testing.T) {
	opts := options.Client().ApplyURI(mtest.ClusterURI())
	integtest.AddTestServerAPIVersion(opts)
	keyVaultClient, err := mongo.Connect(opts)
	require.NoErrorf(t, err, "error on Connect: %v", err)

	var provider awsCredentialsProvider
	ceo := options.ClientEncryption().
		SetKeyVaultNamespace("keyvault.datakeys").
		SetKmsProviders(map[string]map[string]any{
			"aws": {},
		}).
		SetAWSCredentialsProvider(&provider)
	clientEncryption, err := mongo.NewClientEncryption(keyVaultClient, ceo)
	require.NoErrorf(t, err, "error on NewClientEncryption: %v", err)

	dkOpts := options.DataKey().SetMasterKey(bson.D{
		{"region", "us-east-1"},
		{"key", "arn:aws:kms:us-east-1:579766882180:key/89fcc2c4-08b0-4bd9-9f25-e30687b580d0"},
	})
	_, err = clientEncryption.CreateDataKey(context.Background(), "aws", dkOpts)
	require.NoErrorf(t, err, "unexpected error %v", err)
	require.GreaterOrEqual(t, provider.cnt, 1, "expected credential provider to be called once")
}

func TestClientSideEncryptionProse_26_CustomAWS_Case3_AE_WithCredProvidersAndIncorrectKMSProviders(t *testing.T) {
	var provider awsCredentialsProvider
	aeo := options.AutoEncryption().
		SetKeyVaultNamespace("keyvault.datakeys").
		SetKmsProviders(map[string]map[string]any{
			"aws": {
				"accessKeyId":     awsAccessKeyID,
				"secretAccessKey": awsSecretAccessKey,
			},
		}).
		SetAWSCredentialsProvider(&provider)
	co := options.Client().SetAutoEncryptionOptions(aeo).ApplyURI(mtest.ClusterURI())
	integtest.AddTestServerAPIVersion(co)
	_, err := mongo.Connect(co)
	require.ErrorContainsf(t, err, "can only provide a custom AWS credential provider",
		"unexpected error: %v", err)
}

func TestClientSideEncryptionProse_26_CustomAWS_Case4_CE_WithCredProvidersAndEnvVars(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", os.Getenv("FLE_AWS_ACCESS_KEY_ID"))
	t.Setenv("AWS_SECRET_ACCESS_KEY", os.Getenv("FLE_AWS_SECRET_ACCESS_KEY"))

	opts := options.Client().ApplyURI(mtest.ClusterURI())
	integtest.AddTestServerAPIVersion(opts)
	keyVaultClient, err := mongo.Connect(opts)
	require.NoErrorf(t, err, "error on Connect: %v", err)

	var provider awsCredentialsProvider
	ceo := options.ClientEncryption().
		SetKeyVaultNamespace("keyvault.datakeys").
		SetKmsProviders(map[string]map[string]any{
			"aws": {},
		}).
		SetAWSCredentialsProvider(&provider)
	clientEncryption, err := mongo.NewClientEncryption(keyVaultClient, ceo)
	require.NoErrorf(t, err, "error on NewClientEncryption: %v", err)

	dkOpts := options.DataKey().SetMasterKey(bson.D{
		{"region", "us-east-1"},
		{"key", "arn:aws:kms:us-east-1:579766882180:key/89fcc2c4-08b0-4bd9-9f25-e30687b580d0"},
	})
	_, err = clientEncryption.CreateDataKey(context.Background(), "aws", dkOpts)
	require.NoErrorf(t, err, "unexpected error %v", err)
	require.GreaterOrEqual(t, provider.cnt, 1, "expected credential provider to be called once")
}
