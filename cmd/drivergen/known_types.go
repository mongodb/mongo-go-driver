package main

import (
	"fmt"
	"go/types"

	"golang.org/x/tools/go/packages"
)

var Deployment types.Type
var ServerSelector types.Type
var Namespace types.Type
var WriteConcern types.Type
var ReadConcern types.Type
var ReadPreference types.Type
var ClientSession types.Type
var ClusterClock types.Type

type KnownType uint

const (
	KTUnknown KnownType = iota
	KTDeployment
	KTServerSelector
	KTNamespace
	KTWriteConcern
	KTReadConcern
	KTReadPreference
	KTClientSession
	KTClusterClock
)

var knownPackages = [...]string{
	"go.mongodb.org/mongo-driver/x/mongo/driverx",
	"go.mongodb.org/mongo-driver/mongo/writeconcern",
	"go.mongodb.org/mongo-driver/mongo/readconcern",
	"go.mongodb.org/mongo-driver/mongo/readpref",
	"go.mongodb.org/mongo-driver/x/mongo/driver/session",
	"go.mongodb.org/mongo-driver/x/network/description",
	".",
}

func LoadKnownTypes(pkgs []*packages.Package) error {
	var driverxPkg, writeconcernPkg, readconcernPkg, readprefPkg, sessionPkg, descriptionPkg *packages.Package
	var err error

	for _, pkg := range pkgs {
		switch pkg.Name {
		case "driverx":
			driverxPkg = pkg
		case "writeconcern":
			writeconcernPkg = pkg
		case "readconcern":
			readconcernPkg = pkg
		case "readpref":
			readprefPkg = pkg
		case "session":
			sessionPkg = pkg
		case "description":
			descriptionPkg = pkg
		}
	}

	if driverxPkg == nil {
		return fmt.Errorf("couldn't load package %s", knownPackages[0])
	}

	Deployment, err = loadType("Deployment", driverxPkg)
	if err != nil {
		return err
	}

	Namespace, err = loadType("Namespace", driverxPkg)
	if err != nil {
		return err
	}

	if writeconcernPkg == nil {
		return fmt.Errorf("couldn't load package %s", knownPackages[1])
	}

	WriteConcern, err = loadType("WriteConcern", writeconcernPkg)
	if err != nil {
		return err
	}
	// WriteConcern = types.NewPointer(WriteConcern) // We always use *writeconcern.WriteConcern

	if readconcernPkg == nil {
		return fmt.Errorf("couldn't load package %s", knownPackages[2])
	}

	ReadConcern, err = loadType("ReadConcern", readconcernPkg)
	if err != nil {
		return err
	}
	// ReadConcern = types.NewPointer(ReadConcern) // We always use *readconcern.ReadConcern

	if readprefPkg == nil {
		return fmt.Errorf("couldn't load package %s", knownPackages[3])
	}

	ReadPreference, err = loadType("ReadPref", readprefPkg)
	if err != nil {
		return err
	}
	// ReadPreference = types.NewPointer(ReadPreference) // We always use *readpref.ReadPref

	if sessionPkg == nil {
		return fmt.Errorf("couldn't load package %s", knownPackages[4])
	}

	ClientSession, err = loadType("Client", sessionPkg)
	if err != nil {
		return err
	}
	// ClientSession = types.NewPointer(ClientSession) // We always use *session.Client

	ClusterClock, err = loadType("ClusterClock", sessionPkg)
	if err != nil {
		return err
	}
	// ClusterClock = types.NewPointer(ClusterClock) // We always use *session.ClusterClock

	if descriptionPkg == nil {
		return fmt.Errorf("couldn't load package %s", knownPackages[5])
	}

	ServerSelector, err = loadType("ServerSelector", descriptionPkg)
	if err != nil {
		return err
	}

	return nil
}

func loadType(name string, pkg *packages.Package) (types.Type, error) {
	t := pkg.Types.Scope().Lookup(name)
	if t == nil {
		return nil, fmt.Errorf("%s not found in package %s", name, pkg)
	}
	return t.Type(), nil
}
