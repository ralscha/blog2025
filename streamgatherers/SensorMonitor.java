package streamgatherers;

import java.time.Instant;
import java.util.Spliterator;
import java.util.concurrent.BlockingQueue;
import java.util.concurrent.Executors;
import java.util.concurrent.LinkedBlockingQueue;
import java.util.concurrent.TimeUnit;
import java.util.function.Consumer;
import java.util.stream.Gatherer;
import java.util.stream.StreamSupport;

public class SensorMonitor {
  private static final BlockingQueue<SensorReading> queue = new LinkedBlockingQueue<>();

  public static void main(String[] args) {
    try (var executor = Executors.newVirtualThreadPerTaskExecutor()) {
      // Start producer thread
      executor.submit(() -> {
        long count = 0;
        while (count++ < 100) {
          // Generate fake sensor reading
          var reading = new SensorReading(Instant.now(), 20 + Math.random() * 10);

          try {
            queue.put(reading);
            System.out.println("[PRODUCER] Generated: " + reading);
            TimeUnit.SECONDS.sleep(1);
          }
          catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            throw new RuntimeException(e);
          }
        }
        try {
          queue.put(new SensorReading(Instant.now(), -1));
        }
        catch (InterruptedException e) {
          Thread.currentThread().interrupt();
          throw new RuntimeException(e);
        }
      });

      executor.submit(() -> {
        StreamSupport.stream(new QueueSpliterator(), false).gather(averagePerMinute())
            .forEach(avg -> System.out.println("[CONSUMER] " + avg));
      });
    }
  }

  static class State {
    long currentMinute = -1;
    double sum;
    int count;
  }

  static Gatherer<SensorReading, State, String> averagePerMinute() {
    return Gatherer.ofSequential(State::new, (state, reading, downstream) -> {
      long minuteBucket = reading.timestamp().getEpochSecond() / 60;

      if (state.currentMinute == -1) {
        state.currentMinute = minuteBucket;
      }

      if (minuteBucket != state.currentMinute) {
        double avg = state.sum / state.count;
        String result = "Minute %d: %.2f°C (%d samples)".formatted(state.currentMinute,
            avg, state.count);
        boolean canSendMore = downstream.push(result);

        if (!canSendMore) {
          return false;
        }

        // Reset
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
        downstream.push("Final average: %.2f°C".formatted(avg));
      }
    });
  }

  static class QueueSpliterator implements Spliterator<SensorReading> {
    @Override
    public boolean tryAdvance(Consumer<? super SensorReading> action) {
      try {
        SensorReading reading = queue.take();
        if (reading.temperature() == -1) {
          return false;
        }
        action.accept(reading);
        return true;
      }
      catch (InterruptedException e) {
        Thread.currentThread().interrupt();
        return false;
      }
    }

    @Override
    public Spliterator<SensorReading> trySplit() {
      return null;
    }

    @Override
    public long estimateSize() {
      return Long.MAX_VALUE;
    }

    @Override
    public int characteristics() {
      return Spliterator.ORDERED | Spliterator.NONNULL;
    }
  }
}