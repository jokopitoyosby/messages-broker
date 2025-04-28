package consumer

// MessageHandler represents a function that handles messages of a specific type
type MessageHandler func(topic string, data []byte) error

// ConsumerConfig holds the configuration for any consumer implementation
type ConsumerConfig struct {
	// Common configuration fields
	Name        string
	MaxRetries  int
	BatchSize   int
	Concurrency int
}

// Message represents a consumed message with metadata
type Message struct {
	Data    []byte
	Topic   string
	Key     []byte
	Headers map[string][]byte
}

/*
Interface Consumer
- to handle message from rabbitmq, kafka, redis, etc
*/
type Consumer interface {

	// Start initializes and starts the consumer
	Start() error

	// Stop gracefully shuts down the consumer
	Stop() error
}
