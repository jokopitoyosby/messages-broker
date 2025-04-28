package main

import (
	"fmt"
	"time"

	"github.com/jokopitoyosby/messages-broker/publisher"
	"github.com/nats-io/nats.go"
)

func main() {
	streamName := "GPS_STREAM"
	url := "nats://localhost:4222,nats://localhost:4223,nats://localhost:4224"
	username := "nats"
	password := "your_secure_password"
	pub := publisher.NewNatsPublisher(url, streamName, []string{"gps.position"}, username, password, func(topic string, msg *nats.Msg) ([]byte, error) {
		return []byte(msg.Data), nil
	})
	pub.Start()

	for {
		time.Sleep(1 * time.Millisecond)
		msg := time.Now().Format("2006-01-02 15:04:05")
		err := pub.Publish("gps.position", []byte(msg))
		if err != nil {
			fmt.Printf("err publish: %v\n", err)
			continue
		}
		//fmt.Printf("Published to gps.position: %v\n", msg)
	}
}
