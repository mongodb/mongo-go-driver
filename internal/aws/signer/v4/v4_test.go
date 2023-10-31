// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
//
// Based on github.com/aws/aws-sdk-go by Amazon.com, Inc. with code from:
// - github.com/aws/aws-sdk-go/blob/v1.44.225/aws/signer/v4/v4_test.go
// See THIRD-PARTY-NOTICES for original license terms

package v4

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/internal/aws"
	"go.mongodb.org/mongo-driver/internal/aws/credentials"
	"go.mongodb.org/mongo-driver/internal/credproviders"
)

func epochTime() time.Time { return time.Unix(0, 0) }

func TestStripExcessHeaders(t *testing.T) {
	vals := []string{
		"",
		"123",
		"1 2 3",
		"1 2 3 ",
		"  1 2 3",
		"1  2 3",
		"1  23",
		"1  2  3",
		"1  2  ",
		" 1  2  ",
		"12   3",
		"12   3   1",
		"12           3     1",
		"12     3       1abc123",
	}

	expected := []string{
		"",
		"123",
		"1 2 3",
		"1 2 3",
		"1 2 3",
		"1 2 3",
		"1 23",
		"1 2 3",
		"1 2",
		"1 2",
		"12 3",
		"12 3 1",
		"12 3 1",
		"12 3 1abc123",
	}

	stripExcessSpaces(vals)
	for i := 0; i < len(vals); i++ {
		if e, a := expected[i], vals[i]; e != a {
			t.Errorf("%d, expect %v, got %v", i, e, a)
		}
	}
}

func buildRequest(body string) (*http.Request, io.ReadSeeker) {
	reader := strings.NewReader(body)
	return buildRequestWithBodyReader("dynamodb", "us-east-1", reader)
}

func buildRequestReaderSeeker(serviceName, region, body string) (*http.Request, io.ReadSeeker) {
	reader := &readerSeekerWrapper{strings.NewReader(body)}
	return buildRequestWithBodyReader(serviceName, region, reader)
}

func buildRequestWithBodyReader(serviceName, region string, body io.Reader) (*http.Request, io.ReadSeeker) {
	var bodyLen int

	type lenner interface {
		Len() int
	}
	if lr, ok := body.(lenner); ok {
		bodyLen = lr.Len()
	}

	endpoint := "https://" + serviceName + "." + region + ".amazonaws.com"
	req, _ := http.NewRequest("POST", endpoint, body)
	req.URL.Opaque = "//example.org/bucket/key-._~,!@#$%^&*()"
	req.Header.Set("X-Amz-Target", "prefix.Operation")
	req.Header.Set("Content-Type", "application/x-amz-json-1.0")

	if bodyLen > 0 {
		req.Header.Set("Content-Length", strconv.Itoa(bodyLen))
	}

	req.Header.Set("X-Amz-Meta-Other-Header", "some-value=!@#$%^&* (+)")
	req.Header.Add("X-Amz-Meta-Other-Header_With_Underscore", "some-value=!@#$%^&* (+)")
	req.Header.Add("X-amz-Meta-Other-Header_With_Underscore", "some-value=!@#$%^&* (+)")

	var seeker io.ReadSeeker
	if sr, ok := body.(io.ReadSeeker); ok {
		seeker = sr
	} else {
		seeker = aws.ReadSeekCloser(body)
	}

	return req, seeker
}

func buildSigner() Signer {
	return Signer{
		Credentials: newTestStaticCredentials(),
	}
}

func newTestStaticCredentials() *credentials.Credentials {
	return credentials.NewCredentials(&credproviders.StaticProvider{Value: credentials.Value{
		AccessKeyID:     "AKID",
		SecretAccessKey: "SECRET",
		SessionToken:    "SESSION",
	}})
}

func TestSignRequest(t *testing.T) {
	req, body := buildRequest("{}")
	signer := buildSigner()
	_, err := signer.Sign(req, body, "dynamodb", "us-east-1", epochTime())
	if err != nil {
		t.Errorf("Expected no err, got %v", err)
	}

	expectedDate := "19700101T000000Z"
	expectedSig := "AWS4-HMAC-SHA256 Credential=AKID/19700101/us-east-1/dynamodb/aws4_request, SignedHeaders=content-length;content-type;host;x-amz-date;x-amz-meta-other-header;x-amz-meta-other-header_with_underscore;x-amz-security-token;x-amz-target, Signature=a518299330494908a70222cec6899f6f32f297f8595f6df1776d998936652ad9"

	q := req.Header
	if e, a := expectedSig, q.Get("Authorization"); e != a {
		t.Errorf("expect\n%v\nactual\n%v\n", e, a)
	}
	if e, a := expectedDate, q.Get("X-Amz-Date"); e != a {
		t.Errorf("expect\n%v\nactual\n%v\n", e, a)
	}
}

