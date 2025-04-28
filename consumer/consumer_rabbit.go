package consumer

import (
	"fmt"
	"log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQConsumer struct {
	//config    models.Config
	host      string
	port      int
	username  string
	password  string
	exchange  string
	queues    []string
	callbacks map[string]func([]byte) error

	conn    *amqp.Connection
	channel *amqp.Channel
	mu      sync.Mutex

	done    chan struct{}
	wg      sync.WaitGroup
	isReady bool
	notify  chan struct{}
}

/*
config, err := parseConfig(configFile)

	if err != nil {
		log.Fatalf("Failed to parse config: %v", err)
	}
*/
func NewConsumer(host string, port int, username string, password string, exchange string, queues []string, callbacks map[string]func([]byte) error) *RabbitMQConsumer {

	return &RabbitMQConsumer{
		host:      host,
		port:      port,
		username:  username,
		password:  password,
		exchange:  exchange,
		queues:    queues,
		callbacks: callbacks,
		done:      make(chan struct{}),
		notify:    make(chan struct{}, 1),
	}
}

func (c *RabbitMQConsumer) Start() {
	c.wg.Add(1)
	go c.handleReconnect()
	// Tunggu koneksi pertama berhasil
	<-c.notify
}
func (c *RabbitMQConsumer) Stop() {
	close(c.done)
	c.wg.Wait()

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *RabbitMQConsumer) handleReconnect() {
	defer c.wg.Done()

	for {
		c.mu.Lock()
		c.isReady = false
		c.mu.Unlock()

		err := c.connect()
		if err != nil {
			log.Printf("Failed to connect. Retrying... Error: %v", err)
			select {
			case <-c.done:
				return
			case <-time.After(5 * time.Second):
				continue
			}
		}

		c.mu.Lock()
		c.isReady = true
		c.mu.Unlock()

		// Notify bahwa koneksi sudah ready
		select {
		case c.notify <- struct{}{}:
		default:
		}

		// Monitor koneksi
		closeChan := c.conn.NotifyClose(make(chan *amqp.Error))
		select {
		case err := <-closeChan:
			log.Printf("Connection closed: %v", err)
		case <-c.done:
			return
		}
	}
}

func (c *RabbitMQConsumer) connect() error {
	conn, err := amqp.DialConfig(
		fmt.Sprintf("amqp://%s:%s@%s:%d/",
			c.username,
			c.password,
			c.host,
			c.port),
		amqp.Config{
			Heartbeat: 10 * time.Second,
			Locale:    "en_US",
		},
	)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to open channel: %w", err)
	}

	// Set QoS
	err = ch.Qos(
		10,    // prefetch count
		0,     // prefetch size
		false, // global
	)
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	// err = ch.ExchangeDeclare(
	// 	c.exchange,
	// 	"direct",
	// 	true,
	// 	false,
	// 	false,
	// 	false,
	// 	nil,
	// )
	// if err != nil {
	// 	conn.Close()
	// 	return err
	// }

	for _, queue := range c.queues {
		_, err = ch.QueueDeclare(
			queue,
			true,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			conn.Close()
			return err
		}

		// err = ch.QueueBind(
		// 	queue,
		// 	queue,
		// 	c.exchange,
		// 	false,
		// 	nil,
		// )
		// if err != nil {
		// 	conn.Close()
		// 	return err
		// }

		msgs, err := ch.Consume(
			queue,
			"",
			false,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			conn.Close()
			return err
		}

		c.wg.Add(1)
		go c.handleMessages(queue, msgs)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.conn = conn
	c.channel = ch

	return nil
}

func (c *RabbitMQConsumer) handleMessages(queue string, msgs <-chan amqp.Delivery) {
	defer c.wg.Done()

	const maxRetries = 3
	for msg := range msgs {
		callback, ok := c.callbacks[queue]
		if !ok {
			log.Printf("No callback for queue %s", queue)
			msg.Nack(false, false) // Reject message without requeue
			continue
		}

		var err error
		for retry := 0; retry < maxRetries; retry++ {
			err = callback(msg.Body)
			if err == nil {
				break
			}
			log.Printf("Callback error (attempt %d/%d): %v", retry+1, maxRetries, err)
			time.Sleep(time.Duration(retry+1) * time.Second) // Exponential backoff
		}

		if err != nil {
			log.Printf("Failed to process message after %d retries: %v", maxRetries, err)
			msg.Nack(false, true) // Reject message with requeue
			continue
		}

		if err := msg.Ack(false); err != nil {
			log.Printf("Failed to ack message: %v", err)
		}
	}

	log.Printf("Message channel closed for queue %s", queue)
}
