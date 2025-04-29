package consumer

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

type NatsConsumer struct {
	conn          *nats.Conn
	js            nats.JetStreamContext
	topics        []string
	done          chan struct{}
	stop          chan struct{}
	data          chan []byte
	reconnected   chan bool
	connecting    bool
	url           string
	handler       MessageHandler
	subsriberName string
	//streamName     string
	connected      bool
	lock           sync.Mutex
	OnConnected    func(*nats.Conn)
	OnDisconnected func(*nats.Conn, error)
	OnReconnected  func(*nats.Conn)
	OnClosed       func(*nats.Conn)
}

func NewNatsConsumer(url string, username string, password string) *NatsConsumer {
	return &NatsConsumer{
		//streamName: streamName,
		//topics:     topics,
		done: make(chan struct{}),
		stop: make(chan struct{}),
		data: make(chan []byte, 10000),
		//reconnected:   make(chan bool),
		url:       url,
		connected: false,
	}
}

func (c *NatsConsumer) Start() error {
	go c.Connect()
	return nil
}

func (c *NatsConsumer) Stop() error {
	close(c.stop)
	return nil
}

func (p *NatsConsumer) CreateStream(nc *nats.Conn, streamName string, topics []string, replicas int) (*nats.StreamInfo, error) {
	fmt.Printf("create stream\n")
	if p.js == nil {
		js, err := nc.JetStream()
		if err != nil {
			fmt.Printf("err create JetStream Context: %v\n", err)
			return nil, err
		}
		p.js = js
	} else {
		fmt.Printf("js already created\n")
	}

	streamInfo, err := p.js.StreamInfo(streamName)
	if err == nil {
		fmt.Println("Stream already exists")
		return streamInfo, nil
	}
	fmt.Printf("stream not found, create stream\n")
	streamInfo, err = p.js.AddStream(&nats.StreamConfig{
		Name:      streamName,
		Subjects:  topics,
		Retention: nats.InterestPolicy, // Pesan disimpan berdasarkan batas waktu/jumlah
		MaxAge:    24 * time.Hour,      // Simpan pesan maksimal 24 jam
		Storage:   nats.FileStorage,    // Simpan ke disk (persisten)
		Replicas:  replicas,
	})
	return streamInfo, err

}
func (p *NatsConsumer) Connect() {
	opts := []nats.Option{
		nats.ReconnectWait(2 * time.Second),
		nats.MaxReconnects(-1), // Unlimited reconnects
		nats.MaxPingsOutstanding(5),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if p.OnDisconnected != nil {
				p.OnDisconnected(nc, err)
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			if p.OnReconnected != nil {
				p.OnReconnected(nc)
			}
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			log.Printf("NATS connection closed")
			if p.OnClosed != nil {
				p.OnClosed(nc)
			}
		}),
	}
	fmt.Printf("connect to cluster p.url: %v\n", p.url)

	nc, err := nats.Connect(p.url, opts...)
	if err == nil {
		if p.OnConnected != nil {
			p.OnConnected(nc)
		}
		/*for {
			_, err := p.CreateStream(nc, p.streamName, p.topics, 1)
			if err != nil {
				log.Printf("Failed to create stream: %v", err)
				if nc.IsClosed() {
					p.lock.Lock()
					if p.js != nil {
						p.js = nil
					}
					p.lock.Unlock()
					go func() {
						time.Sleep(1 * time.Second)
						p.Connect()
					}()
					return
				}
				time.Sleep(1 * time.Second)
				continue
			} else {
				fmt.Printf("stream %s created\n", p.streamName)
				break
			}
		}
		if p.js == nil {
			fmt.Printf("js is nil\n")
			return
		} else {
			fmt.Printf("js is not nil\n")
		}
		p.lock.Lock()
		p.conn = nc
		p.lock.Unlock()

		for _, topic := range p.topics {
			fmt.Printf("topic: %v\n", topic)
			go p.Subscribe(p.streamName, topic, p.subsriberName)
		}*/
		p.monitorConnection()
	} else {
		go func() {
			time.Sleep(1 * time.Second)
			p.Connect()
		}()
	}
}
func (c *NatsConsumer) Subscribe(streamName string, topic string, subsriberName string) {
	var sub *nats.Subscription
	c.lock.Lock()

	_, err := c.js.ConsumerInfo(streamName, subsriberName)

	if err == nil {
		fmt.Println("Consumer already exists")
	} else {
		fmt.Printf("consumer %s not found, create consumer\n", subsriberName)
		_, err := c.js.AddConsumer(streamName, &nats.ConsumerConfig{
			Name:          subsriberName,
			Durable:       subsriberName,
			DeliverPolicy: nats.DeliverLastPolicy,
			AckPolicy:     nats.AckExplicitPolicy,
			AckWait:       30 * time.Second,
			MaxAckPending: 1000,
		})

		if err != nil {
			if c.conn.IsClosed() {
				c.lock.Unlock()
				return
			}
		}
	}

	fmt.Println("subscribe streamName: ", streamName, " topic: ", topic, " subsriberName: ", subsriberName)
	for {
		if c.js == nil {
			c.lock.Unlock()
			return
		}
		s, err := c.js.PullSubscribe(topic, subsriberName,
			nats.BindStream(streamName),
			nats.Bind(streamName, subsriberName),
			nats.ManualAck(),
		)

		if err != nil {
			log.Printf("Failed to subscribe to topic: %v", err)
			c.lock.Unlock()
			return
		} else {
			sub = s
			break
		}
	}

	c.lock.Unlock()

	for {
		select {
		case <-c.stop:
			if sub != nil {
				sub.Unsubscribe()
			}
			return
		default:
		}

		c.lock.Lock()
		if c.conn == nil || c.conn.IsClosed() {
			c.lock.Unlock()
			fmt.Println("1. consumer not connected, c.conn == nil || c.conn.IsClosed()")
			return
		}

		for {
			msgs, err := sub.Fetch(100, nats.MaxWait(5*time.Second))
			if err != nil {
				if err == nats.ErrTimeout {
					continue
				}
				log.Printf("Fetch error: %v", err)
				time.Sleep(1 * time.Millisecond)
				continue
			}

			for _, msg := range msgs {
				if err := c.handler(topic, msg.Data); err != nil {
					msg.Nak()
					continue
				}
				if err := msg.Ack(); err != nil {
					log.Printf("Ack failed: %v", err)
				}
			}
		}
	}
}
func (p *NatsConsumer) Request(topic string, message []byte) error {
	msg, err := p.conn.Request(topic, message, 10*time.Second)
	if err != nil {
		return err
	}
	fmt.Printf("msg: %s\n", msg.Data)
	return nil
}
func (c *NatsConsumer) monitorConnection() {
	defer close(c.reconnected)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			c.lock.Lock()
			if c.connecting {
				c.lock.Unlock()
				continue
			}
			if c.conn.IsClosed() {
				go func() {
					time.Sleep(1 * time.Second)
					c.Connect()
				}()
			}
			c.lock.Unlock()
		}
	}
}
func (c *NatsConsumer) readMessages(topic string, consumerName string) {
	/*var sub *nats.Subscription
	retryDelay := time.Second

	for {
		select {
		case <-c.stop:
			if sub != nil {
				sub.Unsubscribe()
			}
			return
		default:
		}

		c.lock.Lock()
		if c.conn == nil || c.conn.IsClosed() || !c.connected {
			c.lock.Unlock()
			time.Sleep(retryDelay)
			retryDelay = minDuration(retryDelay*2, 10*time.Second)
			continue
		}

		if sub == nil {
			var err error
			sub, err = c.js.PullSubscribe(topic, consumerName,
				nats.Bind(c.streamName, consumerName),
				nats.ManualAck(),
				nats.DeliverLast(),
				nats.AckExplicit())
			if err != nil {
				c.lock.Unlock()
				log.Printf("Subscribe error: %v", err)
				time.Sleep(retryDelay)
				continue
			}
			retryDelay = time.Second
		}
		c.lock.Unlock()

		msg, err := sub.NextMsg(5 * time.Second)
		if err != nil {
			if err == nats.ErrTimeout {
				continue
			}
			log.Printf("NextMsg error: %v", err)
			sub = nil // Force recreate subscription
			time.Sleep(retryDelay)
			continue
		}

		if err := c.handler(topic, msg.Data); err != nil {
			msg.Nak()
			continue
		}
		msg.Ack()
	}*/
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
