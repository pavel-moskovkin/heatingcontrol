package mosquito

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"heatingcontrol/config"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	TopicReadingsTemperature = "/readings/temperature"
	TopicSetValve            = "/actuators/room-1"
)

type SensorData struct {
	SensorID string `json:"sensorID"`
	Type     string `json:"type"`
	Value    int    `json:"value"`
}

type ValveLevel struct {
	Level uint `json:"level"`
}

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("Connected")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("Connect lost: %v", err)
}

type Client struct {
	mqtt.Client
	ValveListener chan SensorData
}

func NewMqttClient(cfg *config.Config) *Client {
	var broker = "broker.emqx.io"
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", broker, cfg.MqttPort))
	opts.SetClientID("go_mqtt_client")
	opts.SetUsername("emqx")
	opts.SetPassword("public")
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}
	return &Client{
		Client:        client,
		ValveListener: make(chan SensorData, 0),
	}
}

func (c *Client) PubValveLevel(value uint) {
	log.Printf("PubValveLevel: %v\n", value)
	sensorData := ValveLevel{
		Level: value,
	}
	payload, _ := json.Marshal(&sensorData)
	token := c.Publish(TopicSetValve, 0, false, string(payload))
	token.Wait()
	time.Sleep(time.Second)
}

func (c *Client) PubData(sensorID, value int) {
	sensorData := SensorData{
		SensorID: fmt.Sprintf("sensor-%v", sensorID),
		Type:     "temperature",
		Value:    value,
	}
	payload, _ := json.Marshal(&sensorData)
	token := c.Publish(TopicReadingsTemperature, 0, false, string(payload))
	token.Wait()
	log.Printf("Published %+v", sensorData)
	// time.Sleep(time.Second)
}

func (c *Client) SubData() {
	var handler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
		data := SensorData{}
		bytes := msg.Payload()
		if err := json.Unmarshal(bytes, &data); err != nil {
			log.Panic(err)
		}
		c.ValveListener <- data
	}
	token := c.Subscribe(TopicReadingsTemperature, 1, handler)
	token.Wait()
	fmt.Printf("Subscribed to topic %s\n", TopicReadingsTemperature)
}

func (c *Client) Stop() {
	close(c.ValveListener)
}
