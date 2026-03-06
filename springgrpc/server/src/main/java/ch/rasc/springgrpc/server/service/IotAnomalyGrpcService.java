package ch.rasc.springgrpc.server.service;

import ch.rasc.springgrpc.proto.AlertSubscriptionRequest;
import ch.rasc.springgrpc.proto.AnomalyAlert;
import ch.rasc.springgrpc.proto.IotAnomalyServiceGrpc;
import ch.rasc.springgrpc.proto.ReadingAssessment;
import ch.rasc.springgrpc.proto.SensorReadingRequest;
import io.grpc.stub.StreamObserver;
import java.time.Instant;
import java.util.Locale;
import java.util.UUID;
import java.util.concurrent.ThreadLocalRandom;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Service;

@Service
public class IotAnomalyGrpcService extends IotAnomalyServiceGrpc.IotAnomalyServiceImplBase {

  private static final Logger log = LoggerFactory.getLogger(IotAnomalyGrpcService.class);

  private final AnomalyScoringService anomalyScoringService;

  public IotAnomalyGrpcService(AnomalyScoringService anomalyScoringService) {
    this.anomalyScoringService = anomalyScoringService;
  }

  @Override
  public void evaluateReading(
      SensorReadingRequest request,
      StreamObserver<ReadingAssessment> responseObserver) {

    AnomalyScoringService.ReadingScore score = this.anomalyScoringService.evaluate(
        request.getSensorId(),
        request.getMetricType(),
        request.getValue(),
        request.getBaseline(),
        request.getCapturedAtEpochMs());

    ReadingAssessment assessment = ReadingAssessment.newBuilder()
        .setSensorId(request.getSensorId())
        .setAnomaly(score.anomaly())
        .setZScore(score.zScore())
        .setSeverity(score.severity())
        .setSummary(score.summary())
        .setRecommendedCheckAfterSeconds(score.recommendedCheckAfterSeconds())
        .build();

    log.info("Evaluated reading for sensor={} severity={} anomaly={}",
        request.getSensorId(), assessment.getSeverity(), assessment.getAnomaly());

    responseObserver.onNext(assessment);
    responseObserver.onCompleted();
  }

  @Override
  public void subscribeAlerts(
      AlertSubscriptionRequest request,
      StreamObserver<AnomalyAlert> responseObserver) {

    int maxEvents = request.getMaxEvents() <= 0 ? 8 : request.getMaxEvents();

    log.info("Starting alert stream: site={} group={} maxEvents={}",
        request.getSiteId(), request.getDeviceGroup(), maxEvents);

    try {
      for (int i = 1; i <= maxEvents; i++) {
        double threshold = 75.0;
        double observed = threshold + ThreadLocalRandom.current().nextDouble(-10.0, 22.0);
        String severity = observed >= 95.0 ? "CRITICAL" : (observed >= 85.0 ? "HIGH" : "MEDIUM");

        AnomalyAlert alert = AnomalyAlert.newBuilder()
            .setAlertId(UUID.randomUUID().toString())
            .setSensorId("sensor-" + String.format(Locale.ROOT, "%03d", i))
            .setSiteId(request.getSiteId())
            .setMetricType("temperature_celsius")
            .setObservedValue(observed)
            .setThreshold(threshold)
            .setSeverity(severity)
            .setMessage(String.format(Locale.ROOT,
                "%s anomaly: %.2f exceeds threshold %.2f at %s",
                severity,
                observed,
                threshold,
                Instant.now()))
            .setDetectedAtEpochMs(System.currentTimeMillis())
            .build();

        responseObserver.onNext(alert);
        Thread.sleep(700L);
      }

      responseObserver.onCompleted();
      log.info("Completed alert stream for site={}", request.getSiteId());
    }
    catch (InterruptedException ex) {
      Thread.currentThread().interrupt();
      responseObserver.onError(ex);
    }
  }
}
