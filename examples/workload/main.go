package main

import (
	"context"
	"flag"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"time"

	"gopkg.in/mgo.v2/bson"

	"github.com/10gen/mongo-go-driver/cluster"
	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/msg"
	"github.com/10gen/mongo-go-driver/ops"
	"github.com/10gen/mongo-go-driver/readpref"
)

var concurrency = flag.Int("concurrency", 24, "how much concurrency should be used")
var ns = flag.String("namespace", "test.foo", "the namespace to use for test data")

var n = rand.New(rand.NewSource(time.Now().Unix()))

func main() {

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt)

	c, err := cluster.New()
	if err != nil {
		log.Fatalf("unable to create cluster: %s", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	log.Println("prepping")
	err = prep(ctx, c)
	if err != nil {
		log.Fatalf("unable to prep: %s", err)
	}
	log.Println("done prepping")

	log.Println("working")
	for i := 0; i < *concurrency; i++ {
		go work(ctx, i, c)
	}

	<-done
	log.Println("interupt received: shutting down")
	cancel()
	c.Close()
	log.Println("finished")
}

func prep(ctx context.Context, c *cluster.Cluster) error {

	var docs []bson.D
	for i := 0; i < 1000; i++ {
		docs = append(docs, bson.D{{"_id", i}})
	}

	ns := ops.ParseNamespace(*ns)
	deleteCommand := bson.D{
		{"delete", ns.Collection},
		{"deletes", []bson.D{
			bson.D{
				{"q", bson.M{}},
				{"limit", 0},
			},
		}},
	}
	deleteRequest := msg.NewCommand(
		msg.NextRequestID(),
		ns.DB,
		false,
		deleteCommand,
	)
	insertCommand := bson.D{
		{"insert", ns.Collection},
		{"documents", docs},
	}
	insertRequest := msg.NewCommand(
		msg.NextRequestID(),
		ns.DB,
		false,
		insertCommand,
	)

	s, err := c.SelectServer(ctx, cluster.WriteSelector())
	if err != nil {
		return err
	}

	connection, err := s.Connection(ctx)
	if err != nil {
		return err
	}
	defer connection.Close()

	return conn.ExecuteCommands(ctx, connection, []msg.Request{deleteRequest, insertRequest}, []interface{}{&bson.D{}, &bson.D{}})
}

func work(ctx context.Context, idx int, c *cluster.Cluster) {
	ns := ops.ParseNamespace(*ns)
	rp := readpref.Nearest()
	for {
		select {
		case <-ctx.Done():
		default:

			limit := n.Intn(999) + 1

			s, err := c.SelectServer(ctx, cluster.ReadPrefSelector(rp))
			if err != nil {
				log.Printf("%d-failed selecting a server: %s", idx, err)
				continue
			}

			pipeline := []bson.D{
				bson.D{{"$limit", limit}},
			}

			cursor, err := ops.Aggregate(ctx, &ops.SelectedServer{s, rp}, ns, pipeline, ops.AggregationOptions{
				BatchSize: 200,
			})
			if err != nil {
				log.Printf("%d-failed executing aggregate: %s", idx, err)
				continue
			}
			defer cursor.Close(ctx)

			count := 0
			var result bson.D
			for cursor.Next(ctx, &result) {
				count++
				// just iterate this guy...
			}
			if cursor.Err() != nil {
				log.Printf("%d-failed iterating aggregate results: %s", idx, err)
				continue
			}

			log.Printf("%d-iterated %d docs", idx, count)
		}
	}
}
