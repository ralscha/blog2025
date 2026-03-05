package streamgatherers;

import java.time.Instant;
import java.util.List;
import java.util.Map;
import java.util.stream.Collectors;
import java.util.stream.Gatherer;

record SensorReading(Instant timestamp, double temperature) {
}

public class SensorAverage {
  public static void main(String[] args) {
    List<SensorReading> readings = List.of(
        new SensorReading(Instant.parse("2024-03-15T10:00:00Z"), 22.5),
        new SensorReading(Instant.parse("2024-03-15T10:00:30Z"), 23.1),
        new SensorReading(Instant.parse("2024-03-15T10:01:15Z"), 24.8),
        new SensorReading(Instant.parse("2024-03-15T10:01:45Z"), 25.3));

    Map<Long, Double> averages = readings.stream()
        .collect(Collectors.groupingBy(
            reading -> (long) (reading.timestamp().getEpochSecond() / 60),
            Collectors.averagingDouble(SensorReading::temperature)));

    System.out.println("Using collect(): " + averages);

    List<String> streamResults = readings.stream().gather(averagePerMinute()).toList();

    System.out.println("Using Gatherer: " + streamResults);
  }

  static Gatherer<SensorReading, ?, String> averagePerMinute() {
    class State {
      long currentMinute = -1;
      double sum;
      int count;
    }

    return Gatherer.ofSequential(State::new, (state, reading, downstream) -> {
      long minuteBucket = reading.timestamp().getEpochSecond() / 60;

      if (state.currentMinute == -1) {
        state.currentMinute = minuteBucket;
      }

      if (minuteBucket != state.currentMinute) {
        double avg = state.sum / state.count;
        boolean canSendMore = downstream
            .push("Minute %d: %.2f°C".formatted(state.currentMinute, avg));
        if (!canSendMore) {
          return false;
        }

        state.currentMinute = minuteBucket;
        state.sum = 0;
        state.count = 0;
      }

      state.sum += reading.temperature();
      state.count++;
      return true;
    }, (state, downstream) -> {
      if (state.count > 0) {
        double avg = state.sum / state.count;
        downstream.push("Minute %d: %.2f°C".formatted(state.currentMinute, avg));
      }
    });
  }
}