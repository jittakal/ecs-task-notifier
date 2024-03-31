package internal

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"log/slog"
	"strconv"

	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// Algorithm
// Get List of Tasks for the ECS Service
// Iterate over Task List
// Get List of container-instance id's once
// Iterate over each task and its list of containers
// Match containerPort with InputPort
// Get Host - Private IP Address
// Get Host Port Address

// Protocol - Private IP address - Host Port - Notify URI
// http://private-ip-address:hostport/notify-uri

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
	ec2Client *ec2.Client
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
		withEc2Client(cfg).
		withSQSClient(cfg)

	return awsService, nil
}

func (awsService *AWSService) withEcsClient(cfg aws.Config) *AWSService {
	ecsClient := ecs.NewFromConfig(cfg)
	awsService.ecsClient = ecsClient
	return awsService
}

func (awsService *AWSService) withEc2Client(cfg aws.Config) *AWSService {
	ec2Client := ec2.NewFromConfig(cfg)
	awsService.ec2Client = ec2Client
	return awsService
}

func (awsService *AWSService) withSQSClient(cfg aws.Config) *AWSService {
	sqsClient := sqs.NewFromConfig(cfg)
	awsService.sqsClient = sqsClient
	return awsService
}

// Get EC2 Instance Proviate IP Address
func (awsService *AWSService) ec2PrivateAddress(ctx context.Context, instanceId string) (*string, error) {

	requestId := RequestIdFromContext(ctx)
	instances, err := awsService.ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceId},
	})
	if err != nil {
		slog.Error("failed to describe ec2 instances details", "requestId", requestId, "errorMessage", err)
		return nil, err
	}

	var privateIPs []string
	for _, reservation := range instances.Reservations {
		for _, instance := range reservation.Instances {
			// Assumed simple networking with only one private IP address
			// Changes might required as per netwroking
			for _, networkInterface := range instance.NetworkInterfaces {
				privateIPs = append(privateIPs, aws.ToString(networkInterface.PrivateIpAddress))
			}
		}
	}

	if len(privateIPs) == 0 {
		return nil, errors.New("private ip address not found")
	}
	return &privateIPs[0], nil
}

// Get List of Container Instances and its Private IP Addresses
func (awsService *AWSService) listContainerInstances(ctx context.Context, cluster string) (map[string]string, error) {

	requestId := RequestIdFromContext(ctx)
	listContainerInstancesInput := &ecs.ListContainerInstancesInput{
		Cluster: aws.String(cluster),
	}
	containerInstanceIpAddresses := make(map[string]string)

	paginator := ecs.NewListContainerInstancesPaginator(awsService.ecsClient, listContainerInstancesInput)
	for paginator.HasMorePages() {
		containerInstances, ciErr := paginator.NextPage(ctx)
		if ciErr != nil {
			slog.Error("failed to paginate list of container instances", "requestId", requestId, "errorMessage", ciErr)
			return nil, ciErr
		}

		containerInstanceDetails, cidErr := awsService.ecsClient.DescribeContainerInstances(ctx, &ecs.DescribeContainerInstancesInput{
			Cluster:            aws.String(cluster),
			ContainerInstances: containerInstances.ContainerInstanceArns,
		})
		if cidErr != nil {
			slog.Error("failed to describe container instance details", "requestId", requestId, "errorMessage", cidErr)
			return nil, cidErr
		}

		for _, instance := range containerInstanceDetails.ContainerInstances {
			// TODO Explore way of getting direct private IP Address
			privateAddress, err := awsService.ec2PrivateAddress(ctx, *instance.Ec2InstanceId)
			if err != nil {
				slog.Error("failed to get private IP address", "requestId", requestId, "errorMessage", err)
				// Continue with other container instances
				// TODO - Pending Error Handling on Missing Private IP Address
				// return nil, err
			} else {
				slog.Info("Received privateAddress", "requestId", requestId, "ipAddress", *privateAddress)
				containerInstanceIpAddresses[*instance.ContainerInstanceArn] = *privateAddress
			}
		}
	}
	return containerInstanceIpAddresses, nil
}

