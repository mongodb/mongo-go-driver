package driverx

import "go.mongodb.org/mongo-driver/x/mongo/driver/session"

type KillCursorsResult struct {
	CursorsKilled   []int64
	CursorsNotFound []int64
	CursorsAlive    []int64
	Legacy          bool
}

type KillCursorsOperation struct {
	id []int64
	ns Namespace

	clock  *session.ClusterClock
	client *session.Client

	result KillCursorsResult
}
