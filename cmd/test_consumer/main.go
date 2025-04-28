package main

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jokopitoyosby/messages-broker/consumer"
)

var handler consumer.MessageHandler

func main() {
	handler = func(topic string, data []byte) error {
		fmt.Printf("topic:%s data: %v\n", topic, string(data))
		return nil
	}
	clientID := "node-1-" + uuid.New().String()
	streamName := "GPS_STREAM"
	clusterAddress := "nats://localhost:4222,nats://localhost:4223,nats://localhost:4224"
	topics := []string{"gps.position"}
	pub := consumer.NewNatsConsumer(clusterAddress, streamName, topics, handler, clientID)
	pub.Start()
	for {
		time.Sleep(1 * time.Second)
	}
}
