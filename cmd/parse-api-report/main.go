// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
    "bufio"
    "fmt"
    "log"
    "os"
    "strings"
)

func main() {
	var line string
	var suppress bool
	var found_change bool = false
	var found_summary bool = false

    // open file to read
    f_read, err := os.Open("api-report.txt")
    if err != nil {
        log.Fatal(err)
    }
    // remember to close the file at the end of the program
    defer f_read.Close()

    // open file to write
    f_write, err := os.Create("api-report.md")
    if err != nil {
        log.Fatal(err)
    }
    // remember to close the file at the end of the program
    defer f_write.Close()

    fmt.Fprint(f_write, "## API Change Report\n")

    // read the file line by line using scanner
    scanner := bufio.NewScanner(f_read)

    for scanner.Scan() {
        // do something with a line
        line = scanner.Text()
        if strings.Index(line, "## ") == 0 {
        	line = "##" + line
        }

        if strings.Contains(line, "/mongo/integration/") {
        	suppress = true
        }
        if strings.Index(line, "# summary") == 0 {
        	suppress = true
        	found_summary = true
        }

        if strings.Contains(line, "go.mongodb.org/mongo-driver") {
        	line = strings.Replace(line, "go.mongodb.org/mongo-driver", ".", -1)
        	line = "##" + line
        }
        if !suppress {
        	fmt.Fprint(f_write, "%s\n", line)
        	found_change = true
        }
        if len(line) == 0 {
        	suppress = false
        }
    }

    if !found_change {
    	fmt.Fprint(f_write, "No changes found!\n")
    }

    if !found_summary {
    	log.Fatal("Could not parse api summary")
    }

    if err := scanner.Err(); err != nil {
        log.Fatal(err)
    }

}

/*
Empty report

# summary
Base version: v1.13.0-prerelease.0.20230801162915-143066290584 (14306629058485de0736f05e970a56a4a5dc9c43)
Cannot suggest a release version.
Can only suggest a release version when compared against the most recent version of this major: v1.13.0-prerelease.


Fuller report

# go.mongodb.org/mongo-driver/bson
## compatible changes
RawValue.IsZero: added

# go.mongodb.org/mongo-driver/bson/bsontype
## compatible changes
Type.IsValid: added

# go.mongodb.org/mongo-driver/event
## compatible changes
CommandFailedEvent.DatabaseName: added
CommandFinishedEvent.DatabaseName: added
CommandSucceededEvent.DatabaseName: added

# go.mongodb.org/mongo-driver/mongo
## compatible changes
(*Cursor).SetComment: added
(*Cursor).SetMaxTime: added

# go.mongodb.org/mongo-driver/mongo/description
## compatible changes
SelectedServer.SessionTimeoutMinutesPtr: added
Server.SessionTimeoutMinutesPtr: added
Topology.SessionTimeoutMinutesPtr: added

# go.mongodb.org/mongo-driver/mongo/integration/unified
## incompatible changes
ParseTestFile: changed from func(*testing.T, []byte, ...*Options) ([]go.mongodb.org/mongo-driver/mongo/integration/mtest.RunOnBlock, []*TestCase) to func(*testing.T, []byte, bool, ...*Options) ([]go.mongodb.org/mongo-driver/mongo/integration/mtest.RunOnBlock, []*TestCase)
commandMonitoringEvent.CommandFailedEvent: changed from *struct{CommandName *string "bson:\"commandName\""; HasServerConnectionID *bool "bson:\"hasServerConnectionId\""; HasServiceID *bool "bson:\"hasServiceId\""} to *struct{CommandName *string "bson:\"commandName\""; DatabaseName *string "bson:\"databaseName\""; HasServerConnectionID *bool "bson:\"hasServerConnectionId\""; HasServiceID *bool "bson:\"hasServiceId\""}
commandMonitoringEvent.CommandSucceededEvent: changed from *struct{CommandName *string "bson:\"commandName\""; Reply go.mongodb.org/mongo-driver/bson.Raw "bson:\"reply\""; HasServerConnectionID *bool "bson:\"hasServerConnectionId\""; HasServiceID *bool "bson:\"hasServiceId\""} to *struct{CommandName *string "bson:\"commandName\""; DatabaseName *string "bson:\"databaseName\""; Reply go.mongodb.org/mongo-driver/bson.Raw "bson:\"reply\""; HasServerConnectionID *bool "bson:\"hasServerConnectionId\""; HasServiceID *bool "bson:\"hasServiceId\""}
## compatible changes
clientLogMessages.IgnoreMessages: added

# go.mongodb.org/mongo-driver/x/mongo/driver
## compatible changes
(*BatchCursor).SetComment: added
(*BatchCursor).SetMaxTime: added
(*ListCollectionsBatchCursor).SetComment: added
(*ListCollectionsBatchCursor).SetMaxTime: added
CursorOptions.MarshalValueEncoderFn: added
ErrNoCursor: added
LegacyNotPrimaryErrMsg: added

# go.mongodb.org/mongo-driver/x/mongo/driver/connstring
## compatible changes
ErrLoadBalancedWithDirectConnection: added
ErrLoadBalancedWithMultipleHosts: added
ErrLoadBalancedWithReplicaSet: added
ErrSRVMaxHostsWithLoadBalanced: added
ErrSRVMaxHostsWithReplicaSet: added

# go.mongodb.org/mongo-driver/x/mongo/driver/session
## incompatible changes
(*Client).UnpinConnection: removed

# summary
Cannot suggest a release version.
Incompatible changes were detected.
*/