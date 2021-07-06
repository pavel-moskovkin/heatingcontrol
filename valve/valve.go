package valve

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"heatingcontrol/config"
	"heatingcontrol/mosquito"
)

type Valve struct {
	cli          mosquito.Client
	cfg          *config.Config
	currentLevel *uint
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
		SetLevel:     make(chan uint, cfg.SensorsCount),
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
			}
		}
	}(v.cli.ValveListener)
}

func (v *Valve) ProcessData(d *mosquito.SensorData) {
	id, err := strconv.Atoi(strings.Split(d.SensorID, "-")[1])
	if err != nil {
		log.Fatalf("Error parsing sensor ID: %v", err.Error())
	}
	log.Printf("[valve] Receiced data: %+v\n", *d)

	i := d.Value
	v.sensorsCache[id] = &i

	// check if received all sensor data
	for _, val := range v.sensorsCache {
		if val == nil {
			return
		}
	}

	cfgTempLvl := v.cfg.TemperatureLevel
	time.Sleep(time.Second)

	// first try - setting valve openness equal to required temperature
	if v.currentLevel == nil {
		// TODO const
		log.Printf("[valve] Setting valve level to default %v\n", 50)
		v.setLevel(uint(50))
		return
	}

	var total int
	for _, val := range v.sensorsCache {
		total += *val
	}
	average := total / v.cfg.SensorsCount
	log.Printf("[valve] Average temperature %v\n", average)

	if average != cfgTempLvl {
		// todo move to const
		onePercent := float32(cfgTempLvl) / float32(100)
		percentOf := float32(average) / onePercent
		fmt.Printf("percentOf = %v\n", percentOf)
		if percentOf > 100 {
			percentOf = 100
		} else {
			percentOf = 100 - percentOf
		}
		fmt.Printf("percentOf = %v\n", percentOf)

		var setLevel uint
		changeOpenness := 50 * int(percentOf) / 100
		fmt.Printf("changeOpenness = %v\n", changeOpenness)

		if average > cfgTempLvl {
			// set valve openness lower 50% [0;50)
			setLevel = 50 - uint(changeOpenness)
		}
		if average < cfgTempLvl {
			// set valve openness higher 50% [50;100]
			setLevel = 50 + uint(changeOpenness)
		}
		fmt.Printf("setLevel = %v\n", setLevel)

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
	v.cli.PubValveLevel(value)
	v.resetCache()
}
