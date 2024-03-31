package internal

import (
	"context"
	"testing"
)

var listServicesTests = map[string]struct {
	cluster string
}{
	"test case 1": {"ecs_cluster_name"},
}

func TestListECSServices(t *testing.T) {
	ctx := context.TODO()
	awsService, err := NewAWSService(ctx)
	if err != nil {
		t.Fail()
	}

	for name, tc := range listServicesTests {
		t.Run(name, func(t *testing.T) {
			actual, err := awsService.ListECSServices(ctx, tc.cluster)
			if err != nil {
				t.Fail()
			} else if len(actual) <= 0 {
				t.Fail()
			}
		})
	}
}
