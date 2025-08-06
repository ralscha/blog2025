package ch.rasc.subscriberexamples;

import java.util.concurrent.TimeUnit;

import org.eclipse.paho.client.mqttv3.IMqttMessageListener;
import org.eclipse.paho.client.mqttv3.MqttClient;
import org.eclipse.paho.client.mqttv3.MqttException;

import ch.rasc.mqttdemo.Sensor;

public class WildcardSubscriber {

  public static void main(String[] args) throws MqttException, InterruptedException {
    String clientId = "subscriber-1";
    try (MqttClient client = new MqttClient("tcp://127.0.0.1:1883", clientId)) {
      run(client);
    }
  }

  private static void run(MqttClient client) throws MqttException, InterruptedException {
    client.connect();

    // Message handler for parsing protobuf messages
    IMqttMessageListener messageHandler = (topic, message) -> {
      try {
        Sensor.SensorReading reading = Sensor.SensorReading
            .parseFrom(message.getPayload());
        String location = extractLocationFromTopic(topic);
        System.out.printf("Received reading: %.1f°C from %s (topic: %s)%n",
            reading.getValue(), location, topic);
      }
      catch (Exception e) {
        System.err.printf("Failed to parse message from topic %s: %s%n", topic,
            e.getMessage());
      }
    };

    String singleLevelTopic = "sensors/+/temperature";
    String multiLevelTopic = "sensors/#";

    client.subscribe(singleLevelTopic, messageHandler);
    System.out.printf("Subscribed to single-level wildcard: %s%n", singleLevelTopic);

    client.subscribe(multiLevelTopic, messageHandler);
    System.out.printf("Subscribed to multi-level wildcard: %s%n", multiLevelTopic);

    TimeUnit.SECONDS.sleep(60);
    client.disconnect();
  }

  private static String extractLocationFromTopic(String topic) {
    String[] parts = topic.split("/");
    if (parts.length >= 2) {
      return parts[1];
    }
    return "unknown";
  }
}
