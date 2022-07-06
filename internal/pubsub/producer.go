package pubsub

import (
	"context"
	"encoding/json"
	"log"

	"github.com/segmentio/kafka-go"
	"parabellum.crawler/model"
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

func (prod *Producer) PublicMessage(ctx context.Context, message *model.MessageProduce) error {
	valueJson, err := json.Marshal(message.Value)
	if err != nil {
		log.Printf("Error marshalling %v to json: %v\n", message.Value, err)

		return err
	}

	msg := kafka.Message{
		Key:   []byte(message.Key),
		Value: valueJson,
		Time:  message.Time,
	}

	log.Println("Publishing into Kafka topic:", prod.kafkaWriter.Topic)
	msgOut := string(msg.Value)
	if len(msgOut) > 250 {
		msgOut = msgOut[:250] + "\t..."
	}
	log.Println("\t", msgOut)

	return prod.kafkaWriter.WriteMessages(ctx, msg)
}

func (prod *Producer) Close() error {
	return prod.kafkaWriter.Close()
}
