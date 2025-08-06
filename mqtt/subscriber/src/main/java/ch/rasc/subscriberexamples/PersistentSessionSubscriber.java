package ch.rasc.subscriberexamples;

import java.time.LocalTime;
import java.util.concurrent.TimeUnit;

import org.eclipse.paho.client.mqttv3.IMqttMessageListener;
import org.eclipse.paho.client.mqttv3.MqttCallback;
import org.eclipse.paho.client.mqttv3.MqttClient;
import org.eclipse.paho.client.mqttv3.MqttConnectOptions;
import org.eclipse.paho.client.mqttv3.MqttException;
import org.eclipse.paho.client.mqttv3.MqttMessage;

import ch.rasc.mqttdemo.Sensor;

public class PersistentSessionSubscriber {
  public static void main(String[] args) throws MqttException, InterruptedException {
    String clientId = "persistent-subscriber-3";
    try (MqttClient client = new MqttClient("tcp://127.0.0.1:1883", clientId)) {
      run(client);
    }
  }

  private static void run(MqttClient client) throws MqttException, InterruptedException {
    MqttConnectOptions options = new MqttConnectOptions();
    options.setCleanSession(false); // Enable persistent session
    options.setKeepAliveInterval(60); // seconds

    client.setCallback(new MqttCallback() {
      @Override
      public void connectionLost(Throwable cause) {
        System.err.println("Connection lost: " + cause.getMessage());
      }

      @Override
      public void messageArrived(String topic,
          org.eclipse.paho.client.mqttv3.MqttMessage message) throws Exception {
        processMessage(topic, message);
      }

      @Override
      public void deliveryComplete(
          org.eclipse.paho.client.mqttv3.IMqttDeliveryToken token) {
        // Not used in subscriber, but required to implement MqttCallback
      }
    });

    var response = client.connectWithResult(options);

    if (!response.getSessionPresent()) {
      System.out.println("New session - subscribing to topic");
      // Only subscribe if this is a new session
      client.subscribe("sensors/outdoor/temperature", 1, messageListener());
    }
    else {
      System.out.println("Resuming existing session - subscription should be restored");
    }

    TimeUnit.SECONDS.sleep(120);
    client.disconnect();
  }

  private static IMqttMessageListener messageListener() {
    IMqttMessageListener messageHandler = (topic, message) -> {
      processMessage(topic, message);
    };
    return messageHandler;
  }

  private static void processMessage(String topic, MqttMessage message) {
    try {
      System.out.println("Received message on topic: " + topic);
      Sensor.SensorReading reading = Sensor.SensorReading.parseFrom(message.getPayload());
      System.out.printf(
          "[%s] Received temperature: %.1f°C (QoS: %d, retained: %s, dup: %s)%n",
          LocalTime.now(), reading.getValue(), message.getQos(), message.isRetained(),
          message.isDuplicate());
    }
    catch (Exception e) {
      System.err.printf("Failed to parse message: %s%n", e.getMessage());
    }
  }
}
