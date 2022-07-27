package model

import (
	"time"
)

//MessageConsume received messages representation
type MessageConsume struct {
	Key    string       //message key from pubsub provider
	Value  *TaskConsume //message value
	Time   time.Time    //time of the message
	Origin any          //message itself (for committing)
}

//TaskConsume received task format
type TaskConsume struct {
	ID        string   `json:"id"`        //main task id
	URL       string   `json:"url"`       //main task url to crawl
	ForwardTo []string `json:"forwardTo"` //list of test-services topics names to send results to
}
