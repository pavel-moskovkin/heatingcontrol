package mosquito

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"heatingcontrol/config"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	TopicReadingsTemperature = "/readings/temperature"
	TopicSetValve            = "/actuators/room-1"
	TemperatureType          = "temperature"
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
	log.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	log.Println("Connected")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	log.Printf("[ERROR] Connect lost: %v", err)
}

type Client struct {
	mqtt.Client
	ValveListener chan SensorData
}

func NewMqttClient(cfg *config.Config) *Client {
	if cfg.Mqtt.DebugMode {
		mqtt.ERROR = log.New(os.Stdout, "[ERROR] ", 0)
		mqtt.CRITICAL = log.New(os.Stdout, "[CRIT] ", 0)
		mqtt.WARN = log.New(os.Stdout, "[WARN]  ", 0)
		mqtt.DEBUG = log.New(os.Stdout, "[DEBUG] ", 0)
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", cfg.Mqtt.Broker, cfg.Mqtt.Port))
	opts.SetClientID(cfg.Mqtt.ClientID)
	opts.SetUsername(cfg.Mqtt.Username)
	opts.SetPassword(cfg.Mqtt.Password)
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	opts.ConnectRetryInterval = time.Second
	opts.ConnectRetry = true
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("[mqtt] Error establishing connectiion: %v", token.Error())
	}

	return &Client{
		Client:        client,
		ValveListener: make(chan SensorData, 0),
	}
}

func (c *Client) PubValveLevel(value uint) {
	log.Printf("[mqtt] Publishing valve level: %v\n", value)
	valveLevel := ValveLevel{
		Level: value,
	}
	payload, err := json.Marshal(&valveLevel)
	if err != nil {
		log.Printf("[ERROR][mqtt] Error Marshaling json: %+v", valveLevel)
		return
	}
	token := c.Publish(TopicSetValve, 0, false, string(payload))
	token.Wait()
}

func (c *Client) PubData(sensorID, value int) {
	sensorData := SensorData{
		SensorID: fmt.Sprintf("sensor-%v", sensorID),
		Type:     TemperatureType,
		Value:    value,
	}
	payload, err := json.Marshal(&sensorData)
	if err != nil {
		log.Printf("[ERROR][mqtt] Error Marshaling json: %+v", sensorData)
		return
	}
	token := c.Publish(TopicReadingsTemperature, 0, false, string(payload))
	token.Wait()
}

func (c *Client) SubData() {
	var handler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
		data := SensorData{}
		bytes := msg.Payload()
		if err := json.Unmarshal(bytes, &data); err != nil {
			log.Printf("[ERROR][mqtt] Error Unmarshaling json: %+v", string(bytes))
			return
		}
		c.ValveListener <- data
	}
	token := c.Subscribe(TopicReadingsTemperature, 1, handler)
	token.Wait()
	log.Printf("[mqtt] Subscribed to topic %s\n", TopicReadingsTemperature)
}

func (c *Client) Stop() {
	close(c.ValveListener)
	c.Disconnect(250)
}
