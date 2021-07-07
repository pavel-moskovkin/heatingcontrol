package sensor

import (
	"log"
	"math"
	"math/rand"
	"time"

	"heatingcontrol/config"
	"heatingcontrol/mosquito"
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

func NewSensor(cfg *config.Config, client mosquito.Client, valveLevel chan uint) *Sensor {
	s := &Sensor{
		cfg:        cfg,
		cli:        client,
		id:         numInstances,
		done:       make(chan struct{}, 1),
		valveLevel: valveLevel,
	}
	numInstances++
	return s
}

func (s *Sensor) Start() {
	// init in range [10.1;50.1)
	s.temperature = 10 + float64(rand.Intn(400)+1)/10
	s.cli.PubData(s.id, s.temperature)

	go func(cli mosquito.Client) {
		for {
			select {
			case lvl := <-s.valveLevel:
				s.randomTemperatureChange()
				log.Printf("[sensor-%v] reveived new valve level: %v", s.id, lvl)
				changeOpenness := defineChangeTemperaturePercentage(lvl)
				s.temperature = s.temperature + s.temperature*float64(changeOpenness)/100
				// round float to 1 decimal place
				s.temperature = math.Round(s.temperature*10) / 10

				s.cli.PubData(s.id, s.temperature)
				time.Sleep(s.cfg.SensorMeasureTimeout)
			case <-s.done:
				close(s.done)
				return
			}
		}
	}(s.cli)
}

func (s *Sensor) Stop() {
	s.done <- struct{}{}
}

// randomly decrease area temperature by [0;2) degrees
func (s *Sensor) randomTemperatureChange() {
	s.temperature = s.temperature - float64(rand.Intn(2))
}

// depending on valve openness level, return value representing a number on
// how many percents will be changed current temperature.
// Positive number means that the temperature will be increased, negative number means that
// the temperature will be increased on than percentage.
func defineChangeTemperaturePercentage(valveLevel uint) int {
	// round up to get higher value
	tmp := float64(valveLevel) / float64(10)
	valveLevel = uint(math.Round(tmp+0.4999) * 10)

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
	default:
		return 0
	}
}
