package main

import (
	"fmt"
	"log"
	"time"

	"heatingcontrol/config"
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

	for i := 0; i < cfg.SensorsCount; i++ {
		s := sensor.NewSensor(cfg, *client, v.SetLevel)
		log.Printf("[sensor-%v] created\n", i)
		s.Start()
	}
	// TODO defer s.Stop()

	time.Sleep(100 * time.Second)
	fmt.Println("done.")
}
