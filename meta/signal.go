package meta

type Signal int

const (
	Heartbeat Signal = iota
	NodeChange
	TopicPartitionAddChange
	TopicDeleteChange
	TopicReplicaAddChange
	TopicPartitionDeleteChange
	FetchMetadata
	Pickup
)

var (
	HeartbeatStr                  = "heartbeat"
	NodeChangeStr                 = "node-change"
	TopicPartitionAddChangeStr    = "topic-partition-add-change"
	TopicDeleteChangeStr          = "topic-delete-change"
	TopicReplicaAddChangeStr      = "topic-replica-add-change"
	TopicPartitionDeleteChangeStr = "topic-partition-delete-change"
	FetchMetadataStr              = "fetch-metadata"
	PickupStr                     = "pickup"
)

var SignalTypes = []string{
	HeartbeatStr,
	NodeChangeStr,
	TopicPartitionAddChangeStr,
	TopicDeleteChangeStr,
	TopicReplicaAddChangeStr,
	TopicPartitionDeleteChangeStr,
	FetchMetadataStr,
	PickupStr,
}

func (st Signal) String() string {
	return SignalTypes[st]
}
