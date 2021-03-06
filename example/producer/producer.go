package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"yithQ/message"
)

var msgStr = `the great race of Yith can be through space and time :`

func main() {
	producerUrl := "http://localhost:9970/produce"
	msgs := make([]*message.Message, 0)
	for i := 0; i < 1<<12; i++ {
		msg := &message.Message{
			Body: []byte(msgStr + strconv.Itoa(i)),
		}
		msgs = append(msgs, msg)
	}
	messages := &message.Messages{
		Topic:       "yith",
		PartitionID: 001,
		Msgs:        msgs,
		MetaVersion: 0,
	}
	byt, err := json.Marshal(messages)
	if err != nil {
		panic("json marshal error : " + err.Error())
	}
	_, err = http.Post(producerUrl, "application/json", bytes.NewBuffer(byt))
	if err != nil {
		panic("producer http post error : " + err.Error())
	}
}
