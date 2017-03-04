package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/10gen/mongo-go-driver/auth"
	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/msg"
	"gopkg.in/mgo.v2/bson"
)

var host = flag.String("host", "localhost:27017", "the host to authenticate against")
var source = flag.String("source", "admin", "where the user exists")
var mech = flag.String("mech", "SCRAM-SHA-1", "the mechanism to use")
var props = flag.String("props", "", "mechanism properties")

var db = flag.String("db", "test", "where to issue a count command")
var col = flag.String("col", "test", "where to issue a count command")

func main() {
	flag.Parse()
	args := flag.Args()

	props, err := parseProps()
	if err != nil {
		log.Fatalf("failed to parse props flag: %v", err)
	}

	cred := &auth.Cred{
		Source: *source,
		Props:  props,
	}

	if len(args) == 0 {
		if *mech != "GSSAPI" {
			cred.Username = "root"
			cred.Password = "root"
			cred.PasswordSet = true
		}
	} else {
		cred.Username = args[0]
		if len(args) > 1 {
			cred.Password = args[1]
			cred.PasswordSet = true
		}
	}

	authenticator, err := auth.CreateAuthenticator(*mech, cred)
	if err != nil {
		log.Fatalf("unable to create authenticator: %v", err)
	}

	ctx := context.Background()

	connection, err := auth.Dial(ctx, authenticator, conn.Dial, conn.Endpoint(*host))
	if err != nil {
		log.Fatalf("failed connecting: %v", err)
	}

	cmd := msg.NewCommand(
		msg.NextRequestID(),
		*db,
		false,
		bson.D{{"count", *col}},
	)

	var result bson.D
	conn.ExecuteCommand(ctx, connection, cmd, &result)
	if err != nil {
		log.Fatalf("failed executing count command: %v", err)
	}

	log.Println(result)
}

func parseProps() (map[string]string, error) {
	if props == nil || *props == "" {
		return nil, nil
	}

	result := make(map[string]string)
	pairs := strings.Split(*props, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, ":", 2)
		if len(kv) != 2 || kv[0] == "" {
			return nil, fmt.Errorf("invalid authMechanism property")
		}
		result[kv[0]] = kv[1]
	}

	return result, nil
}
