package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/jokopitoyosby/messages-broker/consumer"
	"github.com/jokopitoyosby/messages-broker/models"
)

var handler consumer.MessageHandler

func main() {
	handler = func(topic string, data []byte) error {
		var trackingData models.TrackingData
		if err := json.Unmarshal(data, &trackingData); err != nil {
			fmt.Printf("err: %v\n", err)
			return err
		}
		fmt.Printf("\n\ntrackingData: %+v\n\n", trackingData)
		return nil
	}
	clientID := "server1"
	streamName := "GPS_STREAM"
	//clusterAddress := "nats://localhost:4222,nats://localhost:4223,nats://localhost:4224"
	clusterAddress := "nats://srv3.twingps.com:4222"
	topics := []string{"gps.position", "gps.event"}
	pub := consumer.NewNatsConsumer(clusterAddress, streamName, topics, handler, clientID)
	pub.Start()
	for {
		time.Sleep(1 * time.Second)
	}
}
