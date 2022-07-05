package pubsub

import (
	"context"
	"log"
	"strings"

	"github.com/segmentio/kafka-go"
)

const groupID = "crawler-service"

type Consumer struct {
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

	return result
}

func (prod *Consumer) ReadMessage(ctx context.Context) (Message, error) {
	message := Message{}

	msg, err := prod.kafkaReader.ReadMessage(ctx)
	if err != nil {
		return message, err
	}

	task := new(Task)
	task.FromJson(msg.Value)
	message.Key = string(msg.Key)
	message.Value = *task
	message.Time = msg.Time

	log.Println("Read from Kafka. Task ID:", message.Value.ID)

	return message, nil
}

func (prod *Consumer) Close() error {
	return prod.kafkaReader.Close()
}
