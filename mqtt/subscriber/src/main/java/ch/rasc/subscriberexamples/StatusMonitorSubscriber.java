package ch.rasc.subscriberexamples;

import java.time.LocalTime;
import java.util.concurrent.TimeUnit;

import org.eclipse.paho.client.mqttv3.IMqttMessageListener;
import org.eclipse.paho.client.mqttv3.MqttClient;
import org.eclipse.paho.client.mqttv3.MqttException;

public class StatusMonitorSubscriber {
  public static void main(String[] args) throws MqttException, InterruptedException {
    String clientId = "subscriber-1";
    try (MqttClient client = new MqttClient("tcp://127.0.0.1:1883", clientId)) {
      run(client);
    }
  }

  private static void run(MqttClient client) throws MqttException, InterruptedException {
    client.connect();

    IMqttMessageListener statusHandler = (topic, message) -> {
      String deviceId = extractDeviceIdFromTopic(topic);
      String status = new String(message.getPayload());
      String timestamp = LocalTime.now().toString();

      System.out.printf("[%s] Device %s is now %s (retained: %s)%n", timestamp, deviceId,
          status, message.isRetained());
    };

    IMqttMessageListener dataHandler = (topic, message) -> {
      String deviceId = extractDeviceIdFromTopic(topic);
      String timestamp = LocalTime.now().toString();

      System.out.printf("[%s] Received data from %s (size: %d bytes)%n", timestamp,
          deviceId, message.getPayload().length);
    };

    String statusTopic = "sensors/+/status";
    String dataTopic = "sensors/+/temperature";

    client.subscribe(statusTopic, statusHandler);
    client.subscribe(dataTopic, dataHandler);

    System.out.printf("Monitoring device status on: %s%n", statusTopic);
    System.out.printf("Monitoring device data on: %s%n", dataTopic);
    System.out.println();

    TimeUnit.MINUTES.sleep(5);
    client.disconnect();
  }

  private static String extractDeviceIdFromTopic(String topic) {
    String[] parts = topic.split("/");
    if (parts.length >= 2) {
      return parts[1]; // Extract device ID from sensors/{device}/...
    }
    return "unknown";
  }
}
