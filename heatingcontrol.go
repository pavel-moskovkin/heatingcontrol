package main

import (
	"log"
	"time"

	"heatingcontrol/config"
	"heatingcontrol/controller"
	"heatingcontrol/mosquito"
	"heatingcontrol/sensor"
	"heatingcontrol/valve"
)

func main() {
	cfg, err := config.ReadConfig()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Starting with Config:\n%+v\n", *cfg)

	client := mosquito.NewMqttClient(cfg)
	defer client.Stop()

	v := valve.NewValve(*client, cfg)
	v.Start()
	defer v.Stop()

	c := controller.NewController(cfg, *client, v)
	c.Start()
	defer c.Stop()

	sensors := make([]*sensor.Sensor, cfg.SensorsCount)
	for i := 0; i < cfg.SensorsCount; i++ {
		s := sensor.NewSensor(cfg, *client, c.SetValveLevel)
		log.Printf("[sensor-%v] created\n", i)
		s.Start()
		sensors[i] = s
	}

	defer func(sensors []*sensor.Sensor) {
		for _, s := range sensors {
			s.Stop()
		}
	}(sensors)

	select {
	case <-time.After(cfg.WorkTime):
		// some useful info here if needed
		log.Println("Stopping work after timeout reached")
	}

	log.Println("done.")
}