func TestSignUnseekableBody(t *testing.T) {
	req, body := buildRequestWithBodyReader("mock-service", "mock-region", bytes.NewBuffer([]byte("hello")))
	signer := buildSigner()
	_, err := signer.Sign(req, body, "mock-service", "mock-region", time.Now())
	if err == nil {
		t.Fatalf("expect error signing request")
	}

	if e, a := "unseekable request body", err.Error(); !strings.Contains(a, e) {
		t.Errorf("expect %q to be in %q", e, a)
	}
}

func TestSignPreComputedHashUnseekableBody(t *testing.T) {
	req, body := buildRequestWithBodyReader("mock-service", "mock-region", bytes.NewBuffer([]byte("hello")))

	signer := buildSigner()

	req.Header.Set("X-Amz-Content-Sha256", "some-content-sha256")
	_, err := signer.Sign(req, body, "mock-service", "mock-region", time.Now())
	if err != nil {
		t.Fatalf("expect no error, got %v", err)
	}

	hash := req.Header.Get("X-Amz-Content-Sha256")
	if e, a := "some-content-sha256", hash; e != a {
		t.Errorf("expect %v, got %v", e, a)
	}
}

func TestSignPrecomputedBodyChecksum(t *testing.T) {
	req, body := buildRequest("hello")
	req.Header.Set("X-Amz-Content-Sha256", "PRECOMPUTED")
	signer := buildSigner()
	_, err := signer.Sign(req, body, "dynamodb", "us-east-1", time.Now())
	if err != nil {
		t.Errorf("Expected no err, got %v", err)
	}
	hash := req.Header.Get("X-Amz-Content-Sha256")
	if e, a := "PRECOMPUTED", hash; e != a {
		t.Errorf("expect %v, got %v", e, a)
	}
}

