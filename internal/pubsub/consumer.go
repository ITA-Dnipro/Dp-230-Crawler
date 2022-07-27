package pubsub

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"

	"github.com/segmentio/kafka-go"
	"parabellum.crawler/internal/model"
)

const groupID = "crawler-service"

//KafkaReader interface mostly for test implementing purposes
type KafkaReader interface {
	FetchMessage(context.Context) (kafka.Message, error)
	CommitMessages(context.Context, ...kafka.Message) error
	Close() error
}

//Consumer structure representing message consumer
type Consumer struct {
	Topic       string      //topic name
	kafkaReader KafkaReader //reader itself
}

//RealKafkaReader returns filled kafka.Reader from kafka-go lib
func RealKafkaReader(url, topic string) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:  strings.Split(url, ","),
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 10e3,
		MaxBytes: 10e5,
	})
}

//NewConsumer is a constructor for [pubsub.Consumer]
func NewConsumer(krd KafkaReader, topic string) *Consumer {
	result := new(Consumer)
	result.kafkaReader = krd
	result.Topic = topic

	return result
}

//FetchMessage returns [model.MessageConsume] with a received message content
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

//CommitMessage commits given message, so it can be considered as processed
func (cons *Consumer) CommitMessage(ctx context.Context, msg *model.MessageConsume) error {
	m, ok := msg.Origin.(*kafka.Message)
	if !ok {
		return errors.New("message Origin of wrong type")
	}

	return cons.kafkaReader.CommitMessages(ctx, *m)
}

//Close closes consumers' KafkaReader
func (cons *Consumer) Close() error {
	return cons.kafkaReader.Close()
}
