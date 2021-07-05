package valve

import (
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
			case lvl := <-v.SetLevel:
				v.cli.PubValveLevel(lvl)
			}
		}
	}(v.cli.ValveListener)
}

func (v *Valve) ProcessData(d *mosquito.SensorData) {
	id, err := strconv.Atoi(strings.Split(d.SensorID, "-")[1])
	if err != nil {
		log.Fatalf("Error parsing sensor ID: %v", err.Error())
	}
	log.Printf("[valve] Receiced data: %+v\n", d)

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
		log.Printf("[valve] Setting valve level to default %v\n", cfgTempLvl)
		set := uint(cfgTempLvl)
		v.currentLevel = &set
		// todo loop
		v.SetLevel <- set
		v.resetCache()
		return
	}

	var total int
	for _, val := range v.sensorsCache {
		total += *val
	}
	average := total / v.cfg.SensorsCount
	log.Printf("Average temperature %v\n", average)

	if average != cfgTempLvl {
		// todo move to const
		onePercent := float32(cfgTempLvl) / float32(100)
		percentOf := float32(average) / onePercent
		if percentOf > 100 {
			percentOf = 100
		}
		var setLevel uint
		changeOpenness := cfgTempLvl * int(percentOf) / 100
		if average > cfgTempLvl {
			// set valve openness lower
			setLevel = uint(cfgTempLvl) - uint(changeOpenness)
		}
		if average < cfgTempLvl {
			// set valve openness higher
			setLevel = uint(cfgTempLvl) + uint(changeOpenness)
		}

		log.Printf("Setting valve level from %v to %v\n", *v.currentLevel, setLevel)

		for i := 0; i < v.cfg.SensorsCount; i++ {
			v.SetLevel <- setLevel
		}

		v.currentLevel = &setLevel
		v.resetCache()
	}
}

func (v *Valve) resetCache() {
	for _, v := range v.sensorsCache {
		if v != nil {
			v = nil
		}
	}
}
