# Amazon Elastic Container Service (ECS)

Amazon Elastic Container Service (ECS) provides a fully managed container service solution. It facilitate deployment, scalaing and high availability of containerized workloads. 


![Amazon ECS Service Task](./docs/images/ecs_service.png)


**Capacity Providers:**

Define infrastructure workload capacity offers two options i.e. EC2 Instance or Fargate. ECS service task container running on compute infrastructure.

**ECS Service:**

It represent the desired state of containerized application which includes 
- Task Definition - Defines the container image, CPU, memory requirements, and networking configuration.
- Deployment Configuration - Specifies the number of tasks to run and how deployments should be rolled out (e.g., blue/green deployments)
- Auto-scaling Configuration - Enables automatic scaling of tasks based on predefined metrics like CPU utilization or application load

**Traffic Routing:**

Amazon Elastic Load Balancer (ALB/NLB) use to route the incoming traffic to healthy container instances running within ECS Service registered with dedicated Target Group.


# Amazon ECS Service Task Notifier - PoC

## Problem Overview

The PoC is to triggering each ECS Task of an ECS Service within an ECS Cluster using Event-Driven Architecture

Consider a scenario where we have to send event notification to indivusual task running within ECS service on such event container will take certain action. With passing request through a routing wont help as request will be handled by only one of ECS Task.

## Solution Overview


### Assumptions

- The containerized application is deployed as a microservices application (REST API).
- The Task Definition includes a key-value pair under `dockerlabels` key, such as `NOTIFY_ME_CONTAINER_PORT` and  `NOTIFY_ME_API_URI`. This value indicates which ECS Services are candidates to receive event notifications.
- A Notify API is hosted by a container within the microservice application. for e.g. `/v1.0/notify`
- The ECS Cluster uses EC2 instances as its capacity provider.
- The EC2 instances for the ECS Cluster run within private subnets of a VPC.


![Amazon ECS Service Task Notifier PoC](./docs/images/ecs_task_notifier_poc.png)



### Key components:

**Observer Service:**

In an AWS environment, various services can function as observer services. These may include DynamoDB streams, AWS S3 object lifecycle events, and messages/events received on Amazon SQS.

As part of the proof of concept (PoC), chosen to publish events to an Amazon SQS queue as an event notification. ECS tasks designated as event subscribers will be notified of these events by invoking a Notify API endpoint, for example, `/v1.0/notify` API.

```json
{
    "cluster": "ecs_cluster_name"
}
```

Not all ECS services need to be event subscribers. By leveraging a `dockerlabels` configuration, we can identify ECS services implementing a "Notify API" (e.g., `/v1.0/notify`) and are thus eligible to receive event notifications. This convention simplifies deployment by avoiding unnecessary notifications to services that don't handle events.

**Task Notifier:**

The Task Notifier is implemented adhering to the event-driven architecture modern pattern, leveraging AWS serverless services such as three Lambda functions and two SQS queues to execute notification triggering concurrently.

- ECS Service Discovery Lambda:

This Lambda function is triggered by messages in the observer SQS queue. It retrieves a list of all ECS services for the specified cluster and filters those with specific key-value pairs, such as `NOTIFY_ME_CONTAINER_PORT` and `NOTIFY_ME_API_URI`, part of the dockerlabels section in the TaskDefinition. It then prepares a message for each filtered ECS service and publishes it to the `ecs_service` SQS queue for further processing.

```json
{
    "cluster": "ecs_cluster_name",
    "service": "ecs_service_name",
    "notify_me_container_port": "notify_me_container_port",
    "notify_me_api_uri": "notify_me_api_uri"
}
```

- ECS Service Task Discovery Lambda:

Triggered by messages in the `ecs_service` SQS queue, this Lambda function retrieves details for all ECS tasks based on the cluster and service name provided. It gathers information such as IP address, host port, and notification API URI, then publishes these details to the `ecs_service_tasks` SQS queue.

```json
{
    "notify_task_arn": "notify_task_arn", // Used for logging and auditing
    "notify_me_host_address": "notify_me_host_address",
    "notify_me_host_port": "notify_me_host_port",
    "notify_me_api_uri": "notify_me_api_uri"
}
```

- ECS Service Task Notify Lambda:

Triggered by messages in the `ecs_service_tasks` SQS queue, this Lambda function executes the ECS Task Notification API for each task.


# Amazon ECS Service Task Notifier - Infrastructure

## Pre-requisites

- VPC two private subnet id's
- Security Group for Lambda function



![Amazon ECS Service Task IaC](./docs/images/ecs_task_notifier_iac.png)

| Sr.No. | AWS Service      | Name                                  | Purpose                         |
|--------|------------------|---------------------------------------|---------------------------------|
| 1      | SQS              | ecs_service_notification_aws_region   | Observer Service                |
| 2      | Lambda Function  | ecs_service_discovery                 | ECS Service Discovery           |
| 3      | SQS              | ecs_service_aws_region                | ECS Service Message             |
| 4      | Lambda Function  | ecs_service_task_discovery            | ECS Service Task Discovery      |
| 5      | SQS              | ecs_service_task_aws_region           | ECS Task Message                |
| 6      | Lambda Function  | ecs_service_task_notify               | ECS Service Task Notifier       |
| *      | S3 Bucket        | ecs_task_notifier_lambdas_aws_region | Lambda Function Archives  |


## Out of scope

- ECS Cluster
- Micro-services deployed as a containerized application
- VPC Network

# How to Use

**TODO** Steps to provision infrastructure and test.

# Reference

- ToDo