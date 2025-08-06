package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"publisherexamples/mqttdemo"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"google.golang.org/protobuf/proto"
)

func main() {
	opts := mqtt.NewClientOptions()
	opts.AddBroker("tcp://localhost:1883")
	opts.SetClientID("sensor-1")
	opts.SetKeepAlive(30 * time.Second)
	opts.SetPingTimeout(10 * time.Second)
	// Last Will and Testament (LWT) setup. qos 1, retain true
	opts.SetWill("sensors/1/status", "offline", 1, true)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to connect to broker: %v", token.Error())
	}
	defer client.Disconnect(250)

	// qos 1, retain true
	token := client.Publish("sensors/1/status", 1, true, "online")
	token.Wait()
	if token.Error() != nil {
		log.Fatalf("Failed to publish online status: %v", token.Error())
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	temperature := float32(18.0)

	fmt.Println("Publishing temperature data every 5 seconds. Press Ctrl+C for graceful shutdown.")

	for {
		select {
		case <-ticker.C:
			temperature += (float32(time.Now().Unix()%3) - 1) * 0.5
			reading := &mqttdemo.SensorReading{
				Value: temperature,
			}
			payload, err := proto.Marshal(reading)
			if err != nil {
				log.Printf("Failed to marshal protobuf: %v", err)
				continue
			}

			token := client.Publish("client1/temperature", 1, false, payload)
			token.Wait()
			if token.Error() != nil {
				log.Printf("Failed to publish temperature: %v", token.Error())
			} else {
				fmt.Printf("Published temperature: %.1f°C\n", temperature)
			}

		case <-signalChan:
			token := client.Publish("sensors/1/status", 1, true, "offline")
			token.Wait()
			if token.Error() != nil {
				log.Printf("Failed to publish offline status: %v", token.Error())
			} else {
				fmt.Println("Published offline status")
			}
			return
		}
	}
}
