package publisher

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

type DataToPublish struct {
	Topic string
	Data  []byte
}

type NatsPublisher struct {
	conn            *nats.Conn
	url             string
	reconnected     chan bool
	stopChan        chan struct{}
	done            chan struct{}
	data            chan *DataToPublish
	topics          []string
	connecting      bool
	js              nats.JetStreamContext
	streamName      string
	username        string
	password        string
	OnConnect       func(nc *nats.Conn)
	OnDisconnect    func(nc *nats.Conn)
	OnStreamCreated func(nc *nats.Conn, js nats.JetStreamContext)
}

func NewNatsPublisher(url string, username string, password string) *NatsPublisher {
	return &NatsPublisher{
		url:         url,
		username:    username,
		password:    password,
		reconnected: make(chan bool),
		stopChan:    make(chan struct{}),
		done:        make(chan struct{}),
		data:        make(chan *DataToPublish),
	}
}

func (p *NatsPublisher) CreateStream(js nats.JetStreamContext, streamName string, topics []string, replicas int) error {
	_, err := js.StreamInfo(streamName)
	if err == nil {
		return nil
	}
	_, err = js.AddStream(&nats.StreamConfig{
		Name:      p.streamName,
		Subjects:  p.topics,
		Retention: nats.LimitsPolicy, // Pesan disimpan berdasarkan batas waktu/jumlah
		MaxAge:    24 * time.Hour,    // Simpan pesan maksimal 24 jam
		Storage:   nats.FileStorage,  // Simpan ke disk (persisten)
		Replicas:  replicas,
	})
	return err
}
func (p *NatsPublisher) Connect() error {
	fmt.Printf("Connecting to NATS: %s\n", p.url)
	opts := []nats.Option{
		nats.UserInfo(p.username, p.password),
		nats.ReconnectWait(2 * time.Second),
		nats.MaxReconnects(-1), // Unlimited reconnects
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			log.Printf("Disconnected from NATS: %v", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Printf("Reconnected to NATS at %s", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			log.Printf("NATS connection closed")
		}),
	}

	nc, err := nats.Connect(p.url, opts...)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %v", err)
	}
	fmt.Println("Connected to NATS")

	// 2. Buat JetStream Context
	// js, err := nc.JetStream()
	// if err != nil {
	// 	log.Printf("Gagal akses JetStream:%+v\n", err)
	// 	nc.Close()
	// 	return fmt.Errorf("failed to create JetStream context: %v", err)
	// }
	if p.OnConnect != nil {
		p.OnConnect(nc)
	}
	// _, err = js.StreamInfo(p.streamName)
	// if err == nil {
	// 	err = p.CreateStream(js, p.streamName, p.topics, 3)
	// 	if err != nil {
	// 		fmt.Printf("err: %v\n", err)
	// 		nc.Close()
	// 		return fmt.Errorf("failed to create stream: %v", err)
	// 	}
	// }
	// if p.OnStreamCreated != nil {
	// 	p.OnStreamCreated(nc, js)
	// }

	// p.js = js
	// p.conn = nc
	return nil
}

func (p *NatsPublisher) Publish(topic string, message []byte) error {
	if p.js == nil {
		return fmt.Errorf("js is not initialized")
	}
	_, err := p.js.PublishAsync(topic, message, nats.ExpectStream(p.streamName))
	if err != nil {
		return err
	}

	//fmt.Printf("ack: %v\n", ack)
	return nil
	// fmt.Printf("ack: %+v\n", ack)
	// if p.conn == nil {
	// 	return fmt.Errorf("connection is not established")
	// }
	// if !p.conn.IsConnected() {
	// 	return fmt.Errorf("connection is not established")
	// }

	// select {
	// case p.data <- []*DataToPublish{
	// 	{
	// 		Topic: topic,
	// 		Data:  message,
	// 	},
	// }:
	// 	return nil
	// default:
	// 	return fmt.Errorf("data channel is full")
	// }
}

// IsReady checks if the publisher is ready to accept new messages
func (p *NatsPublisher) IsReady() bool {
	select {
	case <-p.done:
		return false
	case <-p.stopChan:
		return false
	default:
		return p.conn.IsConnected()
	}
}

func (p *NatsPublisher) Start() error {
	go p.monitorConnection()
	return nil
}

func (p *NatsPublisher) Stop() {
	close(p.stopChan)
	p.conn.Close()
}

func (p *NatsPublisher) monitorConnection() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case d := <-p.data:
			if p.conn == nil {
				fmt.Println("NATS not connected, skipping publish")
				continue
			}
			if !p.conn.IsConnected() {
				log.Println("Connection lost, stopping application")
				continue
			}
			bytes, err := json.Marshal(d)
			if err != nil {
				fmt.Printf("err json.Marshal: %v\n", err)
				continue
			}
			fmt.Println("Try publish to NATS")
			ack, err := p.js.Publish(d.Topic, bytes)
			if err != nil {
				fmt.Printf("err p.js.Publish: %s %v\n", d.Topic, err)
			}
			fmt.Printf("ack: %+v\n", ack)
		case <-p.stopChan:
			return
		case <-ticker.C:
			if p.connecting {
				continue
			}

			if p.conn == nil || !p.conn.IsConnected() {
				fmt.Println("Try reconnecting to NATS")
				p.connecting = true
				log.Printf("Connection lost, attempting to reconnect...")
				err := p.Connect()
				if err != nil {
					log.Printf("Failed to reconnect to NATS: %v", err)
				} else {
					log.Printf("Reconnected to NATS")
				}
				p.connecting = false
			} else {
				fmt.Println("NATS already connected")
			}
		}
	}
}
