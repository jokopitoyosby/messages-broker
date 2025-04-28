package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/jokopitoyosby/messages-broker/consumer"
	"github.com/jokopitoyosby/messages-broker/models"
)

func main() {
	callbacks := map[string]func([]byte) error{

		"positions": func(body []byte) error {
			//fmt.Printf("body: %v\n", string(body))
			var position models.PositionData
			if err := json.Unmarshal(body, &position); err != nil {
				return fmt.Errorf("failed to unmarshal position: %v", err)
			}

			jsonData, err := json.MarshalIndent(position, "", "    ")
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(string(jsonData))
			return nil
		},
	}
	config, err := parseConfig("config.json")

	if err != nil {
		log.Fatalf("Failed to parse config: %v", err)
	}
	consumer.NewConsumer(config.Host, config.Port, config.Username, config.Password, "position_exchange", []string{"positions"}, callbacks).Start()
	select {}
}

// parseConfig membaca dan mem-parsing file konfigurasi JSON
func parseConfig(configFile string) (models.Config, error) {
	var config models.Config

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
