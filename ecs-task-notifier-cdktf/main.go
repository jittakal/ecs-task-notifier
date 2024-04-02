package main

import (
	"os"
	"path"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
	"github.com/hashicorp/terraform-cdk-go/cdktf"
	"github.com/jittakal/ecs-task-notifier/ecs-task-notifier-cdktf/generated/hashicorp/aws/iamrole"
	"github.com/jittakal/ecs-task-notifier/ecs-task-notifier-cdktf/generated/hashicorp/aws/iamrolepolicy"
	"github.com/jittakal/ecs-task-notifier/ecs-task-notifier-cdktf/generated/hashicorp/aws/lambdaeventsourcemapping"
	"github.com/jittakal/ecs-task-notifier/ecs-task-notifier-cdktf/generated/hashicorp/aws/lambdafunction"
	"github.com/jittakal/ecs-task-notifier/ecs-task-notifier-cdktf/generated/hashicorp/aws/s3bucket"
	"github.com/jittakal/ecs-task-notifier/ecs-task-notifier-cdktf/generated/hashicorp/aws/s3bucketobject"
	"github.com/jittakal/ecs-task-notifier/ecs-task-notifier-cdktf/generated/hashicorp/aws/sqsqueue"

	awsprovider "github.com/cdktf/cdktf-provider-aws-go/aws/v10/provider"
)

const (
	// To be change as per needs
	awsRegion = "us-east-1"

	// To be change as per needs
	_awsVpcPrivateSubnetId1   = "subnet-xxxxxxx"
	_awsVpcPrivateSubnetId2   = "subnet-xxxxxxx"
	_awsLambdaSecurityGroupId = "sg-xxxxxxx"

	lambdaZipBucketName = "ecs-task-notifier-lambdas"

	// To be change as per tests and timeout needs
	lambdaTimeout = 10.0

	ecsServiceNotificationQueueName = "ecs-service-notification"
	ecsServiceQueueName             = "ecs-services"
	ecsServiceTaskQueueName         = "ecs-service-tasks"
)

