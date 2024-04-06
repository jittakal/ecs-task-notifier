package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/spf13/cobra"
)

func getQueueURL(ctx context.Context, client *sqs.Client, queueName string) (*string, error) {
	output, err := client.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{
		QueueName: &queueName,
	})
	if err != nil {
		return nil, err
	}
	return output.QueueUrl, nil
}

func main() {
	var awsRegion, ecsClusterName, sqsQueueName string

	// Initialize the CLI application
	rootCmd := &cobra.Command{
		Use:   "ecs-task-notifier",
		Short: "Send notifications to ECS service",
		Run: func(cmd *cobra.Command, args []string) {
			// Load AWS configuration
			cfg, err := config.LoadDefaultConfig(context.Background())
			if err != nil {
				fmt.Println("Error loading AWS configuration:", err)
				os.Exit(1)
			}

			client := sqs.NewFromConfig(cfg)

			// Get SQS queue URL
			queueURL, err := getQueueURL(context.Background(), client, sqsQueueName)
			if err != nil {
				fmt.Println("Error getting SQS queue URL:", err)
				os.Exit(1)
			}

			// Define the message body
			messageBody := fmt.Sprintf(`{"cluster": "%s"}`, ecsClusterName)

			// Send message to SQS queue
			result, err := client.SendMessage(context.Background(), &sqs.SendMessageInput{
				MessageBody: aws.String(messageBody),
				QueueUrl:    queueURL,
			})
			if err != nil {
				fmt.Println("Error sending message to queue:", err)
				os.Exit(1)
			}

			fmt.Println("Message sent successfully:", *result.MessageId)
		},
	}

	// Define flags for CLI parameters with short form options
	rootCmd.Flags().StringVarP(&awsRegion, "aws-region", "r", "us-east-1", "AWS Region")
	rootCmd.Flags().StringVarP(&ecsClusterName, "ecs-cluster-name", "c", "", "ECS Cluster Name")
	rootCmd.Flags().StringVarP(&sqsQueueName, "sqs-queue-name", "q", "", "SQS Queue Name")

	// Bind flags to environment variables
	rootCmd.MarkFlagRequired("ecs-cluster-name")
	rootCmd.MarkFlagRequired("sqs-queue-name")

	// Execute the CLI application
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
