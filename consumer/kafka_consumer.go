package consumer

import (
	"context"
	"fmt"
	"sync"

	"github.com/IBM/sarama"
)

// MessageType represents the type of message
type MessageType string

const (
	// Common message types
	MessageTypeEvent    MessageType = "event"
	MessageTypeCommand  MessageType = "command"
	MessageTypeRequest  MessageType = "request"
	MessageTypeResponse MessageType = "response"
)

// KafkaConsumer implements the Consumer interface for Kafka
type KafkaConsumer struct {
	brokers       []string
	topics        []string
	groupID       string
	version       string
	initialOffset int64
	consumer      sarama.ConsumerGroup
	messageCh     chan Message
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	handlers      map[MessageType]MessageHandler
}

// NewKafkaConsumer creates a new Kafka consumer instance
func NewKafkaConsumer(
	brokers []string,
	topics []string,
	groupID string,
	version string,
	initialOffset int64,
	handlers map[MessageType]MessageHandler,
) (*KafkaConsumer, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Parse Kafka version
	kafkaVersion, err := sarama.ParseKafkaVersion(version)
	if err != nil {
		return nil, fmt.Errorf("invalid kafka version: %v", err)
	}

	// Create consumer group config
	consumerConfig := sarama.NewConfig()
	consumerConfig.Version = kafkaVersion
	consumerConfig.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	consumerConfig.Consumer.Offsets.Initial = initialOffset

	// Create consumer group
	consumer, err := sarama.NewConsumerGroup(brokers, groupID, consumerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer group: %v", err)
	}

	return &KafkaConsumer{
		brokers:       brokers,
		topics:        topics,
		groupID:       groupID,
		version:       version,
		initialOffset: initialOffset,
		consumer:      consumer,
		messageCh:     make(chan Message, 100), // Default batch size
		ctx:           ctx,
		cancel:        cancel,
		handlers:      handlers,
	}, nil
}

// Consume implements the Consumer interface
func (k *KafkaConsumer) Consume(ctx context.Context, messages <-chan Message) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-messages:
			// Process message here
			// This is where you would implement your business logic
			fmt.Printf("Processing message from topic %s: %s\n", msg.Topic, string(msg.Data))
		}
	}
}

// Start implements the Consumer interface
func (k *KafkaConsumer) Start() error {
	k.wg.Add(1)
	go func() {
		defer k.wg.Done()
		for {
			select {
			case <-k.ctx.Done():
				return
			default:
				err := k.consumer.Consume(k.ctx, k.topics, k)
				if err != nil {
					fmt.Printf("Error consuming messages: %v\n", err)
				}
			}
		}
	}()
	return nil
}

// Stop implements the Consumer interface
func (k *KafkaConsumer) Stop() error {
	k.cancel()
	k.wg.Wait()
	close(k.messageCh)
	return k.consumer.Close()
}

// GetMessageChannel implements the Consumer interface
func (k *KafkaConsumer) GetMessageChannel() <-chan Message {
	return k.messageCh
}

// ConsumeClaim implements sarama.ConsumerGroupHandler
func (k *KafkaConsumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		// Convert Kafka headers to map
		headers := make(map[string][]byte)
		for _, header := range message.Headers {
			headers[string(header.Key)] = header.Value
		}

		// Get message type from headers
		messageType := MessageType(headers["message_type"])
		if messageType == "" {
			// Default to event if no message type specified
			messageType = MessageTypeEvent
		}

		// Get handler for this message type
		handler, ok := k.handlers[messageType]
		if !ok {
			fmt.Printf("No handler for message type %s\n", messageType)
			continue
		}

		// Call the appropriate handler
		handler("", message.Value)
		session.MarkMessage(message, "")
	}
	return nil
}

// Setup implements sarama.ConsumerGroupHandler
func (k *KafkaConsumer) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup implements sarama.ConsumerGroupHandler
func (k *KafkaConsumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}
