package sensor

import (
	"log"
	"math/rand"
	"time"

	"heatingcontrol/config"
	"heatingcontrol/mosquito"
)

var (
	instances int
)

type Sensor struct {
	cfg         *config.Config
	cli         mosquito.Client
	id          int
	temperature int
	valveLevel  chan uint
}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func NewSensor(cfg *config.Config, client mosquito.Client, valveLevel chan uint) *Sensor {
	s := &Sensor{
		cfg:        cfg,
		cli:        client,
		id:         instances,
		valveLevel: valveLevel,
	}
	instances++
	return s
}

func (s *Sensor) Start() {
	s.temperature = 10 + rand.Intn(40)
	s.cli.PubData(s.id, s.temperature)

	go func(cli mosquito.Client) {
		for {
			select {
			case lvl := <-s.valveLevel:
				s.randomTemperatureChange()
				log.Printf("[sensor-%v] reveived new valve level: %v", s.id, lvl)
				changeOpenness := defineChangeTemperaturePercentage(lvl)
				s.temperature = s.temperature + s.temperature*changeOpenness/100

				s.cli.PubData(s.id, s.temperature)
				time.Sleep(time.Duration(s.cfg.SensorMeasureTimeout) * time.Second)
			}
		}
	}(s.cli)
}

func (s *Sensor) Stop() {
	close(s.valveLevel)
}

// randomly decrease area temperature by [0;2) degrees
func (s *Sensor) randomTemperatureChange() {
	t := rand.Intn(2)
	s.temperature = s.temperature - t
}

func defineChangeTemperaturePercentage(valveLevel uint) int {
	valveLevel = valveLevel - valveLevel%10
	switch valveLevel {
	case 0:
		return -50
	case 10:
		return -40
	case 20:
		return -30
	case 30:
		return -20
	case 40:
		return -10
	case 50:
		return 0
	case 60:
		return 10
	case 70:
		return 20
	case 80:
		return 30
	case 90:
		return 40
	case 100:
		return 50
	}
	return 0
}
