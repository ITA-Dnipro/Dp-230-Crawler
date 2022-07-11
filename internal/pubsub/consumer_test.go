package pubsub

import (
	"context"
	"errors"
	"testing"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
	"parabellum.crawler/model"
)

type kafkaReaderStub struct {
	valuePayload string
}

func (kr *kafkaReaderStub) FetchMessage(ctx context.Context) (kafka.Message, error) {
	select {
	case <-ctx.Done():
		return kafka.Message{}, ctx.Err()
	default:
		return kafka.Message{Value: []byte(kr.valuePayload)}, nil
	}
}

func (kr *kafkaReaderStub) CommitMessages(ctx context.Context, mes ...kafka.Message) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func (kr *kafkaReaderStub) Close() error {
	return nil
}

func TestFetchMessage(t *testing.T) {
	payload := `{"id":"test-task-1", "url":"testurl", "forwardTo":[ "forward" ]}`

	tabTest := []struct {
		name     string
		consumer *Consumer
		expected *model.MessageConsume
	}{
		{
			name: "correct",
			consumer: NewConsumer(&kafkaReaderStub{
				valuePayload: payload,
			}, "sometopic"),
			expected: &model.MessageConsume{
				Origin: &kafka.Message{Value: []byte(payload)},
				Value: &model.TaskConsume{
					ID:        "test-task-1",
					URL:       "testurl",
					ForwardTo: []string{"forward"},
				},
			},
		},

		{
			name:     "wrong unmarshalling",
			consumer: NewConsumer(&kafkaReaderStub{}, "sometopic"),
			expected: &model.MessageConsume{},
		},
	}

	for _, test := range tabTest {
		t.Run(test.name, func(t *testing.T) {
			received, _ := test.consumer.FetchMessage(context.Background())
			require.Equal(t, test.expected, received, "should equal")
		})
	}
}

func TestCommitMessage(t *testing.T) {
	cons := NewConsumer(&kafkaReaderStub{}, "sometopic")

	tabTest := []struct {
		name     string
		message  model.MessageConsume
		expected error
	}{
		{
			name:     "correct",
			message:  model.MessageConsume{Origin: &kafka.Message{Value: []byte("test")}},
			expected: nil,
		},
		{
			name:     "wrong Origin",
			message:  model.MessageConsume{Origin: nil},
			expected: errors.New("message Origin of wrong type"),
		},
	}
	for _, test := range tabTest {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, cons.CommitMessage(context.Background(), &test.message), "should equal")
		})
	}
}

func TestConsumerClose(t *testing.T) {
	cons := NewConsumer(&kafkaReaderStub{}, "sometopic")

	require.NoError(t, cons.Close(), "no error expected")
}
