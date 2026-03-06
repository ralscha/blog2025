package ch.rasc.springgrpc.client.service;

import ch.rasc.springgrpc.proto.AlertSubscriptionRequest;
import ch.rasc.springgrpc.proto.AnomalyAlert;
import ch.rasc.springgrpc.proto.IotAnomalyServiceGrpc;
import ch.rasc.springgrpc.proto.ReadingAssessment;
import ch.rasc.springgrpc.proto.SensorReadingRequest;
import io.grpc.stub.StreamObserver;
import java.time.Duration;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.TimeUnit;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.boot.CommandLineRunner;
import org.springframework.stereotype.Component;

@Component
public class IotAnomalyClientRunner implements CommandLineRunner {

  private static final Logger log = LoggerFactory.getLogger(IotAnomalyClientRunner.class);

  private final IotAnomalyServiceGrpc.IotAnomalyServiceBlockingStub blockingStub;
  private final IotAnomalyServiceGrpc.IotAnomalyServiceStub asyncStub;

  public IotAnomalyClientRunner(
      IotAnomalyServiceGrpc.IotAnomalyServiceBlockingStub blockingStub,
      IotAnomalyServiceGrpc.IotAnomalyServiceStub asyncStub) {
    this.blockingStub = blockingStub;
    this.asyncStub = asyncStub;
  }

  @Override
  public void run(String... args) throws Exception {
    runUnaryCheck();
    runAlertStream();
  }

  private void runUnaryCheck() {
    SensorReadingRequest request = SensorReadingRequest.newBuilder()
        .setSensorId("sensor-441")
        .setSiteId("warehouse-eu-1")
        .setMetricType("temperature_celsius")
        .setValue(91.4)
        .setBaseline(73.0)
        .setCapturedAtEpochMs(System.currentTimeMillis())
        .build();

    ReadingAssessment assessment = this.blockingStub
        .withDeadlineAfter(Duration.ofSeconds(3))
        .evaluateReading(request);

    log.info("Unary assessment -> sensor={} anomaly={} severity={} zScore={} summary={}",
        assessment.getSensorId(),
        assessment.getAnomaly(),
        assessment.getSeverity(),
        String.format("%.2f", assessment.getZScore()),
        assessment.getSummary());
  }

  private void runAlertStream() throws InterruptedException {
    CountDownLatch done = new CountDownLatch(1);

    AlertSubscriptionRequest streamRequest = AlertSubscriptionRequest.newBuilder()
        .setSiteId("warehouse-eu-1")
        .setDeviceGroup("freezers")
        .setMaxEvents(6)
        .build();

    this.asyncStub.subscribeAlerts(streamRequest, new StreamObserver<>() {
      @Override
      public void onNext(AnomalyAlert alert) {
        log.info("Stream alert -> id={} sensor={} severity={} observed={} threshold={} msg={}",
            alert.getAlertId(),
            alert.getSensorId(),
            alert.getSeverity(),
            String.format("%.2f", alert.getObservedValue()),
            String.format("%.2f", alert.getThreshold()),
            alert.getMessage());
      }

      @Override
      public void onError(Throwable throwable) {
        log.error("Alert stream failed", throwable);
        done.countDown();
      }

      @Override
      public void onCompleted() {
        log.info("Alert stream completed");
        done.countDown();
      }
    });

    if (!done.await(20, TimeUnit.SECONDS)) {
      log.warn("Alert stream timeout reached");
    }
  }
}
