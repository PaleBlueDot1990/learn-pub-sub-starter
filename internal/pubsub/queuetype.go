package pubsub

type SimpleQueueType string 

const (
	Durable   SimpleQueueType = "DURABLE"
	Transient SimpleQueueType = "TRANSIENT"
)