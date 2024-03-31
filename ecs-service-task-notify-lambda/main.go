package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/jittakal/ecs-task-notifier/ecs-service-task-notify-lambda/internal"
)

// Get AWSRequestId from Lambda Context Object
func requestIdFromContext(ctx context.Context) string {
	var requestId string = "x"
	if lc, ok := lambdacontext.FromContext(ctx); ok {
		requestId = lc.AwsRequestID
	}
	return requestId
}

// HandleRequest processes SQS messages and triggers HTTP GET requests
func HandleRequest(ctx context.Context, event *events.SQSEvent) error {
	requestId := requestIdFromContext(ctx)

	for _, record := range event.Records {
		slog.Info("Received Message Details", "requestId", requestId, "messageId", record.MessageId, "messageBody", record.Body)

		var tnm internal.TaskNotifyMessage
		// Unmarshal the JSON string into the TaskNotifyMessage struct
		err := json.Unmarshal([]byte(record.Body), &tnm)
		if err != nil {
			slog.Error("failed to Unmarshal Message to struct", "requestId", requestId, "errorMessage", err)
			return err
		}
		// Make an HTTP GET request
		// Format the URL with placeholders for host, port, and API URI
		// TODO Add Support for https protocol
		// TODO Add Support for pass event message to Notify API
		url := fmt.Sprintf("http://%s:%s%s", tnm.NotifyMeHostAddress,
			tnm.NotifyMeHostPort, tnm.NotifyMeAPIUri)
		slog.Info("notify API formed URL", "requestId", requestId, "URL", url)

		resp, err := http.Get(url)
		if err != nil {
			slog.Error("failed to trigger GET call", "requestId", requestId, "errorMessage", err)
			return err
		}
		defer resp.Body.Close()

		slog.Info("notify API response status code", "requestId", requestId, "statusCode", resp.StatusCode)
		// check status of http GET call
		if resp.StatusCode != 200 {
			return errors.New("failed to trigger Notify API")
		}
	}
	return nil
}

func main() {
	lambda.Start(HandleRequest)
}
