package model

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type MessageProduce struct {
	Key   string
	Value *TaskProduce
	Time  time.Time
}

type TaskProduce struct {
	ID   string   `json:"id"`
	URLs []string `json:"urls"`
}

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
