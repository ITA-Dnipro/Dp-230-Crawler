package pubsub

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

type Task struct {
	ID      string          `json:"id"`
	Payload json.RawMessage `json:"payload"`
}

type ToConsume struct {
	URL       string   `json:"url"`
	ForwardTo []string `json:"forwardTo"`
}

type ToProduce struct {
	URLs []string `json:"urls"`
}

func NewTask(id string) *Task {
	return &Task{
		ID: id,
	}
}

func (tsk *Task) ConsumePayload() (*ToConsume, error) {
	consume := new(ToConsume)
	err := json.Unmarshal(tsk.Payload, consume)
	if err != nil {
		log.Printf("Error unmarshalling task payload to json: %v\n", err)
		return nil, err
	}
	return consume, nil
}

func (tsk *Task) ProducePayload(data *ToProduce) error {
	toPayload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	tsk.Payload = toPayload
	return nil
}

func (tsk *Task) ToJson() []byte {
	res, err := json.Marshal(tsk)
	if err != nil {
		log.Printf("Error marshalling %v to json: %v\n", tsk, err)
		return nil
	}
	return res
}

func (tsk *Task) FromJson(data []byte) error {
	return json.Unmarshal(data, tsk)
}

type Message struct {
	Key   string
	Value Task
	Time  time.Time
}

func NewMessageProduce(taskID string, urls []string) Message {
	tsk := NewTask(taskID)
	curData := &ToProduce{
		URLs: urls,
	}
	tsk.ProducePayload(curData)

	return Message{
		Key:   fmt.Sprint(uuid.New()),
		Value: *tsk,
		Time:  time.Now(),
	}
}
