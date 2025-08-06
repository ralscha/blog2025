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

	reading := &mqttdemo.SensorReading{
		Value: 19.8,
	}

	payload, err := proto.Marshal(reading)
	if err != nil {
		log.Fatalf("Failed to marshal protobuf: %v", err)
	}

	// qos 1, retain true
	token := client.Publish("sensors/bedroom/temperature", 1, true, payload)
	token.Wait()
	if token.Error() != nil {
		log.Fatalf("Failed to publish message: %v", token.Error())
	}
	fmt.Printf("Published retained temperature reading: %.1f°C to topic: %s\n", reading.Value, "sensors/bedroom/temperature")

	// send another retained message after a delay
	time.Sleep(10 * time.Second)
	reading.Value = 20.5
	payload, err = proto.Marshal(reading)
	if err != nil {
		log.Fatalf("Failed to marshal protobuf: %v", err)
	}
	// qos 1, retain true
	token = client.Publish("sensors/bedroom/temperature", 1, true, payload)
	if token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to publish retained message: %v", token.Error())
	}
	fmt.Printf("Published updated retained temperature reading: %.1f°C to topic: %s\n", reading.Value, "sensors/bedroom/temperature")

	time.Sleep(30 * time.Second)

	token = client.Publish("sensors/bedroom/temperature", 1, true, []byte{})
	token.Wait()
	if token.Error() != nil {
		log.Fatalf("Failed to clear retained message: %v", token.Error())
	}
	fmt.Println("Cleared retained message from topic")
}
