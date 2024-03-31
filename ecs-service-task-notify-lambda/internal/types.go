package internal

type TaskNotifyMessage struct {
	NotifyTaskArn       string `json:"notify_task_arn"`
	NotifyMeHostAddress string `json:"notify_me_host_address"`
	NotifyMeHostPort    string `json:"notify_me_host_port"`
	NotifyMeAPIUri      string `json:"notify_me_api_uri"`
}

func NewTaskNotifyMessage() *TaskNotifyMessage {
	return &TaskNotifyMessage{}
}
