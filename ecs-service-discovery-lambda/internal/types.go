package internal

type EcsNotify struct {
	Cluster string `json:"cluster"`
}

func NewEcsNotify() *EcsNotify {
	return &EcsNotify{}
}

type EcsService struct {
	Cluster        string `json:"cluster"`
	Service        string `json:"service"`
	TaskDefinition string `json:"task_definition"`
}

func NewEcsService() *EcsService {
	return &EcsService{}
}

type ServiceMessage struct {
	Cluster               string `json:"cluster"`
	Service               string `json:"service"`
	NotifyMeContainerPort string `json:"notify_me_container_port"`
	NotifyMeAPIUri        string `json:"notify_me_api_uri"`
}

func NewServiceMessage() *ServiceMessage {
	return &ServiceMessage{}
}
