# Makefile for ECS Service Notifications 

.PHONY: get synth deploy destroy

# Default target
default: get

# Run linting on the CDKTF project
get:
	@echo "Running cdktf get ..."
	@cdktf get && go mod tidy

# Synthesize Terraform configuration
synth:
	@echo "Synthesizing Terraform configuration..."
	@cdktf synth

# Deploy infrastructure using CDKTF
deploy:
	@echo "Deploying infrastructure using CDKTF..."
	@cdktf deploy

# Destroy infrastructure
destroy:
	@echo "Destroying infrastructure..."
	@cdktf destroy
