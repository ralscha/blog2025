package main

import (
	"fmt"
	"log"
	"publisherexamples/mqttdemo"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"google.golang.org/protobuf/proto"
)

func main() {
	opts := mqtt.NewClientOptions()
	opts.AddBroker("tcp://localhost:1883")
	opts.SetClientID("sensor-1")
	opts.SetUsername("exampleUser")
	opts.SetPassword("password123")

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to connect to broker: %v", token.Error())
	}
	defer client.Disconnect(250)

	reading := &mqttdemo.SensorReading{
		Value: 22.5,
	}

	payload, err := proto.Marshal(reading)
	if err != nil {
		log.Fatalf("Failed to marshal protobuf: %v", err)
	}

	// 0 is the QoS level, false means no retained message
	token := client.Publish("sensors/living_room/temperature", 0, false, payload)
	token.Wait()
	if token.Error() != nil {
		log.Fatalf("Failed to publish message: %v", token.Error())
	}

	fmt.Printf("Published temperature reading: %.1f°C to topic: %s\n", reading.Value, "sensors/living_room/temperature")
}
