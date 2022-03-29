package controller

import (
	"errors"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"heatingcontrol/config"
	"heatingcontrol/mosquito"
	"heatingcontrol/valve"
)

type Controller struct {
	client                   mosquito.Client
	cfg                      *config.Config
	sensorsCache             map[int]*float64 // [sensor-id]value
	averageTemperatureLedger []float64        // information purposes only
	vlv                      *valve.Valve
	SetValveLevel            chan uint // used to send to the sensors the current valve level
	done                     chan struct{}
}

func NewController(cfg *config.Config, client mosquito.Client, valve *valve.Valve) *Controller {
	sensors := make(map[int]*float64, cfg.SensorsCount)
	for i := 0; i < cfg.SensorsCount; i++ {
		sensors[i] = nil
	}
	return &Controller{
		client:                   client,
		cfg:                      cfg,
		sensorsCache:             sensors,
		vlv:                      valve,
		averageTemperatureLedger: make([]float64, 0),
		SetValveLevel:            make(chan uint, cfg.SensorsCount),
		done:                     make(chan struct{}, 1),
	}
}

func (c *Controller) Start() {
	listener := c.client.SubSensorData()

	go func(cli mosquito.Client) {
		for {
			select {
			case d := <-listener:
				c.ProcessData(&d)
			case <-c.done:
				return
			}
		}
	}(c.client)
}

func (c *Controller) ProcessData(d *mosquito.SensorData) {
	if d.Type != mosquito.TemperatureType {
		log.Printf("[ctrl] Unknown SensorData message type: %v", d.Type)
		return
	}

	id, err := strconv.Atoi(strings.Split(d.SensorID, "-")[1])
	if err != nil {
		log.Printf("[ERROR][ctrl] Error parsing sensor ID from json: %+v :%v", d, err.Error())
	}
	log.Printf("[ctrl] Receiced sensor data: %+v\n", *d)

	c.sensorsCache[id] = &d.Value

	// check if received data from all sensors
	for _, val := range c.sensorsCache {
		if val == nil {
			return
		}
	}

	time.Sleep(time.Second)

	// first try - setting valve openness equal to required temperature
	cfgTempLvl := c.cfg.TemperatureLevel
	if c.vlv.CurrentLevel == nil {
		log.Printf("[ctrl] Setting valve level to default %v\n", valve.DefaultValveOpenness)
		err = c.setValveLevel(valve.DefaultValveOpenness)
		if err != nil {
			log.Printf("[ERROR][ctrl] unable to set valve level: %e\n", err)
		}
		return
	}

	var total float64
	for _, val := range c.sensorsCache {
		total += *val
	}
	average := total / float64(c.cfg.SensorsCount)
	// round float to 1 decimal place
	average = math.Round(average*10) / 10
	log.Printf("[ctrl] Average temperature %v\n", average)
	c.averageTemperatureLedger = append(c.averageTemperatureLedger, average)
	log.Printf("[ctrl] Average temperature history: %v\n", c.averageTemperatureLedger)

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
		changeOpenness := valve.DefaultValveOpenness * uint(percentDifference) / 100
		log.Printf("[DEBUG] changeOpenness = %v\n", changeOpenness)

		// decreasing temperature: set valve openness lower 50% [0;50]
		if average > cfgTempLvl {
			setLevel = valve.DefaultValveOpenness - changeOpenness
		}
		// increasing temperature: set valve openness higher 50% [50;100]
		if average < cfgTempLvl {
			setLevel = valve.DefaultValveOpenness + changeOpenness
		}
		log.Printf("[DEBUG] setLevel = %v\n", setLevel)

		log.Printf("[ctrl] Setting valve level from %v to %v\n", *c.vlv.CurrentLevel, setLevel)

		err = c.setValveLevel(setLevel)
		if err != nil {
			log.Printf("[ERROR][ctrl] unable to set valve level: %e\n", err)
		}
	} else {
		log.Printf("[ctrl] Successfully reached average temperature: %v\n", average)
		log.Printf("[ctrl] Remain same valve openness: %v\n", *c.vlv.CurrentLevel)
		err = c.setValveLevel(*c.vlv.CurrentLevel)
		if err != nil {
			log.Printf("[ERROR][ctrl] unable to set valve level: %e\n", err)
		}
	}
}

func (c *Controller) resetCache() {
	for k := range c.sensorsCache {
		c.sensorsCache[k] = nil
	}
}

func (c *Controller) setValveLevel(value uint) error {
	c.client.PubValveLevel(value)

LOOP:
	for {
		select {
		case <-time.After(c.cfg.SensorMeasureTimeout):
			return errors.New("timeout reached waiting for response from valve")
		case msg := <-c.vlv.MsgReceived:
			if !msg.Ok {
				return errors.New("[error placeholder]")
			}
			break LOOP
		}
	}

	c.vlv.CurrentLevel = &value
	// inform all the sensors about valve level change
	for i := 0; i < c.cfg.SensorsCount; i++ {
		c.SetValveLevel <- value
	}
	c.resetCache()
	return nil
}

func (c *Controller) Stop() {
	c.done <- struct{}{}
	close(c.done)
	close(c.SetValveLevel)
}
