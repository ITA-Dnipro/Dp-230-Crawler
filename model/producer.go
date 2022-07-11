package model

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

//MessageProduce message to send further to test-services topics
type MessageProduce struct {
	Key   string       //message key
	Value *TaskProduce //message value itself
	Time  time.Time    //time of the message
}

//TaskProduce published task format
type TaskProduce struct {
	ID   string   `json:"id"`   //main task id
	URLs []string `json:"urls"` //urls for the receiver to work with
}

//NewMessageProduce is a constructor for [model.MessageProduce]
func NewMessageProduce(taskID string, urls []string) *MessageProduce {
	tsk := &TaskProduce{
		ID:   taskID,
		URLs: urls,
	}

	return &MessageProduce{
		Key:   fmt.Sprint(uuid.New()),
		Value: tsk,
		Time:  time.Now(),
	}
}