func TestSignWithRequestBody(t *testing.T) {
	creds := newTestStaticCredentials()
	signer := NewSigner(creds)

	expectBody := []byte("abc123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := ioutil.ReadAll(r.Body)
		r.Body.Close()
		if err != nil {
			t.Errorf("expect no error, got %v", err)
		}
		if e, a := expectBody, b; !reflect.DeepEqual(e, a) {
			t.Errorf("expect %v, got %v", e, a)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	req, err := http.NewRequest("POST", server.URL, nil)
	if err != nil {
		t.Errorf("expect not no error, got %v", err)
	}

	_, err = signer.Sign(req, bytes.NewReader(expectBody), "service", "region", time.Now())
	if err != nil {
		t.Errorf("expect not no error, got %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Errorf("expect not no error, got %v", err)
	}
	if e, a := http.StatusOK, resp.StatusCode; e != a {
		t.Errorf("expect %v, got %v", e, a)
	}
}

func TestSignWithRequestBody_Overwrite(t *testing.T) {
	creds := newTestStaticCredentials()
	signer := NewSigner(creds)

	var expectBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := ioutil.ReadAll(r.Body)
		r.Body.Close()
		if err != nil {
			t.Errorf("expect not no error, got %v", err)
		}
		if e, a := len(expectBody), len(b); e != a {
			t.Errorf("expect %v, got %v", e, a)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	req, err := http.NewRequest("GET", server.URL, strings.NewReader("invalid body"))
	if err != nil {
		t.Errorf("expect not no error, got %v", err)
	}

	_, err = signer.Sign(req, nil, "service", "region", time.Now())
	req.ContentLength = 0

	if err != nil {
		t.Errorf("expect not no error, got %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Errorf("expect not no error, got %v", err)
	}
	if e, a := http.StatusOK, resp.StatusCode; e != a {
		t.Errorf("expect %v, got %v", e, a)
	}
}

func TestBuildCanonicalRequest(t *testing.T) {
	req, body := buildRequest("{}")
	req.URL.RawQuery = "Foo=z&Foo=o&Foo=m&Foo=a"
	ctx := &signingCtx{
		ServiceName: "dynamodb",
		Region:      "us-east-1",
		Request:     req,
		Body:        body,
		Query:       req.URL.Query(),
		Time:        time.Now(),
	}

	ctx.buildCanonicalString()
	expected := "https://example.org/bucket/key-._~,!@#$%^&*()?Foo=z&Foo=o&Foo=m&Foo=a"
	if e, a := expected, ctx.Request.URL.String(); e != a {
		t.Errorf("expect %v, got %v", e, a)
	}
}

func TestSignWithBody_ReplaceRequestBody(t *testing.T) {
	creds := newTestStaticCredentials()
	req, seekerBody := buildRequest("{}")
	req.Body = ioutil.NopCloser(bytes.NewReader([]byte{}))

	s := NewSigner(creds)
	origBody := req.Body

	_, err := s.Sign(req, seekerBody, "dynamodb", "us-east-1", time.Now())
	if err != nil {
		t.Fatalf("expect no error, got %v", err)
	}

	if req.Body == origBody {
		t.Errorf("expect request body to not be origBody")
	}

	if req.Body == nil {
		t.Errorf("expect request body to be changed but was nil")
	}
}

func TestRequestHost(t *testing.T) {
	req, body := buildRequest("{}")
	req.URL.RawQuery = "Foo=z&Foo=o&Foo=m&Foo=a"
	req.Host = "myhost"
	ctx := &signingCtx{
		ServiceName: "dynamodb",
		Region:      "us-east-1",
		Request:     req,
		Body:        body,
		Query:       req.URL.Query(),
		Time:        time.Now(),
	}

	ctx.buildCanonicalHeaders(ignoredHeaders, ctx.Request.Header)
	if !strings.Contains(ctx.canonicalHeaders, "host:"+req.Host) {
		t.Errorf("canonical host header invalid")
	}
}

func TestSign_buildCanonicalHeaders(t *testing.T) {
	serviceName := "mockAPI"
	region := "mock-region"
	endpoint := "https://" + serviceName + "." + region + ".amazonaws.com"

	req, err := http.NewRequest("POST", endpoint, nil)
	if err != nil {
		t.Fatalf("failed to create request, %v", err)
	}

	req.Header.Set("FooInnerSpace", "   inner      space    ")
	req.Header.Set("FooLeadingSpace", "    leading-space")
	req.Header.Add("FooMultipleSpace", "no-space")
	req.Header.Add("FooMultipleSpace", "\ttab-space")
	req.Header.Add("FooMultipleSpace", "trailing-space    ")
	req.Header.Set("FooNoSpace", "no-space")
	req.Header.Set("FooTabSpace", "\ttab-space\t")
	req.Header.Set("FooTrailingSpace", "trailing-space    ")
	req.Header.Set("FooWrappedSpace", "   wrapped-space    ")

	ctx := &signingCtx{
		ServiceName: serviceName,
		Region:      region,
		Request:     req,
		Body:        nil,
		Query:       req.URL.Query(),
		Time:        time.Now(),
	}

	ctx.buildCanonicalHeaders(ignoredHeaders, ctx.Request.Header)

	expectCanonicalHeaders := strings.Join([]string{
		`fooinnerspace:inner space`,
		`fooleadingspace:leading-space`,
		`foomultiplespace:no-space,tab-space,trailing-space`,
		`foonospace:no-space`,
		`footabspace:tab-space`,
		`footrailingspace:trailing-space`,
		`foowrappedspace:wrapped-space`,
		`host:mockAPI.mock-region.amazonaws.com`,
	}, "\n")
	if e, a := expectCanonicalHeaders, ctx.canonicalHeaders; e != a {
		t.Errorf("expect:\n%s\n\nactual:\n%s", e, a)
	}
}

func BenchmarkSignRequest(b *testing.B) {
	signer := buildSigner()
	req, body := buildRequestReaderSeeker("dynamodb", "us-east-1", "{}")
	for i := 0; i < b.N; i++ {
		_, err := signer.Sign(req, body, "dynamodb", "us-east-1", time.Now())
		if err != nil {
			b.Errorf("Expected no err, got %v", err)
		}
	}
}

var stripExcessSpaceCases = []string{
	`AWS4-HMAC-SHA256 Credential=AKIDFAKEIDFAKEID/20160628/us-west-2/s3/aws4_request, SignedHeaders=host;x-amz-date, Signature=1234567890abcdef1234567890abcdef1234567890abcdef`,
	`123   321   123   321`,
	`   123   321   123   321   `,
	`   123    321    123          321   `,
	"123",
	"1 2 3",
	"  1 2 3",
	"1  2 3",
	"1  23",
	"1  2  3",
	"1  2  ",
	" 1  2  ",
	"12   3",
	"12   3   1",
	"12           3     1",
	"12     3       1abc123",
}

func BenchmarkStripExcessSpaces(b *testing.B) {
	for i := 0; i < b.N; i++ {
		// Make sure to start with a copy of the cases
		cases := append([]string{}, stripExcessSpaceCases...)
		stripExcessSpaces(cases)
	}
}

// readerSeekerWrapper mimics the interface provided by request.offsetReader
type readerSeekerWrapper struct {
	r *strings.Reader
}

func (r *readerSeekerWrapper) Read(p []byte) (n int, err error) {
	return r.r.Read(p)
}

func (r *readerSeekerWrapper) Seek(offset int64, whence int) (int64, error) {
	return r.r.Seek(offset, whence)
}

func (r *readerSeekerWrapper) Len() int {
	return r.r.Len()
}
