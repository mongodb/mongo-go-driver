package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func gateway500() events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       http.StatusText(http.StatusInternalServerError),
	}

}

//const localconn = "mongodb://localhost:27017"

const localconn = "mongodb://host.docker.internal:27017"

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	clientOptions := options.Client().ApplyURI(localconn)

	// Create a MongoClient that points to MONGODB_URI and listens to the
	// ComandMonitor, ServerMonitor, and PoolMonitor events.
	client, err := mongo.NewClient(clientOptions)
	if err != nil {
		return gateway500(), fmt.Errorf("failed to create client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	fmt.Println("client about to connect")

	// Attempt to connect to the client with a 5 second timeout.
	if err = client.Connect(ctx); err != nil {
		return gateway500(), fmt.Errorf("failed to connect: %w", err)
	}

	fmt.Println("client connected...")

	defer client.Disconnect(ctx)

	fmt.Println("ping about to happen")

	//collection := client.Database("x").Collection("y")
	//if _, err := collection.InsertOne(ctx, map[string]interface{}{"x": 1}); err != nil {
	//	panic(err)
	//}

	if err = client.Ping(ctx, nil); err != nil {
		return gateway500(), fmt.Errorf("failed to ping the server: %w", err)
	}

	fmt.Println("ping happened")

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
	}, nil
}

func main() {
	lambda.Start(handler)
}
