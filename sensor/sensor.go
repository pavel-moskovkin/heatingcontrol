package sensor

import (
	"math/rand"
	"time"

	"heatingcontrol/mosquito"
)

var instances int

type Sensor struct {
	cli         mosquito.Client
	id          int
	temperature int
	valveLevel  chan uint
}

func NewSensor(client mosquito.Client, valveLevel chan uint) *Sensor {
	instances++
	return &Sensor{
		cli:        client,
		id:         instances,
		valveLevel: valveLevel,
	}
}

func (s *Sensor) Start() {
	go func(cli mosquito.Client) {
		rand.Seed(time.Now().UTC().UnixNano())
		s.temperature = rand.Intn(100)
		s.cli.PubData(s.id, s.temperature)

		for {
			select {
			case lvl := <-s.valveLevel:
				s.temperature = s.temperature + s.temperature*int(lvl)/100
				s.cli.PubData(s.id, s.temperature)
			}
		}
	}(s.cli)
}

func (s *Sensor) Stop() {
	close(s.valveLevel)
}
