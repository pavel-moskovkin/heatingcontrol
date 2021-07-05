package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	TemperatureLevel int `yaml:"temperature_level"`
	SensorsCount     int `yaml:"sensors_count"`
	MqttPort         int `yaml:"mqtt_port"`
}

func ReadConfig() (*Config, error) {
	var cfg Config
	f, err := os.Open("config.yaml")
	if err != nil {
		msg := fmt.Sprintf("unable to read Config: %v", err.Error())
		return nil, errors.New(msg)
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		msg := fmt.Sprintf("unable to unmarshal Config: %v", err.Error())
		return nil, errors.New(msg)
	}

	return &cfg, nil
}
