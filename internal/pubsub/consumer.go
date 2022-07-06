package pubsub

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"

	"github.com/segmentio/kafka-go"
	"parabellum.crawler/model"
)

const groupID = "crawler-service"

type Consumer struct {
	Topic       string
	kafkaReader *kafka.Reader
}

func NewConsumer(url, topic string) *Consumer {
	result := new(Consumer)
	result.kafkaReader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:  strings.Split(url, ","),
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 10e3,
		MaxBytes: 10e5,
	})
	result.Topic = topic

	return result
}

func (cons *Consumer) FetchMessage(ctx context.Context) (*model.MessageConsume, error) {
	message := &model.MessageConsume{}

	msg, err := cons.kafkaReader.FetchMessage(ctx)
	if err != nil {
		return message, err
	}

	task := new(model.TaskConsume)
	err = json.Unmarshal(msg.Value, task)
	if err != nil {
		return message, err
	}
	message.Key = string(msg.Key)
	message.Value = task
	message.Time = msg.Time
	message.Origin = &msg

	log.Println("Read from Kafka. Task ID:", message.Value.ID)

	return message, nil
}

func (cons *Consumer) CommitMessage(ctx context.Context, msg *model.MessageConsume) error {
	m, ok := msg.Origin.(*kafka.Message)
	if !ok {
		return errors.New("message Origin of wrong type")
	}

	return cons.kafkaReader.CommitMessages(ctx, *m)
}

func (cons *Consumer) Close() error {
	return cons.kafkaReader.Close()
}
