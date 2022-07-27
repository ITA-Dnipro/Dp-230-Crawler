package pubsub

import (
	"context"
	"testing"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
	"parabellum.crawler/internal/model"
)

type kafkaWriterStub struct{}

func (kw *kafkaWriterStub) WriteMessages(ctx context.Context, mes ...kafka.Message) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func (kw *kafkaWriterStub) Close() error {
	return nil
}

func TestPublicMessage(t *testing.T) {
	prod := NewProducer(&kafkaWriterStub{}, "test-topic")
	mes := model.NewMessageProduce("test-task-1", []string{"url1", "url2"})

	require.NoError(t, prod.PublicMessage(context.Background(), mes), "no error expected")
}

func TestProducerClose(t *testing.T) {
	prod := NewProducer(&kafkaWriterStub{}, "test-topic")

	require.NoError(t, prod.Close(), "no error expected")
}
