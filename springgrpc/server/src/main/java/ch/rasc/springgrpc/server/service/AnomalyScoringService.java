package ch.rasc.springgrpc.server.service;

import java.time.Instant;
import java.util.Locale;
import org.springframework.stereotype.Service;

@Service
public class AnomalyScoringService {

  public ReadingScore evaluate(
      String sensorId,
      String metricType,
      double observedValue,
      double baseline,
      long capturedAtEpochMs) {

    double safeBaseline = Math.abs(baseline) < 0.001 ? 1.0 : baseline;
    double normalBand = Math.max(0.5, Math.abs(safeBaseline) * 0.15);
    double zScore = (observedValue - safeBaseline) / normalBand;
    double absZScore = Math.abs(zScore);

    boolean anomaly = absZScore >= 1.8;
    String severity;
    long nextCheck;

    if (absZScore >= 3.5) {
      severity = "CRITICAL";
      nextCheck = 10;
    }
    else if (absZScore >= 2.5) {
      severity = "HIGH";
      nextCheck = 20;
    }
    else if (absZScore >= 1.8) {
      severity = "MEDIUM";
      nextCheck = 45;
    }
    else {
      severity = "NORMAL";
      nextCheck = 120;
    }

    String summary = String.format(
        Locale.ROOT,
        "Sensor %s (%s) at %s measured %.2f vs baseline %.2f",
        sensorId,
        metricType,
        Instant.ofEpochMilli(capturedAtEpochMs),
        observedValue,
        baseline);

    return new ReadingScore(anomaly, zScore, severity, summary, nextCheck);
  }

  public record ReadingScore(
      boolean anomaly,
      double zScore,
      String severity,
      String summary,
      long recommendedCheckAfterSeconds) {
  }
}
