package ch.rasc.subscriberexamples;

import java.util.concurrent.TimeUnit;

import org.eclipse.paho.client.mqttv3.IMqttMessageListener;
import org.eclipse.paho.client.mqttv3.MqttClient;
import org.eclipse.paho.client.mqttv3.MqttConnectOptions;
import org.eclipse.paho.client.mqttv3.MqttException;

import ch.rasc.mqttdemo.Sensor;

public class QoSSubscriber {

  public static void main(String[] args) throws MqttException, InterruptedException {
    String clientId = "subscriber-1";
    try (MqttClient client = new MqttClient("tcp://127.0.0.1:1883", clientId)) {
      run(client);
    }
  }

  private static void run(MqttClient client) throws MqttException, InterruptedException {
    MqttConnectOptions options = new MqttConnectOptions();
    options.setCleanSession(true);

    client.connect(options);

    IMqttMessageListener messageHandler = (topic, message) -> {
      try {
        System.out.println("Received message on topic: " + topic);
        Sensor.SensorReading reading = Sensor.SensorReading
            .parseFrom(message.getPayload());
        System.out.printf("Received temperature: %.1f°C with QoS %d (retained: %s)%n",
            reading.getValue(), message.getQos(), message.isRetained());
      }
      catch (Exception e) {
        System.err.printf("Failed to parse message: %s%n", e.getMessage());
      }
    };

    // Subscribe with different QoS levels
    System.out.println("Subscribing with QoS 0 (at most once)");
    client.subscribe("sensors/kitchen/temperature", 0, messageHandler);

    TimeUnit.SECONDS.sleep(20);

    System.out.println("Resubscribing with QoS 1 (at least once)");
    client.subscribe("sensors/kitchen/temperature", 1, messageHandler);

    TimeUnit.SECONDS.sleep(20);

    client.unsubscribe("sensors/kitchen/temperature");
    System.out.println("Resubscribing with QoS 2 (exactly once)");
    client.subscribe("sensors/kitchen/temperature", 2, messageHandler);

    TimeUnit.SECONDS.sleep(20);

    client.disconnect();
  }
}