func NewMyStack(scope constructs.Construct, id string) cdktf.TerraformStack {
	stack := cdktf.NewTerraformStack(scope, &id)

	// AWS Provider
	awsprovider.NewAwsProvider(stack, jsii.String("AWS"), &awsprovider.AwsProviderConfig{
		Region: jsii.String(awsRegion),
	})

	// Terraform Stack Input Variables
	awsVpcPrivateSubnetId1 := cdktf.NewTerraformVariable(stack, jsii.String("awsVpcPrivateSubnetId1"), &cdktf.TerraformVariableConfig{
		Type:        jsii.String("string"),
		Default:     jsii.String(_awsVpcPrivateSubnetId1),
		Description: jsii.String("VPC Private Subnet Id 1"),
	})

	awsVpcPrivateSubnetId2 := cdktf.NewTerraformVariable(stack, jsii.String("awsVpcPrivateSubnetId2"), &cdktf.TerraformVariableConfig{
		Type:        jsii.String("string"),
		Default:     jsii.String(_awsVpcPrivateSubnetId2),
		Description: jsii.String("VPC Private Subnet Id 2"),
	})

	awsLambdaSecurityGroupId := cdktf.NewTerraformVariable(stack, jsii.String("awsLambdaSecurityGroupId"), &cdktf.TerraformVariableConfig{
		Type:        jsii.String("string"),
		Default:     jsii.String(_awsLambdaSecurityGroupId),
		Description: jsii.String("Lambda Function Security Group"),
	})

	// S3 bucket for lambda archive files
	bucket := s3bucket.NewS3Bucket(stack, jsii.String("ecs_task_notifier_lambda_bucket"), &s3bucket.S3BucketConfig{
		Bucket: jsii.String(lambdaZipBucketName + "-" + awsRegion),
	})
	cwd, _ := os.Getwd()

	// TODO Restrict resource permission to speific resource identified by ARN
	// instead of Resource = "*"
	// TODO Define separate IAM roles for lambda as on required permissions

	// IAM policies for Lambda Excecution
	lambdaRolePolicy := `
	{
		"Version": "2012-10-17",
		"Statement": [
		  {
			"Action": "sts:AssumeRole",
			"Principal": {
			  "Service": "lambda.amazonaws.com"
			},
			"Effect": "Allow",
			"Sid": "LambdaExecutionRole"
		  }
		]
	}`

	// IAM Policies related to ECS
	ecsServicePolicy := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Sid": "ECSDescribeServicePolicy",
				"Effect": "Allow",
				"Action": [
					"ecs:ListServices",
					"ecs:DescribeServices",
					"ecs:ListTasks",
					"ecs:DescribeTasks",
					"ecs:DescribeTaskDefinition",
					"ecs:ListContainerInstances",
					"ecs:DescribeContainerInstances"
				],
				"Resource": "*"
			}
		]
	}`

	// IAM Policies related to EC2
	ec2ServicePolicy := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Sid": "EC2DescribeServicePolicy",
				"Effect": "Allow",
				"Action": [
					"ec2:CreateNetworkInterface",
					"ec2:DeleteNetworkInterface",
					"ec2:DescribeNetworkInterfaces",
					"ec2:DescribeInstances",
					"ec2:CreateTags",
					"ec2:DeleteTags",
					"ec2:AssignPrivateIpAddresses",
					"ec2:UnassignPrivateIpAddresses"
				],
				"Resource": "*"
			}
		]
	}`

	// IAM Policies related to SQS
	sqsServicePolicy := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Sid": "SQSServicePolicy",
				"Effect": "Allow",
				"Action": [
					"sqs:ReceiveMessage",
					"sqs:DeleteMessage",
					"sqs:GetQueueAttributes",
					"sqs:SendMessage"
				],
				"Resource": "*"
			}
		]
	}`

	// IAM policies related to cloudwatch log
	cloudWatchLogServicePolicy := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Sid": "CloudWatchLogServicePolicy",
				"Effect": "Allow",
				"Action": [
					"logs:CreateLogGroup",
					"logs:CreateLogStream",
					"logs:PutLogEvents"
				],
				"Resource": "*"
			}
		]
	}`

	// SQS Queue - ECS Notification - Observer Object
	ecsServiceNotificationQueue := sqsqueue.NewSqsQueue(stack, jsii.String("ecs_service_notification_queue"), &sqsqueue.SqsQueueConfig{
		Name:           jsii.String(ecsServiceNotificationQueueName + "-" + awsRegion),
		MaxMessageSize: jsii.Number(1024),
	})

	// SQS Queue - ECS Services
	ecsServiceQueue := sqsqueue.NewSqsQueue(stack, jsii.String("ecs_services_queue"), &sqsqueue.SqsQueueConfig{
		Name:           jsii.String(ecsServiceQueueName + "-" + awsRegion),
		MaxMessageSize: jsii.Number(1024),
	})

	// Lambda Function - ECS Service Discovery Lambda
	// Trigger on SQS Queue - ECS Notification
	// Publish Messages to SQS Queue - ECS Services
	ecsServiceDiscoveryLambdaFile := cdktf.NewTerraformAsset(stack, jsii.String("ecs_service_discovery_lambda_file"), &cdktf.TerraformAssetConfig{
		Path: jsii.String(path.Join(cwd, "../ecs-service-discovery-lambda/dist/")),
		Type: cdktf.AssetType_ARCHIVE,
	})

	ecsServiceDiscoveryLambdaS3Object := s3bucketobject.NewS3BucketObject(stack, jsii.String("ecs_service_discovery_lambda_archive"), &s3bucketobject.S3BucketObjectConfig{
		Bucket: bucket.Bucket(),
		Key:    jsii.String("ecs-service-discovery-lambda/" + *ecsServiceDiscoveryLambdaFile.FileName()),
		Source: ecsServiceDiscoveryLambdaFile.Path(),
	})

	lambdaRole := iamrole.NewIamRole(stack, jsii.String("ecs_service_discovery_lambda_role"), &iamrole.IamRoleConfig{
		Name:             jsii.String("ECSServiceDiscoveryLambdaRole"),
		AssumeRolePolicy: &lambdaRolePolicy,
	})

	_ = iamrolepolicy.NewIamRolePolicy(stack, jsii.String("ecs_service_discovery_lambda_ecs_execution_policy"), &iamrolepolicy.IamRolePolicyConfig{
		Name:   jsii.String("ECSServiceDiscoveryPolicy"),
		Role:   lambdaRole.Name(),
		Policy: aws.String(ecsServicePolicy),
	})

	_ = iamrolepolicy.NewIamRolePolicy(stack, jsii.String("ecs_service_discovery_lambda_ec2_execution_policy"), &iamrolepolicy.IamRolePolicyConfig{
		Name:   jsii.String("EC2ServiceDiscoveryPolicy"),
		Role:   lambdaRole.Name(),
		Policy: aws.String(ec2ServicePolicy),
	})

	_ = iamrolepolicy.NewIamRolePolicy(stack, jsii.String("ecs_service_discovery_lambda_sqs_execution_policy"), &iamrolepolicy.IamRolePolicyConfig{
		Name:   jsii.String("SQSReadWritePolicy"),
		Role:   lambdaRole.Name(),
		Policy: aws.String(sqsServicePolicy),
	})

	_ = iamrolepolicy.NewIamRolePolicy(stack, jsii.String("ecs_service_discovery_lambda_cwlog_execution_policy"), &iamrolepolicy.IamRolePolicyConfig{
		Name:   jsii.String("CloudWatchLogReadWritePolicy"),
		Role:   lambdaRole.Name(),
		Policy: aws.String(cloudWatchLogServicePolicy),
	})

	lambdaFilePath := cdktf.Token_AsString(cdktf.Fn_Abspath(ecsServiceDiscoveryLambdaFile.Path()), &cdktf.EncodingOptions{})
	hash := cdktf.Fn_Filebase64sha256(lambdaFilePath)

	// Lambda Function - ECS Service Discovery - Notification Component
	ecsServiceDiscoveryLambda := lambdafunction.NewLambdaFunction(stack, jsii.String("ecs_service_discovery_lambda"), &lambdafunction.LambdaFunctionConfig{
		FunctionName:   aws.String("ecs-service-discovery-lambda"),
		S3Bucket:       bucket.Bucket(),
		S3Key:          ecsServiceDiscoveryLambdaS3Object.Key(),
		Role:           lambdaRole.Arn(),
		Runtime:        aws.String("provided.al2"),
		Handler:        aws.String("main"),
		Timeout:        aws.Float64(lambdaTimeout),
		SourceCodeHash: hash,
		VpcConfig: &lambdafunction.LambdaFunctionVpcConfig{
			SecurityGroupIds: &[]*string{awsLambdaSecurityGroupId.StringValue()},
			SubnetIds:        &[]*string{awsVpcPrivateSubnetId1.StringValue(), awsVpcPrivateSubnetId2.StringValue()},
		},
		Environment: &lambdafunction.LambdaFunctionEnvironment{
			Variables: &map[string]*string{
				"SQS_QUEUE_URL": ecsServiceQueue.Url(),
			},
		},
		DependsOn: &[]cdktf.ITerraformDependable{ecsServiceQueue},
	})

	_ = lambdaeventsourcemapping.NewLambdaEventSourceMapping(stack, jsii.String("ecs_service_discovery_lambda_source"), &lambdaeventsourcemapping.LambdaEventSourceMappingConfig{
		EventSourceArn: ecsServiceNotificationQueue.Arn(),
		FunctionName:   ecsServiceDiscoveryLambda.Arn(),
		BatchSize:      jsii.Number(1),
		Enabled:        true,
		DependsOn:      &[]cdktf.ITerraformDependable{ecsServiceQueue, ecsServiceDiscoveryLambda},
	})

	// SQS Queue - ECS Services Tasks
	ecsServiceTaskQueue := sqsqueue.NewSqsQueue(stack, jsii.String("ecs_service_tasks_queue"), &sqsqueue.SqsQueueConfig{
		Name:           jsii.String(ecsServiceTaskQueueName + "-" + awsRegion),
		MaxMessageSize: jsii.Number(1024),
	})

	// Lambda Function - ECS Service Task Discovery
	// Trigger on SQS Queue - ECS Services
	// Publish Messages to SQS Queue - ECS Service Task Queue
	ecsServiceTaskDiscoveryLambdaFile := cdktf.NewTerraformAsset(stack, jsii.String("ecs_service_task_discovery_lambda_file"), &cdktf.TerraformAssetConfig{
		Path: jsii.String(path.Join(cwd, "../ecs-service-task-discovery-lambda/dist/")),
		Type: cdktf.AssetType_ARCHIVE,
	})

	ecsServiceTaskDiscoveryLambdaS3Object := s3bucketobject.NewS3BucketObject(stack, jsii.String("ecs_service_task_discovery_lambda_archive"), &s3bucketobject.S3BucketObjectConfig{
		Bucket: bucket.Bucket(),
		Key:    jsii.String("ecs-service-task-discovery-lambda/" + *ecsServiceTaskDiscoveryLambdaFile.FileName()),
		Source: ecsServiceTaskDiscoveryLambdaFile.Path(),
	})

	taskLambdaFilePath := cdktf.Token_AsString(cdktf.Fn_Abspath(ecsServiceTaskDiscoveryLambdaFile.Path()), &cdktf.EncodingOptions{})
	taskLambdaHash := cdktf.Fn_Filebase64sha256(taskLambdaFilePath)

	// Lambda FUnction - ECS Service Task Discovery
	ecsServiceTaskDiscoveryLambda := lambdafunction.NewLambdaFunction(stack, jsii.String("ecs_service_task_discovery_lambda"), &lambdafunction.LambdaFunctionConfig{
		FunctionName:   aws.String("ecs-service-task-discovery-lambda"),
		S3Bucket:       bucket.Bucket(),
		S3Key:          ecsServiceTaskDiscoveryLambdaS3Object.Key(),
		Role:           lambdaRole.Arn(),
		Runtime:        aws.String("provided.al2"),
		Handler:        aws.String("main"),
		Timeout:        aws.Float64(lambdaTimeout),
		SourceCodeHash: taskLambdaHash,
		VpcConfig: &lambdafunction.LambdaFunctionVpcConfig{
			SecurityGroupIds: &[]*string{awsLambdaSecurityGroupId.StringValue()},
			SubnetIds:        &[]*string{awsVpcPrivateSubnetId1.StringValue(), awsVpcPrivateSubnetId2.StringValue()},
		},
		Environment: &lambdafunction.LambdaFunctionEnvironment{
			Variables: &map[string]*string{
				"SQS_QUEUE_URL": ecsServiceTaskQueue.Url(),
			},
		},
		DependsOn: &[]cdktf.ITerraformDependable{ecsServiceTaskQueue},
	})

	_ = lambdaeventsourcemapping.NewLambdaEventSourceMapping(stack, jsii.String("ecs_service_task_discovery_lambda_source"), &lambdaeventsourcemapping.LambdaEventSourceMappingConfig{
		EventSourceArn: ecsServiceQueue.Arn(),
		FunctionName:   ecsServiceTaskDiscoveryLambda.Arn(),
		BatchSize:      jsii.Number(1),
		Enabled:        true,
		DependsOn:      &[]cdktf.ITerraformDependable{ecsServiceQueue, ecsServiceTaskDiscoveryLambda},
	})

	// Lambda Function - ECS Service Task Notify
	// Trigger on Message - SQS Queue - ECS Service Tasks
	// Trigger Tasks Notify API
	ecsServiceTaskNotifyLambdaFile := cdktf.NewTerraformAsset(stack, jsii.String("ecs_service_task_notify_lambda_file"), &cdktf.TerraformAssetConfig{
		Path: jsii.String(path.Join(cwd, "../ecs-service-task-notify-lambda/dist/")),
		Type: cdktf.AssetType_ARCHIVE,
	})

	ecsServiceTaskNotifyLambdaS3Object := s3bucketobject.NewS3BucketObject(stack, jsii.String("ecs_service_task_notify_lambda_archive"), &s3bucketobject.S3BucketObjectConfig{
		Bucket: bucket.Bucket(),
		Key:    jsii.String("ecs-service-task-notify-lambda/" + *ecsServiceTaskNotifyLambdaFile.FileName()),
		Source: ecsServiceTaskNotifyLambdaFile.Path(),
	})

	notifyLambdaFilePath := cdktf.Token_AsString(cdktf.Fn_Abspath(ecsServiceTaskNotifyLambdaFile.Path()), &cdktf.EncodingOptions{})
	notifyLambdaHash := cdktf.Fn_Filebase64sha256(notifyLambdaFilePath)

	// Lambda Function - ECS Service Task Notification
	ecsServiceTaskNotifyLambda := lambdafunction.NewLambdaFunction(stack, jsii.String("ecs_service_task_notify_lambda"), &lambdafunction.LambdaFunctionConfig{
		FunctionName:   aws.String("ecs-service-task-notify-lambda"),
		S3Bucket:       bucket.Bucket(),
		S3Key:          ecsServiceTaskNotifyLambdaS3Object.Key(),
		Role:           lambdaRole.Arn(),
		Runtime:        aws.String("provided.al2"),
		Handler:        aws.String("main"),
		Timeout:        aws.Float64(lambdaTimeout),
		SourceCodeHash: notifyLambdaHash,
		VpcConfig: &lambdafunction.LambdaFunctionVpcConfig{
			SecurityGroupIds: &[]*string{awsLambdaSecurityGroupId.StringValue()},
			SubnetIds:        &[]*string{awsVpcPrivateSubnetId1.StringValue(), awsVpcPrivateSubnetId2.StringValue()},
		},
	})

	_ = lambdaeventsourcemapping.NewLambdaEventSourceMapping(stack, jsii.String("ecs_service_task_notify_lambda_source"), &lambdaeventsourcemapping.LambdaEventSourceMappingConfig{
		EventSourceArn: ecsServiceTaskQueue.Arn(),
		FunctionName:   ecsServiceTaskNotifyLambda.Arn(),
		BatchSize:      jsii.Number(1),
		Enabled:        true,
		DependsOn:      &[]cdktf.ITerraformDependable{ecsServiceTaskQueue, ecsServiceTaskNotifyLambda},
	})

	// Output SQS Queue URL
	cdktf.NewTerraformOutput(stack, jsii.String("EcsServicesNotificationQueueId"), &cdktf.TerraformOutputConfig{
		Value: ecsServiceNotificationQueue.Id(),
	})

	cdktf.NewTerraformOutput(stack, jsii.String("EcsServicesQueueId"), &cdktf.TerraformOutputConfig{
		Value: ecsServiceQueue.Id(),
	})

	cdktf.NewTerraformOutput(stack, jsii.String("EcsServiceDiscoveryLambdaArn"), &cdktf.TerraformOutputConfig{
		Value: ecsServiceDiscoveryLambda.Arn(),
	})

	cdktf.NewTerraformOutput(stack, jsii.String("EcsTasksQueueId"), &cdktf.TerraformOutputConfig{
		Value: ecsServiceTaskQueue.Id(),
	})

	return stack
}

func main() {
	app := cdktf.NewApp(nil)

	NewMyStack(app, "ecs-task-notifier-cdktf")
	app.Synth()
}
