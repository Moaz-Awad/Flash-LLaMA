package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config structure to hold LLaMA configuration
type Config struct {
	LLaMA struct {
		APIURL string `json:"api_url"`
	} `json:"llama"`
}

// LoadConfig loads the configuration from a JSON file
func LoadConfig(configFilePath string) (*Config, error) {
	file, err := os.Open(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("error opening config file: %v", err)
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return nil, fmt.Errorf("error decoding config file: %v", err)
	}

	return &config, nil
}
