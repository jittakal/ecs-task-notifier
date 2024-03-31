package internal

import (
	"context"
	"testing"
)

var listContainerInstances = map[string]struct {
	cluster string
}{
	"test case 1": {"ecs_cluster_name_1"},
	"test case 2": {"ecs_cluster_name_2"},
}

func TestListContainerInstances(t *testing.T) {
	ctx := context.TODO()
	awsService, err := NewAWSService(ctx)
	if err != nil {
		t.Fail()
	}

	for name, tc := range listContainerInstances {
		t.Run(name, func(t *testing.T) {
			actual, err := awsService.listContainerInstances(ctx, tc.cluster)
			if err != nil {
				t.Fail()
			} else if len(actual) <= 0 {
				t.Fail()
			}
		})
	}
}

var discoverServiceTasks = map[string]struct {
	cluster string
	service string
	port    string
}{
	"test case 1": {"ecs_cluster_name", "ecs_service_name", "ecs_task_container_port"},
}

func TestDiscoverTasks(t *testing.T) {
	ctx := context.TODO()
	awsService, err := NewAWSService(ctx)
	if err != nil {
		t.Fail()
	}

	for name, tc := range discoverServiceTasks {
		t.Run(name, func(t *testing.T) {
			var serviceMessage *ServiceMessage = NewServiceMessage()
			serviceMessage.Cluster = tc.cluster
			serviceMessage.Service = tc.service
			serviceMessage.NotifyMeContainerPort = tc.port

			actual, err := awsService.DiscoverServiceTasks(ctx, serviceMessage)
			if err != nil {
				t.Fail()
			} else if len(actual) <= 0 {
				t.Fail()
			}
		})
	}
}
