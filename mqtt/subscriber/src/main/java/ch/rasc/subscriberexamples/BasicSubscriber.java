package ch.rasc.subscriberexamples;

import java.util.concurrent.TimeUnit;

import org.eclipse.paho.client.mqttv3.MqttClient;
import org.eclipse.paho.client.mqttv3.MqttException;

import ch.rasc.mqttdemo.Sensor;

public class BasicSubscriber {
  public static void main(String[] args) throws MqttException, InterruptedException {
    String clientId = "subscriber-1";
    try (MqttClient client = new MqttClient("tcp://127.0.0.1:1883", clientId)) {
      run(client);
    }
  }

  private static void run(MqttClient client) throws MqttException, InterruptedException {
    client.connect();

    client.subscribe("sensors/living_room/temperature", (topic, message) -> {
      try {
        Sensor.SensorReading reading = Sensor.SensorReading
            .parseFrom(message.getPayload());
        System.out.printf("Received temperature reading: %.1f°C from topic: %s%n",
            reading.getValue(), topic);
      }
      catch (Exception e) {
        System.err.printf("Failed to parse message: %s%n", e.getMessage());
      }
    });

    TimeUnit.SECONDS.sleep(30);
    client.disconnect();
  }
}
