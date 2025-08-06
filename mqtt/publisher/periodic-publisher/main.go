package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"
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
	opts.SetKeepAlive(60 * time.Second)
	opts.SetPingTimeout(10 * time.Second)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to connect to broker: %v", token.Error())
	}
	defer client.Disconnect(250)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	baseTemperature := float32(15.0)
	startTime := time.Now()

	fmt.Println("Publishing simulated outdoor temperature data every 10 seconds. Press Ctrl+C to stop.")

	for {
		select {
		case <-ticker.C:
			elapsed := time.Since(startTime).Hours()
			variation := float32(10.0 * math.Sin(2*math.Pi*elapsed/24.0))
			noise := (rand.Float32() - 0.5) * 2.0
			temperature := baseTemperature + variation + noise

			reading := &mqttdemo.SensorReading{
				Value: temperature,
			}

			payload, err := proto.Marshal(reading)
			if err != nil {
				log.Printf("Failed to marshal protobuf: %v", err)
				continue
			}

			// qos 1, retain false
			token := client.Publish("sensors/outdoor/temperature", 1, false, payload)
			token.Wait()
			if token.Error() != nil {
				log.Printf("Failed to publish temperature: %v", token.Error())
			} else {
				fmt.Printf("Published outdoor temperature: %.1f°C (elapsed: %.1fh)\n", temperature, elapsed)
			}

		case <-signalChan:
			fmt.Println("\nReceived shutdown signal. Disconnecting...")
			return
		}
	}
}
