package internal

type ServiceMessage struct {
	Cluster               string `json:"cluster"`
	Service               string `json:"service"`
	NotifyMeContainerPort string `json:"notify_me_container_port"`
	NotifyMeAPIUri        string `json:"notify_me_api_uri"`
}

func NewServiceMessage() *ServiceMessage {
	return &ServiceMessage{}
}

type TaskNotifyMessage struct {
	NotifyTaskArn       string `json:"notify_task_arn"`
	NotifyMeHostAddress string `json:"notify_me_host_address"`
	NotifyMeHostPort    string `json:"notify_me_host_port"`
	NotifyMeAPIUri      string `json:"notify_me_api_uri"`
}

func NewTaskNotifyMessage() *TaskNotifyMessage {
	return &TaskNotifyMessage{}
}
