package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/jittakal/ecs-task-notifier/ecs-service-task-discovery-lambda/internal"
)

func HandleRequest(ctx context.Context, event *events.SQSEvent) error {
	requestId := internal.RequestIdFromContext(ctx)

	awsService, err := internal.NewAWSService(ctx)
	if err != nil {
		return err
	}

	// Accessing environment variables
	sqsQueueURL, keyNotExists := os.LookupEnv("SQS_QUEUE_URL")
	if !keyNotExists {
		slog.Error("Environment variable value is missing", "Key", "SQS_QUEUE_URL")
		return fmt.Errorf("environment key missing: %v", "SQS_QUEUE_URL")
	}

	for _, record := range event.Records {
		slog.Info("Received Message Details", "requestId", requestId, "messageId", record.MessageId, "messageBody", record.Body)

		var serviceMessage internal.ServiceMessage
		// Unmarshal the JSON string into the Person struct
		err := json.Unmarshal([]byte(record.Body), &serviceMessage)
		if err != nil {
			slog.Error("Failed to Unmarshal Message to struct", "requestId", requestId, "errorMessage", err)
			return err
		}
		slog.Info("ECS service details", "serviceName", serviceMessage.Service)

		taskNotifyMessages, discoverTaskErr := awsService.DiscoverServiceTasks(ctx, &serviceMessage)
		if discoverTaskErr != nil {
			return discoverTaskErr
		}

		for _, taskNotifyMessage := range taskNotifyMessages {
			taskMsgId, publishErr := awsService.PublishServiceMessage(ctx, sqsQueueURL, taskNotifyMessage)

			if publishErr != nil {
				return publishErr // put message on retry
			}
			slog.Info("Message published successfully", "requestId", requestId, "messageId", *taskMsgId)
		}
	}
	return nil
}

func main() {
	lambda.Start(HandleRequest)
}
