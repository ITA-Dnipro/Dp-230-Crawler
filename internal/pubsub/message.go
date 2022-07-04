package pubsub

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Message struct {
	Key   string
	Value string
	Time  time.Time
}

func NewMessage(msg string) Message {
	return Message{
		Key:   fmt.Sprint(uuid.New()),
		Value: msg,
		Time:  time.Now(),
	}
}
