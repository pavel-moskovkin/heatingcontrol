package sensor

import (
	"log"
	"math"
	"math/rand"
	"time"

	"heatingcontrol/config"
	"heatingcontrol/mosquito"
	"heatingcontrol/valve"
)

var (
	numInstances int
)

type Sensor struct {
	cfg         *config.Config
	cli         mosquito.Client
	id          int
	temperature float64
	done        chan struct{}
	valveLevel  chan uint // read to adjust current sensor temperature
}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func NewSensor(cfg *config.Config, client mosquito.Client, valveLevelChange chan uint) *Sensor {
	// init in range [0.1;100)
	temperature := float64(rand.Intn(1000)+1) / 10
	s := &Sensor{
		cfg:         cfg,
		cli:         client,
		id:          numInstances,
		temperature: temperature,
		done:        make(chan struct{}, 1),
		valveLevel:  valveLevelChange,
	}
	numInstances++
	return s
}

func (s *Sensor) Start() {
	s.cli.PubSensorData(s.id, s.temperature)

	go func(cli mosquito.Client) {
		for {
			select {
			case lvl := <-s.valveLevel:
				s.randomTemperatureChange()
				log.Printf("[sensor-%v] reveived new valve level: %v", s.id, lvl)
				changeOpenness := valve.DefineChangeTemperaturePercentage(lvl)
				s.temperature = s.temperature + s.temperature*float64(changeOpenness)/100
				// round float to 1 decimal place
				s.temperature = math.Round(s.temperature*10) / 10

				s.cli.PubSensorData(s.id, s.temperature)
				time.Sleep(s.cfg.SensorMeasureTimeout)
			case <-s.done:
				return
			}
		}
	}(s.cli)
}

func (s *Sensor) Stop() {
	s.done <- struct{}{}
	close(s.done)
}

// randomly decrease area temperature by [0;2) degrees
func (s *Sensor) randomTemperatureChange() {
	s.temperature = s.temperature - float64(rand.Intn(2))
}
