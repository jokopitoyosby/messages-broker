package publisher

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Config struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type RabbitMQPublisher struct {
	exchange string
	queues   []string

	conn    *amqp.Connection
	channel *amqp.Channel
	mu      sync.Mutex

	done     chan struct{}
	wg       sync.WaitGroup
	isReady  bool
	notify   chan struct{}
	Username string
	Password string
	Host     string
	Port     int
}

func NewPublisher(username, password, host string, port int, exchange string, queues []string) *RabbitMQPublisher {

	return &RabbitMQPublisher{
		Username: username,
		Password: password,
		Host:     host,
		Port:     port,
		exchange: exchange,
		queues:   queues,
		done:     make(chan struct{}),
		notify:   make(chan struct{}, 1),
	}
}

func (p *RabbitMQPublisher) Start() {
	p.wg.Add(1)
	go p.handleReconnect()

	// Tunggu koneksi pertama berhasil
	<-p.notify
}
func (p *RabbitMQPublisher) Stop() {
	close(p.done)
	p.wg.Wait()

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.channel != nil {
		p.channel.Close()
	}
	if p.conn != nil {
		p.conn.Close()
	}
}
func (p *RabbitMQPublisher) handleReconnect() {
	defer p.wg.Done()

	for {
		p.mu.Lock()
		p.isReady = false
		p.mu.Unlock()

		err := p.connect()
		if err != nil {
			log.Printf("Failed to connect. Retrying... Error: %v", err)
			select {
			case <-p.done:
				return
			case <-time.After(5 * time.Second):
				continue
			}
		}

		p.mu.Lock()
		p.isReady = true
		p.mu.Unlock()

		// Notify bahwa koneksi sudah ready
		select {
		case p.notify <- struct{}{}:
		default:
		}

		// Monitor koneksi
		closeChan := p.conn.NotifyClose(make(chan *amqp.Error))
		select {
		case err := <-closeChan:
			log.Printf("Connection closed: %v", err)
		case <-p.done:
			return
		}
	}
}

func (p *RabbitMQPublisher) connect() error {
	conn, err := amqp.DialConfig(
		fmt.Sprintf("amqp://%s:%s@%s:%d/",
			p.Username,
			p.Password,
			p.Host,
			p.Port),
		amqp.Config{
			Heartbeat: 10 * time.Second,
			Locale:    "en_US",
		},
	)
	if err != nil {
		return err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return err
	}

	err = ch.ExchangeDeclare(
		p.exchange,
		"direct",
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

	for _, queue := range p.queues {
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

		err = ch.QueueBind(
			queue,
			queue,
			p.exchange,
			false,
			nil,
		)
		if err != nil {
			conn.Close()
			return err
		}
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.conn = conn
	p.channel = ch

	return nil
}

func (p *RabbitMQPublisher) Publish(routingKey string, body []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isReady || p.channel == nil {
		return fmt.Errorf("connection not ready")
	}

	return p.channel.Publish(
		p.exchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
		},
	)
}

// Fungsi parseConfig dan main sama seperti sebelumnya

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
