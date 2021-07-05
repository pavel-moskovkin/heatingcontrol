package valve

import (
	"log"
	"strconv"
	"strings"

	"heatingcontrol/config"
	"heatingcontrol/mosquito"
)

type Valve struct {
	cli          mosquito.Client
	cfg          *config.Config
	SetLevel     chan uint
	sensorsCache map[int]*int
}

func NewValve(client mosquito.Client, cfg *config.Config) *Valve {
	sensors := make(map[int]*int, cfg.SensorsCount)
	for i := 0; i < cfg.SensorsCount; i++ {
		sensors[i] = nil
	}
	return &Valve{
		cli:          client,
		cfg:          cfg,
		SetLevel:     make(chan uint, 0),
		sensorsCache: sensors,
	}
}

func (v *Valve) Start() {
	v.cli.SubData()

	go func(ch chan mosquito.SensorData) {
		for {
			select {
			case d := <-ch:
				v.ProcessData(&d)
			case lvl := <-v.SetLevel:
				v.cli.PubValveLevel(lvl)
			}
		}
	}(v.cli.Ch)
}

func (v *Valve) ProcessData(d *mosquito.SensorData) {
	id, err := strconv.Atoi(strings.Split(d.SensorID, "-")[1])
	if err != nil {
		log.Fatalf("Error parsing sensor ID: %v", err.Error())
	}
	log.Printf("[valve] Receiced data: %+v\n", d)

	i := d.Value
	v.sensorsCache[id] = &i

	ready := true
	for _, val := range v.sensorsCache {
		if val == nil {
			ready = false
		}
	}

	if ready {
		var average int
		for _, val := range v.sensorsCache {
			average += *val
		}
		average = average / v.cfg.SensorsCount
		log.Printf("Average temperature %v\n", average)

		if average != v.cfg.TemperatureLevel {
			onePercent := float32(v.cfg.TemperatureLevel) / float32(100)
			percentOf := float32(average) / onePercent
			setLevel := (100 - int(percentOf)) / 2
			log.Printf("Settiinig valve level to %v\n", setLevel)
			v.SetLevel <- uint(setLevel)
		}
	}

}

func (v *Valve) resetCache() {
	for _, v := range v.sensorsCache {
		if v != nil {
			v = nil
		}
	}
}
