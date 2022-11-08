# Frequently Encountered Issues

These are fixes or information for common issues encountered by Go Driver users. If none of these are helpful, [create a GODRIVER Jira ticket](https://jira.mongodb.org/secure/CreateIssue!default.jspa) and we'll try to troubleshoot your issue!

## `WriteXXX` can only write while positioned on a Element or Value but is positioned on a TopLevel

The [`bson.Marshal`](https://pkg.go.dev/go.mongodb.org/mongo-driver/bson#Marshal) function requires a parameter that can be decoded into a BSON Document, i.e. a [`primitive.D`](https://github.com/mongodb/mongo-go-driver/blob/master/bson/bson.go#L31). Therefore the error message

> `WriteXXX` can only write while positioned on a Element or Value but is positioned on a TopLevel

occurs when the input to `bson.Marshal` is something *other* than a BSON Document. Examples of this occurrence include

- `WriteString`: the input into `bson.Marshal` is a string
- `WriteNull`: the input into `bson.Marshal` is null
- `WriteInt32`: the input into `bson.Marshal` is an integer

Many CRUD operations in the Go Driver use `bson.Marshal` under the hood, so it's possible to encounter this particular error without directly attempting to encode data. For example, when using a sort on [`FindOneAndUpdate`](https://pkg.go.dev/go.mongodb.org/mongo-driver/mongo#Collection.FindOneAndUpdate) this error can occur when not properly initializing the `sort` variable:

```go
var sort bson.D // this is nil and will result in a WriteNull error
opts := options.FindOneAndUpdate().SetSort(sort)
update := bson.D{{"$inc", bson.D{{"x", 1}}}}
sr := coll.FindOneAndUpdate(ctx, bson.D{}, update)
if err := sr.Err(); err != nil {
	log.Fatalf("error getting single result: %v", err)
}
```

The above example is resolved by initializing the `sort` variable:

```go
sort := bson.D{}
```

## Convert BSON Document to JSON

There are a variety of marshalers that can be used to encode a BSON document as JSON, including [MarshalExtJSON](https://pkg.go.dev/github.com/mongodb/mongo-go-driver/bson#MarshalExtJSON):

```go
doc := bson.D{{"x", 1}}

jsonBytes, err := bson.MarshalExtJSON(doc, true, false)
if err != nil {
	log.Fatalf("error encoding json: %v", err)
}

m := make(map[string]interface{})
if err := json.Unmarshal(jsonBytes, &m); err != nil {
	log.Fatalf("error decoding json: %v", err)
}
fmt.Printf("json: %v\n", m)
```

## Authentication Failed

When connecting to a MongoDB deployment with password authentication enabled, if your authentication information or configuration are incorrect, you may encounter an error message like:

> connection() error occurred during connection handshake: auth error: sasl conversation error: unable to authenticate using mechanism "SCRAM-SHA-256": (AuthenticationFailed) Authentication failed.

That error can be caused by a number of issues. The error message intentionally omits the exact authentication failure reason (see the [OWASP Authentication and Error Messages guidelines](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html#authentication-and-error-messages) for an explanation). Possible causes of the error include:
- Incorrect password.
- Incorrect username.
- Incorrect authentication database (i.e. [authSource](https://www.mongodb.com/docs/manual/reference/connection-string/#mongodb-urioption-urioption.authSource)).

If you encounter an `AuthenticationFailed` error like the one above, check that the username and password in your [connection string](https://www.mongodb.com/docs/manual/reference/connection-string/) or [SetAuth](https://pkg.go.dev/go.mongodb.org/mongo-driver/mongo/options#ClientOptions.SetAuth) call are correct. In most cases, [authSource](https://www.mongodb.com/docs/manual/reference/connection-string/#mongodb-urioption-urioption.authSource) does not need to be specified unless you are using a non-default authentication database (the default is "admin").
