package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Config struktur untuk konfigurasi koneksi RabbitMQ
type Config struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// PositionData struktur data posisi yang diterima
type PositionData struct {
	DeviceID  string    `json:"device_id"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Timestamp time.Time `json:"timestamp"`
	Speed     float64   `json:"speed"`
}

// RabbitMQConsumer struct untuk manajemen consumer
type RabbitMQConsumer struct {
	config     Config
	connection *amqp.Connection
	channel    *amqp.Channel
	done       chan bool
	exchange   string
	queues     []string
	callbacks  map[string]func([]byte) error
}

// NewRabbitMQConsumer membuat instance baru consumer
func NewRabbitMQConsumer(configFile string, exchange string, queues []string, callbacks map[string]func([]byte) error) (*RabbitMQConsumer, error) {
	config, err := parseConfig(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %v", err)
	}

	return &RabbitMQConsumer{
		config:    config,
		done:      make(chan bool),
		exchange:  exchange,
		queues:    queues,
		callbacks: callbacks,
	}, nil
}

// parseConfig membaca dan mem-parsing file konfigurasi JSON
func parseConfig(configFile string) (Config, error) {
	var config Config

	file, err := os.ReadFile(configFile)
	if err != nil {
		return config, fmt.Errorf("error reading config file: %v", err)
	}

	err = json.Unmarshal(file, &config)
	if err != nil {
		return config, fmt.Errorf("error parsing config JSON: %v", err)
	}

	if config.Host == "" {
		return config, errors.New("host is required in config")
	}
	if config.Port == 0 {
		config.Port = 5672 // default port
	}

	return config, nil
}

// connect membuat koneksi ke RabbitMQ
func (c *RabbitMQConsumer) connect() error {
	connString := fmt.Sprintf("amqp://%s:%s@%s:%d/",
		c.config.Username,
		c.config.Password,
		c.config.Host,
		c.config.Port)

	conn, err := amqp.Dial(connString)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to open a channel: %v", err)
	}

	// Deklarasikan exchange
	err = ch.ExchangeDeclare(
		c.exchange, // name
		"direct",   // type
		true,       // durable
		false,      // auto-deleted
		false,      // internal
		false,      // no-wait
		nil,        // arguments
	)
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to declare exchange: %v", err)
	}

	// Deklarasikan queues dan binding
	for _, queue := range c.queues {
		_, err = ch.QueueDeclare(
			queue, // name
			true,  // durable
			false, // delete when unused
			false, // exclusive
			false, // no-wait
			nil,   // arguments
		)
		if err != nil {
			conn.Close()
			return fmt.Errorf("failed to declare queue %s: %v", queue, err)
		}

		// Binding queue ke exchange dengan routing key sama dengan nama queue
		err = ch.QueueBind(
			queue,      // queue name
			queue,      // routing key
			c.exchange, // exchange
			false,
			nil,
		)
		if err != nil {
			conn.Close()
			return fmt.Errorf("failed to bind queue %s: %v", queue, err)
		}
	}

	c.connection = conn
	c.channel = ch

	return nil
}

// Start mengkonsumsi pesan dari RabbitMQ
func (c *RabbitMQConsumer) Start() error {
	// Coba koneksi pertama kali
	if err := c.connect(); err != nil {
		return err
	}

	// Mulai consumer untuk setiap queue
	for _, queue := range c.queues {
		msgs, err := c.channel.Consume(
			queue, // queue
			"",    // consumer
			false, // auto-ack (false karena kita ingin manual ack)
			false, // exclusive
			false, // no-local
			false, // no-wait
			nil,   // args
		)
		if err != nil {
			return fmt.Errorf("failed to register consumer for queue %s: %v", queue, err)
		}

		go c.handleMessages(queue, msgs)
	}

	go c.monitorConnection()

	return nil
}

// handleMessages memproses pesan yang diterima
func (c *RabbitMQConsumer) handleMessages(queue string, msgs <-chan amqp.Delivery) {
	for msg := range msgs {
		callback, ok := c.callbacks[queue]
		if !ok {
			log.Printf("No callback registered for queue %s", queue)
			continue
		}

		// Panggil callback function
		if err := callback(msg.Body); err != nil {
			log.Printf("Callback error for queue %s: %v. Message will not be acknowledged.", queue, err)
			// Tidak melakukan msg.Nack() karena kita ingin mencoba lagi nanti
			continue
		}

		// Ack pesan jika callback sukses
		if err := msg.Ack(false); err != nil {
			log.Printf("Failed to acknowledge message from queue %s: %v", queue, err)
		}
	}

	log.Printf("Message channel for queue %s closed. Attempting to reconnect...", queue)
	c.reconnect()
}

// monitorConnection memantau koneksi dan channel
func (c *RabbitMQConsumer) monitorConnection() {
	connClose := c.connection.NotifyClose(make(chan *amqp.Error))
	chanClose := c.channel.NotifyClose(make(chan *amqp.Error))

	select {
	case err := <-connClose:
		log.Printf("Connection closed: %v", err)
		c.reconnect()
	case err := <-chanClose:
		log.Printf("Channel closed: %v", err)
		c.reconnect()
	case <-c.done:
		return
	}
}

// reconnect mencoba melakukan koneksi ulang
func (c *RabbitMQConsumer) reconnect() {
	// Tutup koneksi yang ada jika masih terbuka
	if c.channel != nil {
		c.channel.Close()
	}
	if c.connection != nil {
		c.connection.Close()
	}

	// Implementasi exponential backoff untuk reconnection
	backoffTime := time.Second
	maxBackoff := 30 * time.Second

	for {
		select {
		case <-c.done:
			return
		default:
			log.Printf("Attempting to reconnect in %v...", backoffTime)
			time.Sleep(backoffTime)

			err := c.connect()
			if err == nil {
				log.Println("Reconnected successfully. Restarting consumers...")
				if err := c.Start(); err != nil {
					log.Printf("Failed to restart consumers after reconnect: %v", err)
					continue
				}
				return
			}

			log.Printf("Reconnection failed: %v", err)
			backoffTime *= 2
			if backoffTime > maxBackoff {
				backoffTime = maxBackoff
			}
		}
	}
}

// Stop menghentikan koneksi dan goroutine
func (c *RabbitMQConsumer) Stop() {
	close(c.done)

	if c.channel != nil {
		c.channel.Close()
	}
	if c.connection != nil {
		c.connection.Close()
	}
}

func main() {
	// Contoh callback functions untuk setiap queue
	callbacks := map[string]func([]byte) error{
		"geofenceservice": func(body []byte) error {
			var position PositionData
			if err := json.Unmarshal(body, &position); err != nil {
				return fmt.Errorf("failed to unmarshal position: %v", err)
			}

			// Proses geofence checking di sini
			log.Printf("[GeofenceService] Processing position for device %s: %.6f,%.6f",
				position.DeviceID, position.Latitude, position.Longitude)
			return nil
		},
		"posisionservice": func(body []byte) error {
			var position PositionData
			if err := json.Unmarshal(body, &position); err != nil {
				return fmt.Errorf("failed to unmarshal position: %v", err)
			}

			// Proses penyimpanan posisi di sini
			log.Printf("[PositionService] Storing position for device %s: %.6f,%.6f at %v",
				position.DeviceID, position.Latitude, position.Longitude, position.Timestamp)
			return nil
		},
	}

	// Buat consumer dengan file konfigurasi
	consumer, err := NewRabbitMQConsumer(
		"config.json",
		"position_exchange",
		[]string{"geofenceservice", "posisionservice"},
		callbacks,
	)
	if err != nil {
		log.Fatalf("Failed to create RabbitMQ consumer: %v", err)
	}

	// Mulai mengkonsumsi pesan
	if err := consumer.Start(); err != nil {
		log.Fatalf("Failed to start RabbitMQ consumer: %v", err)
	}

	log.Println("RabbitMQ consumer is running. Press CTRL+C to exit.")

	// Tunggu sinyal untuk keluar
	select {}
}
