package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/jittakal/ecs-task-notifier/ecs-service-discovery-lambda/internal"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func HandleRequest(ctx context.Context, event *events.SQSEvent) error {
	requestId := internal.RequestIdFromContext(ctx)

	awsService, err := internal.NewAWSService(ctx)
	if err != nil {
		return err
	}

	// Accessing environment variables
	// ecsClusterName := os.Getenv("ECS_CLUSTER_NAME")
	sqsQueueURL, keyNotExists := os.LookupEnv("SQS_QUEUE_URL")
	if !keyNotExists {
		slog.Error("Environment variable value is missing", "Key", "SQS_QUEUE_URL")
		return fmt.Errorf("environment key missing: %v", "SQS_QUEUE_URL")
	}

	for _, record := range event.Records {
		slog.Info("Received Message Details", "requestId", requestId, "messageId", record.MessageId, "messageBody", record.Body)

		var ecsNotifyMessage internal.EcsNotify
		// Unmarshal the JSON string into the Person struct
		err := json.Unmarshal([]byte(record.Body), &ecsNotifyMessage)
		if err != nil {
			slog.Error("Failed to Unmarshal Message to struct", "requestId", requestId, "errorMessage", err)
			return err
		}
		ecsClusterName := ecsNotifyMessage.Cluster

		services, listServiceErr := awsService.ListECSServices(ctx, ecsClusterName)
		if listServiceErr != nil {
			// TODO Add code block to check if ECS cluster exists
			// Check if the error is of type ClusterNotFoundException

			// var clusterNotFoundErr *types.ClusterNotFoundException
			// if errors.As(listServiceErr, &clusterNotFoundErr) {
			// Handle the specific error
			//	log.Printf("ECS cluster not found:", aws.ToString(clusterNotFoundErr.Message))
			//	return []*EcsService{}, nil
			// }
			return listServiceErr
		}
		slog.Info("Total number of services", "length", len(services))

		filteredServices, filterServiceErr := awsService.FilterECSServices(ctx, services)
		if filterServiceErr != nil {
			return filterServiceErr
		}
		slog.Info("Total number of filtered services", "length", len(filteredServices))

		for _, serviceMessage := range filteredServices {
			svcMsgId, publishErr := awsService.PublishServiceMessage(ctx, sqsQueueURL, serviceMessage)

			if publishErr != nil {
				return publishErr // put message on retry
			}
			slog.Info("Message published successfully", "requestId", requestId, "messageId", *svcMsgId)
		}
	}

	return nil
}

func main() {
	lambda.Start(HandleRequest)
}
