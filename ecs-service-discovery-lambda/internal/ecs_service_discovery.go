package internal

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// Get AWSRequestId from Lambda Context Object
func RequestIdFromContext(ctx context.Context) string {
	var requestId string = "x"
	if lc, ok := lambdacontext.FromContext(ctx); ok {
		requestId = lc.AwsRequestID
	}
	return requestId
}

type AWSService struct {
	ecsClient *ecs.Client
	sqsClient *sqs.Client
}

func NewAWSService(ctx context.Context) (*AWSService, error) {
	requestId := RequestIdFromContext(ctx)
	awsService := &AWSService{}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		slog.Error("Failed to load default config", "requestId", requestId, "errorMessage", err)
		return nil, err
	}

	awsService = awsService.withEcsClient(cfg).
		withSQSClient(cfg)

	return awsService, nil
}

func (awsService *AWSService) withEcsClient(cfg aws.Config) *AWSService {
	ecsClient := ecs.NewFromConfig(cfg)
	awsService.ecsClient = ecsClient
	return awsService
}

func (awsService *AWSService) withSQSClient(cfg aws.Config) *AWSService {
	sqsClient := sqs.NewFromConfig(cfg)
	awsService.sqsClient = sqsClient
	return awsService
}

// List All the ECS Services running within ECS Cluster
func (awsService *AWSService) ListECSServices(ctx context.Context, cluster string) ([]*EcsService, error) {
	requestId := RequestIdFromContext(ctx)

	// Initialize variables for pagination
	var nextToken *string
	var allServices []types.Service

	// Paginate through ECS cluster services
	for {

		respListSvcs, errListSvcs := awsService.ecsClient.ListServices(ctx, &ecs.ListServicesInput{
			Cluster:   aws.String(cluster),
			NextToken: nextToken,
		})
		if errListSvcs != nil {
			slog.Error("Failed to list ECS cluster services", "requestId", requestId, "errorMessage", errListSvcs)
			return nil, errListSvcs
		}

		// Describe services for the cluster with pagination token
		respServices, errServices := awsService.ecsClient.DescribeServices(ctx, &ecs.DescribeServicesInput{
			Services: respListSvcs.ServiceArns,
			Cluster:  aws.String(cluster),
		})
		if errServices != nil {
			slog.Error("Failed to describe ECS cluster services", "requestId", requestId, "errorMessage", errServices)
			return nil, errServices
		}

		// Append services to the list
		allServices = append(allServices, respServices.Services...)

		// Check if there are more services to fetch
		if respListSvcs.NextToken == nil {
			break
		}

		nextToken = respListSvcs.NextToken
	}

	var ecsServices []*EcsService
	for _, service := range allServices {
		ecsServices = append(ecsServices, &EcsService{
			Cluster:        cluster,
			Service:        aws.ToString(service.ServiceName),
			TaskDefinition: aws.ToString(service.TaskDefinition),
		})
	}

	return ecsServices, nil
}

// Filter ECS Services latest TaskDefinition matching required dockerlabels
func (awsService *AWSService) FilterECSServices(ctx context.Context, services []*EcsService) ([]*ServiceMessage, error) {
	requestId := RequestIdFromContext(ctx)

	var filteredServices []*ServiceMessage
	for _, service := range services {
		taskDefinition, err := awsService.ecsClient.DescribeTaskDefinition(ctx, &ecs.DescribeTaskDefinitionInput{
			TaskDefinition: aws.String(service.TaskDefinition),
		})
		if err != nil {
			slog.Error("Failed to describe task definition", "requestId", requestId, "errorMessage", err)
			return nil, err
		}

		for _, containerDefinition := range taskDefinition.TaskDefinition.ContainerDefinitions {
			// NOTIFY_ME_CONTAINER_PORT = 8080
			// NOTIFY_ME_API_URI = /v1.0/notify

			dockerLabels := containerDefinition.DockerLabels
			nmcPort, nmcPortOk := dockerLabels["NOTIFY_ME_CONTAINER_PORT"]
			nmApiUri, nmApiUriOk := dockerLabels["NOTIFY_ME_API_URI"]

			// Check if Docker Label Exisits for above two keys
			if nmcPortOk && nmApiUriOk {
				ecsService := NewServiceMessage()
				ecsService.Cluster = service.Cluster
				ecsService.Service = service.Service
				ecsService.NotifyMeContainerPort = nmcPort
				ecsService.NotifyMeAPIUri = nmApiUri

				filteredServices = append(filteredServices, ecsService)
				break // found the match
			}
		}
	}

	return filteredServices, nil
}

// Publish ECS Service Messages to SQS for further processing
func (awsService *AWSService) PublishServiceMessage(ctx context.Context, sqsQueueURL string, serviceMessage *ServiceMessage) (*string, error) {

	requestId := RequestIdFromContext(ctx)
	slog.Info("Request to publish the message received", "requestId", requestId, "serviceMessage", *serviceMessage)

	msgJsonBytes, jsonMarshalErr := json.Marshal(serviceMessage)
	if jsonMarshalErr != nil {
		slog.Error("failed to json.Marshal for serviceMessage", "requestId", requestId, "errorMessage", jsonMarshalErr)
		return nil, jsonMarshalErr
	}

	sendMsgOutput, sendMsgErr := awsService.sqsClient.SendMessage(ctx, &sqs.SendMessageInput{
		MessageBody: aws.String(string(msgJsonBytes)),
		QueueUrl:    aws.String(sqsQueueURL),
	})

	if sendMsgErr != nil {
		slog.Error("failed to pushlish message to SQS", "requestId", requestId, "errorMessage", sendMsgErr)
		return nil, sendMsgErr
	}

	return sendMsgOutput.MessageId, nil
}
