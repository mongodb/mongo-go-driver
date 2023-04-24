package main

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

//// metadata supports command, heartbeat, and pool event handlers to record
//// event durations, as well as the number of heartbeats, commands, and open
//// conections.
//type metadata struct {
//	commandCount      int
//	commandDuration   int64
//	heartbeatCount    int
//	heartbeatDuration int64
//	openConnections   int
//}
//
//// averageCommandDurationNS calculates the average time (ns) required for a
//// command (success or failure) to execute. If no commands have been executed,
//// return 0.
//func (md *metadata) averageCommandDurationNS() float64 {
//	if md.commandCount == 0 {
//		return 0
//	}
//
//	return float64(md.commandDuration) / float64(md.commandCount)
//}
//
//// averageHeartbeatDurationNS calculates the average time (ns) between
//// heartbeats (success or failure). If no heartbeats have occured, return 0.
//func (md *metadata) averageHeartbeatDuration() float64 {
//	if md.heartbeatCount == 0 {
//		return 0
//	}
//
//	return float64(md.heartbeatDuration) / float64(md.heartbeatCount)
//}
//
//// commandSucceeded is an "event#CommandMonitor.Succeeded" function for counting
//// the	number of successful command events and recording a running duration of
//// all commands.
//func (md *metadata) commandSucceeded(_ context.Context, e *event.CommandSucceededEvent) {
//	md.commandCount++
//	md.commandDuration += e.DurationNanos
//}
//
//// commandFailed is an "event#CommandMonitor.Failed" function for counting the
//// number of failed events and recording a running duration of all commands.
//func (md *metadata) commandFailed(_ context.Context, e *event.CommandFailedEvent) {
//	md.commandCount++
//	md.commandDuration += e.DurationNanos
//}
//
//// poolEvent is an "event#PoolEvent.Event" function for incrementing on created
//// connections and decrementing on closed connections.
//func (md *metadata) poolEvent(e *event.PoolEvent) {
//	switch e.Type {
//	case event.ConnectionCreated:
//		md.openConnections++
//	case event.ConnectionClosed:
//		md.openConnections--
//	}
//}
//
//// serverHeartbeatSucceeded is an "event#ServerMonitor.ServerHeartbeatSucceeded"
//// function for counting the number of heartbeat events and recording a running
//// duration of all server events.
//func (md *metadata) serverHeartbeatSucceeded(e *event.ServerHeartbeatSucceededEvent) {
//	md.heartbeatCount++
//	md.heartbeatDuration += e.DurationNanos
//}
//
//// serverHeartbeatFailed is an "event#ServerMonitor.ServerHeartbeatFailed"
//// function for counting the number of heartbeat events and recording a running
//// duration of all server events.
//func (md *metadata) serverHeartbeatFailed(e *event.ServerHeartbeatFailedEvent) {
//	md.heartbeatCount++
//	md.heartbeatDuration += e.DurationNanos
//}
//
//// response is the data we return in the body of the API Gateway response.
//type response struct {
//	AvgCommandDuration   float64 `json:"averageCommandDuration"`
//	AvgHeartbeatDuration float64 `json:"averageHeartbeatDuration"`
//	OpenConnections      int     `json:"openConnections"`
//	HeartbeatCount       int     `json:"heartbeatCount"`
//}
//
//func newResponse(md *metadata) *response {
//	rsp := &response{
//		AvgCommandDuration:   md.averageCommandDurationNS(),
//		AvgHeartbeatDuration: md.averageHeartbeatDuration(),
//		OpenConnections:      md.openConnections,
//		HeartbeatCount:       md.heartbeatCount,
//	}
//
//	return rsp
//}
//
//// gateway500 is a convenience function for constructing a gateway response with
//// a 500 status code, indicating an internal server
//// error.
//func gateway500() events.APIGatewayProxyResponse {
//	return events.APIGatewayProxyResponse{
//		StatusCode: http.StatusInternalServerError,
//		Body:       http.StatusText(http.StatusInternalServerError),
//	}
//
//}
//
//// handler is the AWS Lambda handler, executing at runtime.
//func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
//	// Construct a new metadata object and apply it's event function to the
//	// event monitors.
//	md := new(metadata)
//
//	commandMonitor := &event.CommandMonitor{
//		Succeeded: md.commandSucceeded,
//		Failed:    md.commandFailed,
//	}
//
//	serverMonitor := &event.ServerMonitor{
//		ServerHeartbeatSucceeded: md.serverHeartbeatSucceeded,
//		ServerHeartbeatFailed:    md.serverHeartbeatFailed,
//	}
//
//	poolMonitor := &event.PoolMonitor{
//		Event: md.poolEvent,
//	}
//
//	clientOptions := options.Client().ApplyURI(os.Getenv("MONGODB_URI")).
//		SetMonitor(commandMonitor).SetServerMonitor(serverMonitor).
//		SetPoolMonitor(poolMonitor)
//
//	// Create a MongoClient that points to MONGODB_URI and listens to the
//	// ComandMonitor, ServerMonitor, and PoolMonitor events.
//	client, err := mongo.NewClient(clientOptions)
//	if err != nil {
//		return gateway500(), fmt.Errorf("failed to create client: %w", err)
//	}
//
//	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
//	defer cancel()
//
//	// Attempt to connect to the client with a 5 second timeout.
//	if err = client.Connect(ctx); err != nil {
//		return gateway500(), fmt.Errorf("failed to connect: %w", err)
//	}
//
//	defer client.Disconnect(ctx)
//
//	collection := client.Database("faas").Collection("lambda")
//
//	// Create a document to insert for the automated test.
//	doc := map[string]string{"hello": "world"}
//
//	// Insert the document.
//	_, err = collection.InsertOne(ctx, doc)
//	if err != nil {
//		return gateway500(), fmt.Errorf("failed to insert: %w", err)
//	}
//
//	// Delete the document.
//	_, err = collection.DeleteOne(ctx, doc)
//	if err != nil {
//		return gateway500(), fmt.Errorf("failed to delete: %w", err)
//	}
//
//	body, err := json.Marshal(newResponse(md))
//	if err != nil {
//		return gateway500(), fmt.Errorf("failed to marshal response JSON: %w", err)
//	}
//
//	return events.APIGatewayProxyResponse{
//		Body:       string(body),
//		StatusCode: 200,
//	}, nil
//}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		Body:       string(`{"hello":"world"}`),
		StatusCode: 200,
	}, nil
}

func main() {
	lambda.Start(handler)
}
