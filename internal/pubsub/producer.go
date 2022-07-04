package pubsub

import (
	"context"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	kafkaWriter *kafka.Writer
}

func NewProducer(url, topic string) *Producer {
	result := new(Producer)
	result.kafkaWriter = &kafka.Writer{
		Addr:     kafka.TCP(url),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}
	return result
}

func (prod *Producer) PublicMessage(ctx context.Context, message Message) error {
	msg := kafka.Message{
		Key:   []byte(message.Key),
		Value: []byte(message.Value),
		Time:  message.Time,
	}

	return prod.kafkaWriter.WriteMessages(ctx, msg)
}

func (prod *Producer) Close() error {
	return prod.kafkaWriter.Close()
}
