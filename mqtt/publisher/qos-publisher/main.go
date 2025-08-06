package main

import (
	"fmt"
	"log"
	"publisherexamples/mqttdemo"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"google.golang.org/protobuf/proto"
)

func main() {
	opts := mqtt.NewClientOptions()
	opts.AddBroker("tcp://localhost:1883")
	opts.SetClientID("sensor-1")
	opts.SetCleanSession(true)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to connect to broker: %v", token.Error())
	}
	defer client.Disconnect(250)

	fmt.Println("Connected to broker")

	qosLevels := []byte{0, 1, 2}
	temperatures := []float32{21.0, 21.5, 22.0}

	for i, qos := range qosLevels {
		reading := &mqttdemo.SensorReading{
			Value: temperatures[i],
		}

		payload, err := proto.Marshal(reading)
		if err != nil {
			log.Fatalf("Failed to marshal protobuf: %v", err)
		}

		token := client.Publish("sensors/kitchen/temperature", qos, false, payload)
		token.Wait()
		if token.Error() != nil {
			log.Fatalf("Failed to publish message with QoS %d: %v", qos, token.Error())
		}

		fmt.Printf("Published temperature reading: %.1f°C with QoS %d\n", reading.Value, qos)
		time.Sleep(1 * time.Second)
	}
}
