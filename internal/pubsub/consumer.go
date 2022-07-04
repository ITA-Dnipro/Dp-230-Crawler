package pubsub

import (
	"context"
	"strings"

	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	kafkaReader *kafka.Reader
}

func NewConsumer(url, topic string) *Consumer {
	result := new(Consumer)
	result.kafkaReader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:  strings.Split(url, ","),
		Topic:    topic,
		MinBytes: 10e3,
	})

	return result
}

func (prod *Consumer) ReadMessage(ctx context.Context) (Message, error) {
	message := Message{}

	msg, err := prod.kafkaReader.ReadMessage(ctx)
	if err != nil {
		return message, err
	}

	message.Key = string(msg.Key)
	message.Value = string(msg.Value)
	message.Time = msg.Time

	return message, nil
}

func (prod *Consumer) Close() error {
	return prod.kafkaReader.Close()
}
