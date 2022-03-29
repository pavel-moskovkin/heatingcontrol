package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	TemperatureLevel     float64       `yaml:"temperature_level"`
	SensorsCount         int           `yaml:"sensors_count"`
	SensorMeasureTimeout time.Duration `yaml:"sensor_measure_timeout"`
	WorkTime             time.Duration `yaml:"work_time"` // stop program work after this timeout
	Mqtt                 `yaml:"mqtt"`
}

type Mqtt struct {
	Port      int    `yaml:"port"`
	Broker    string `yaml:"broker"`
	ClientID  string `yaml:"client_id"`
	Username  string `yaml:"username"`
	Password  string `yaml:"password"`
	DebugMode bool   `yaml:"debug_mode"`
}

func ReadConfig(filepath string) (*Config, error) {
	var cfg Config
	f, err := os.Open(filepath)
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
