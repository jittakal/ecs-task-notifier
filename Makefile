# Makefile for Amazon ECS Task Notifier 

# Default AWS Region (change this value as needed)
AWS_REGION ?= us-east-1

.PHONY: get synth deploy destroy lambda test

# Default target
default: get

# Run linting on the CDKTF project
get:
	@echo "Running cdktf get ..."
	@cd ecs-task-notifier-cdktf && \
		cdktf get && \
		go mod tidy

# Synthesize Terraform configuration
synth:
	@echo "Synthesizing Terraform configuration..."
	@cd ecs-task-notifier-cdktf && \
		cdktf synth

# Deploy infrastructure using CDKTF
deploy:
	@echo "Deploying infrastructure using CDKTF..."
	@cd ecs-task-notifier-cdktf && \
		cdktf deploy

# Destroy infrastructure
destroy:
	@echo "Destroying infrastructure..."
	@cd ecs-task-notifier-cdktf && \
		cdktf destroy

#golangci-lint run --enable-all --timeout=5m &&

lambda:
	@echo "Building ecs-service-discovery lambda ..."
	@cd ecs-service-discovery-lambda && \
		go mod tidy && \
		go fmt && \
		mkdir -p dist && \
		GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o ./dist/bootstrap main.go

	@echo "Building ecs-service-task-discovery lambda ..."
	@cd ecs-service-task-discovery-lambda && \
		go mod tidy && \
		go fmt && \
		mkdir -p dist && \
		GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o ./dist/bootstrap main.go

	@echo "Building ecs-service-task-notify lambda ..."
	@cd ecs-service-task-notify-lambda && \
		go mod tidy && \
		go fmt && \
		mkdir -p dist && \
		GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o ./dist/bootstrap main.go

test:
	@echo "Testing ecs-task-notifier ..."
	@cd ecs-task-notifier-test && \
		go run main.go -r $(AWS_REGION) -c $(ECS_CLUSTER_NAME) -q $(SQS_QUEUE_NAME)
