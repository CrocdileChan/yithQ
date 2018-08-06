package yith

import (
	"sync"
	"yithQ/yith/message"
)

type Node struct {
	IP             string
	topicPartition *sync.Map   //map[int]*Partition
}

func NewNode(ip string) *Node{
	return &Node{
		IP:ip,
		topicPartition:&sync.Map{},
	}
}

func (n *Node) AddTopicPartition(tp map[string]*Partition) {
	for topic, p := range tp {
		n.topicPartition.Store(topic,p)
	}
}

func (n *Node) Produce(topic string, msg *message.Message) error {

}

func (n *Node) Consume(topic string,popOffset int64) (*message.Message,error){

}
