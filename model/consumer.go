package model

import (
	"time"
)

type MessageConsume struct {
	Key    string
	Value  *TaskConsume
	Time   time.Time
	Origin any
}

type TaskConsume struct {
	ID        string   `json:"id"`
	URL       string   `json:"url"`
	ForwardTo []string `json:"forwardTo"`
}