func (awsService *AWSService) DiscoverServiceTasks(ctx context.Context, serviceMessage *ServiceMessage) ([]*TaskNotifyMessage, error) {

	requestId := RequestIdFromContext(ctx)
	// container instance IP Addresses
	ciIPAddresses, ciIPAddressesErr := awsService.listContainerInstances(ctx, serviceMessage.Cluster)
	if ciIPAddressesErr != nil {
		return nil, ciIPAddressesErr
	}

	containerPort, containerPortErr := strconv.ParseInt(serviceMessage.NotifyMeContainerPort, 10, 32)
	if containerPortErr != nil {
		slog.Error("failed to parse container port from string to int", "requestId", requestId, "errorMessage", containerPortErr)
		return nil, containerPortErr
	}

	listTasksInput := &ecs.ListTasksInput{
		Cluster:     aws.String(serviceMessage.Cluster),
		ServiceName: aws.String(serviceMessage.Service),
	}

	paginator := ecs.NewListTasksPaginator(awsService.ecsClient, listTasksInput)
	var discoveredTasks []*TaskNotifyMessage

	for paginator.HasMorePages() {
		listTaskPage, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		descTaskOutput, descTaskErr := awsService.ecsClient.DescribeTasks(ctx, &ecs.DescribeTasksInput{
			Cluster: aws.String(serviceMessage.Cluster),
			Tasks:   listTaskPage.TaskArns,
		})

		if descTaskErr != nil {
			return nil, descTaskErr
		}

		for _, task := range descTaskOutput.Tasks {
			log.Printf("Task: %v", aws.ToString(task.TaskArn))
			// Task should be running and LaunchType is of Type EC2
			if aws.ToString(task.LastStatus) == string(types.DesiredStatusRunning) &&
				task.LaunchType == types.LaunchTypeEc2 {
				// Iterate over containers matching containerPort
				// Extract HostPort

				// task.ContainerInstanceArn
				for _, container := range task.Containers {
					if container.HealthStatus == types.HealthStatusHealthy &&
						aws.ToString(container.LastStatus) == string(types.DesiredStatusRunning) {
						for _, networkBinding := range container.NetworkBindings {
							if aws.ToInt32(networkBinding.ContainerPort) == int32(containerPort) {
								if ipAddress, ok := ciIPAddresses[*task.ContainerInstanceArn]; ok {
									taskNotifyMessage := NewTaskNotifyMessage()
									taskNotifyMessage.NotifyTaskArn = *task.TaskArn
									taskNotifyMessage.NotifyMeHostAddress = ipAddress
									taskNotifyMessage.NotifyMeHostPort = strconv.Itoa(int(aws.ToInt32(networkBinding.HostPort)))
									taskNotifyMessage.NotifyMeAPIUri = serviceMessage.NotifyMeAPIUri

									discoveredTasks = append(discoveredTasks, taskNotifyMessage)
								}
							}
						}
					}
				}
			}
		}

	}
	slog.Info("total number of tasks discovered", "lenght", len(discoveredTasks))

	return discoveredTasks, nil
}

// Publish ECS Service Task Messages to SQS for further processing
func (awsService *AWSService) PublishServiceMessage(ctx context.Context, sqsQueueURL string, taskNotifyMessage *TaskNotifyMessage) (*string, error) {

	requestId := RequestIdFromContext(ctx)
	slog.Info("Request to publish the message received", "requestId", requestId, "taskNotifyMessage", *taskNotifyMessage)

	msgJsonBytes, jsonMarshalErr := json.Marshal(taskNotifyMessage)
	if jsonMarshalErr != nil {
		slog.Error("failed to json.Marshal for taskNotifyMessage", "requestId", requestId, "errorMessage", jsonMarshalErr)
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
