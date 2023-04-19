package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	// DefaultHTTPGetAddress Default Address
	DefaultHTTPGetAddress = "https://checkip.amazonaws.com"

	// ErrNoIP No IP found in response
	ErrNoIP = errors.New("No IP in HTTP response")

	// ErrNon200Response non 200 status code in response
	ErrNon200Response = errors.New("Non 200 Response found")
)

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var commandEventDuration int64
	var commandEventCount int64

	commandMonitor := &event.CommandMonitor{
		Succeeded: func(_ context.Context, e *event.CommandSucceededEvent) {
			commandEventDuration += e.DurationNanos
			commandEventCount++
		},
		Failed: func(_ context.Context, e *event.CommandFailedEvent) {
			commandEventDuration += e.DurationNanos
			commandEventCount++
		},
	}

	var heartbeatEventDuration int64
	var heartbeatEventcount int64

	serverMonitor := &event.ServerMonitor{
		ServerHeartbeatSucceeded: func(e *event.ServerHeartbeatSucceededEvent) {
			heartbeatEventDuration += e.DurationNanos
			heartbeatEventcount++
		},
		ServerHeartbeatFailed: func(e *event.ServerHeartbeatFailedEvent) {
			heartbeatEventDuration += e.DurationNanos
			heartbeatEventcount++
		},
	}

	var openConnections int

	poolMonitor := &event.PoolMonitor{
		Event: func(e *event.PoolEvent) {
			switch e.Type {
			case event.ConnectionCreated:
				openConnections++
			case event.ConnectionClosed:
				openConnections--
			}
		},
	}

	clientOptions := options.Client().ApplyURI(os.Getenv("MONGODB_URI")).
		SetMonitor(commandMonitor).SetServerMonitor(serverMonitor).
		SetPoolMonitor(poolMonitor)

	// Create a MongoClient that points to MONGODB_URI.
	client, err := mongo.NewClient(clientOptions)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(10*time.Second))
	defer cancel()

	// Connect to the client.
	if err = client.Connect(ctx); err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	defer client.Disconnect(ctx)

	collection := client.Database("faas").Collection("lambda")
	doc := map[string]string{"hello": "world"}

	_, err = collection.InsertOne(ctx, doc)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	_, err = collection.DeleteOne(ctx, doc)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	var averageCommandDuration float64
	if commandEventCount > 0 {
		averageCommandDuration = float64(commandEventDuration) / float64(commandEventCount)
	}

	var averageHeartbeatDuration float64
	if heartbeatEventcount > 0 {
		averageHeartbeatDuration = float64(heartbeatEventDuration) / float64(heartbeatEventcount)
	}

	response := map[string]interface{}{
		"averageCommandDuration":   averageCommandDuration,
		"averageHeartbeatDuration": averageHeartbeatDuration,
		"openConnections":          openConnections,
		"heartbeatCount":           heartbeatEventcount,
	}

	jsonResponse, err := json.Marshal(response)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return events.APIGatewayProxyResponse{
		Body:       string(jsonResponse),
		StatusCode: 200,
	}, nil
}

func main() {
	lambda.Start(handler)
}
