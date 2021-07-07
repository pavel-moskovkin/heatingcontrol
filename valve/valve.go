package valve

import (
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"heatingcontrol/config"
	"heatingcontrol/mosquito"
)

const (
	DefaultValveOpenness uint = 50
)

type Valve struct {
	client                   mosquito.Client
	cfg                      *config.Config
	currentLevel             *uint
	SetLevel                 chan uint // used to indicate the sensors the current valve level
	sensorsCache             map[int]*int
	averageTemperatureLedger []float64 // information purposes only
	done                     chan struct{}
}

func NewValve(client mosquito.Client, cfg *config.Config) *Valve {
	sensors := make(map[int]*int, cfg.SensorsCount)
	for i := 0; i < cfg.SensorsCount; i++ {
		sensors[i] = nil
	}
	return &Valve{
		client:                   client,
		cfg:                      cfg,
		SetLevel:                 make(chan uint, cfg.SensorsCount),
		sensorsCache:             sensors,
		averageTemperatureLedger: make([]float64, 0),
		done:                     make(chan struct{}, 1),
	}
}

func (v *Valve) Start() {
	v.client.SubData()

	go func(ch chan mosquito.SensorData) {
		for {
			select {
			case d := <-ch:
				v.ProcessData(&d)
			case <-v.done:
				close(v.done)
				return
			}
		}
	}(v.client.ValveListener)
}

func (v *Valve) ProcessData(d *mosquito.SensorData) {
	if d.Type != mosquito.TemperatureType {
		log.Printf("[valve] Unknown SensorData message type: %v", d.Type)
		return
	}

	id, err := strconv.Atoi(strings.Split(d.SensorID, "-")[1])
	if err != nil {
		log.Printf("[ERROR][valve] Error parsing sensor ID from json: %+v :%v", d, err.Error())
	}
	log.Printf("[valve] Receiced sensor data: %+v\n", *d)

	v.sensorsCache[id] = &d.Value

	// check if received data from all sensors
	for _, val := range v.sensorsCache {
		if val == nil {
			return
		}
	}

	time.Sleep(time.Second)

	// first try - setting valve openness equal to required temperature
	cfgTempLvl := v.cfg.TemperatureLevel
	if v.currentLevel == nil {
		log.Printf("[valve] Setting valve level to default %v\n", DefaultValveOpenness)
		v.setLevel(DefaultValveOpenness)
		return
	}

	var total int
	for _, val := range v.sensorsCache {
		total += *val
	}
	average := float64(total) / float64(v.cfg.SensorsCount)
	// round float to 1 decimal place
	average = math.Round(average*10) / 10
	log.Printf("[valve] Average temperature %v\n", average)
	v.averageTemperatureLedger = append(v.averageTemperatureLedger, average)
	log.Printf("[valve] Average temperature history: %v\n", v.averageTemperatureLedger)

	if average != cfgTempLvl {
		onePercent := cfgTempLvl / float64(100)
		percentOf := average / onePercent
		log.Printf("[DEBUG] percentOf = %v\n", percentOf)

		var percentDifference float64
		if percentOf > 100 {
			if percentOf > 200 {
				percentOf = 200
			}
			percentDifference = percentOf - 100
		} else {
			percentDifference = 100 - percentOf
		}
		log.Printf("[DEBUG] percentDifference = %v\n", percentDifference)

		var setLevel uint
		changeOpenness := DefaultValveOpenness * uint(percentDifference) / 100
		log.Printf("[DEBUG] changeOpenness = %v\n", changeOpenness)

		// decreasing temperature: set valve openness lower 50% [0;50]
		if average > cfgTempLvl {
			setLevel = DefaultValveOpenness - changeOpenness
		}
		// increasing temperature: set valve openness higher 50% [50;100]
		if average < cfgTempLvl {
			setLevel = DefaultValveOpenness + changeOpenness
		}
		log.Printf("[DEBUG] setLevel = %v\n", setLevel)

		log.Printf("[valve] Setting valve level from %v to %v\n", *v.currentLevel, setLevel)

		v.setLevel(setLevel)
	} else {
		log.Printf("[valve] Successfully reached average temperature: %v\n", average)
		log.Printf("[valve] Remain same valve openness: %v\n", *v.currentLevel)
		v.setLevel(*v.currentLevel)
	}
}

func (v *Valve) resetCache() {
	for k := range v.sensorsCache {
		v.sensorsCache[k] = nil
	}
}

func (v *Valve) setLevel(value uint) {
	for i := 0; i < v.cfg.SensorsCount; i++ {
		v.SetLevel <- value
	}
	v.currentLevel = &value
	v.client.PubValveLevel(value)
	v.resetCache()
}

func (v *Valve) Stop() {
	v.done <- struct{}{}
	close(v.SetLevel)
}
