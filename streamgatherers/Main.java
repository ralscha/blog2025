package streamgatherers;

import java.math.BigDecimal;
import java.net.URI;
import java.util.ArrayList;
import java.util.List;
import java.util.Optional;
import java.util.concurrent.ThreadLocalRandom;
import java.util.concurrent.atomic.AtomicLong;
import java.util.stream.Collector;
import java.util.stream.Gatherer;
import java.util.stream.Gatherers;
import java.util.stream.IntStream;
import java.util.stream.Stream;

public class Main {
  public static void main(String[] args) {
    customCollectorTerminalExample();
    gatherMethodSignatureExample();
    builtInExactExamples();
    window();
    windowSliding();
    scan();
    fold();
    mapConcurrent();
    custom1();
    custom2();
    custom3();
    custom4();
    custom5();
    custom6();
    custom7();
    quickRefresher();
    sequentialTemplateDemo();
    greedyVsNonGreedyDemo();
    compositionDemo();
  }

  private static void customCollectorTerminalExample() {
    Collector<String, ?, List<String>> toUpperCaseList = Collector.of(ArrayList::new,
        (list, s) -> {
          if (!s.isBlank()) {
            list.add(s.toUpperCase());
          }
        }, (left, right) -> {
          left.addAll(right);
          return left;
        });

    List<String> result = Stream.of("the", "", "fox", "jumps")
        .filter(s -> !s.isBlank())
        .map(String::toUpperCase)
        .collect(toUpperCaseList);

    System.out.println("custom collector result: " + result);
  }

  private static void gatherMethodSignatureExample() {
    Stream<Integer> source = Stream.of(1, 2, 3, 4);
    List<Integer> gathered = gather(source, Gatherers.scan(() -> 0, Integer::sum))
        .toList();
    System.out.println("gather signature example: " + gathered);
  }

  private static <T, R> Stream<R> gather(Stream<T> source,
      Gatherer<? super T, ?, R> gatherer) {
    return source.gather(gatherer);
  }

  private static void builtInExactExamples() {
    List<List<Integer>> fixed = Stream.of(1, 2, 3, 4, 5, 6, 7)
        .gather(Gatherers.windowFixed(3))
        .toList();
    System.out.println("windowFixed: " + fixed);

    List<List<Integer>> sliding = Stream.of(1, 2, 3, 4, 5)
        .gather(Gatherers.windowSliding(3))
        .toList();
    System.out.println("windowSliding: " + sliding);

    Optional<Integer> folded = Stream.of(1, 2, 3, 4)
        .gather(Gatherers.fold(() -> 0, Integer::sum))
        .findFirst();
    System.out.println("fold: " + folded);

    List<Integer> scanned = Stream.of(1, 2, 3, 4)
        .gather(Gatherers.scan(() -> 0, Integer::sum))
        .toList();
    System.out.println("scan: " + scanned);
  }

  private static void window() {
    List<Integer> source = List.of(1, 2, 3, 4, 5, 6, 7, 8, 9, 10);
    List<List<Integer>> result = source.stream().gather(Gatherers.windowFixed(3))
        .toList();

    System.out.println(result);
	// [[1, 2, 3], [4, 5, 6], [7, 8, 9], [10]]
  }

  record Reading(float value, int id) {
  }

  private static boolean isSuspicious(Reading r1, Reading r2) {
    return Math.abs(r1.value() - r2.value()) > 0.5;
  }

  private static void windowSliding() {
    List<Reading> readings = List.of(new Reading(1.0f, 1), new Reading(1.4f, 2),
        new Reading(1.5f, 3), new Reading(1.0f, 4), new Reading(2.0f, 5),
        new Reading(1.5f, 6), new Reading(1.0f, 7));

    List<List<Reading>> suspicious = readings.stream().gather(Gatherers.windowSliding(2))
        .filter(window -> isSuspicious(window.get(0), window.get(1))).toList();

    System.out.println(suspicious);
  }

  private static void scan() {
    record RadiationReading(double microSieverts) {
    }
    record Exposure(double currentValue, double total, int count) {
    }
    List<RadiationReading> readings = List.of(new RadiationReading(0.1),
        new RadiationReading(0.2), new RadiationReading(0.1), new RadiationReading(0.7),
        new RadiationReading(0.3));

    List<Exposure> cumulativeExposure = readings.stream().gather(Gatherers.scan(
        () -> new Exposure(0, 0, 0),
        (exposureTotal, reading) -> new Exposure(reading.microSieverts(),
            exposureTotal.total() + reading.microSieverts(), exposureTotal.count() + 1)))
        .toList();

    System.out.println(cumulativeExposure);
  }

  private static void fold() {
    List<Integer> numbers = List.of(1, 2, 3, 4, 5);
    int sum = numbers.stream().gather(Gatherers.fold(() -> 0, Integer::sum)).toList()
        .get(0);

    System.out.println(sum);

    record Account(BigDecimal balance) {
    }
    record Transaction(BigDecimal amount) {
    }

    List<Transaction> transactions = List.of(new Transaction(new BigDecimal("100")),
        new Transaction(new BigDecimal("-50")), new Transaction(new BigDecimal("25")));

    Account account = transactions.stream()
        .gather(Gatherers.fold(() -> new Account(BigDecimal.ZERO),
            (a, transaction) -> new Account(a.balance().add(transaction.amount()))))
        .findFirst().get();

    System.out.println(account.balance());
  }

  record ProductImage(URI sourceUrl, int productId) {
    public String processImage() throws InterruptedException {
      // Simulate CPU-intensive image processing
      Thread.sleep(ThreadLocalRandom.current().nextInt(100, 500));
      return "Processed thumbnail for %s (%s)".formatted(this.productId, this.sourceUrl);
    }
  }

  private static void mapConcurrent() {
    List<ProductImage> images = IntStream.rangeClosed(1, 10)
        .mapToObj(i -> new ProductImage(
            URI.create("https://cdn.example.com/products/" + i + ".jpg"), i))
        .toList();

    System.out.println("Starting parallel processing (max 4 threads)...");
    List<String> results = images.stream().gather(Gatherers.mapConcurrent(4,
        // Max parallel tasks
        image -> {
          System.out.printf("Processing %s%n", image.productId());
          try {
            return image.processImage();
          }
          catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            throw new RuntimeException(e);
          }
        })).toList();
    results.forEach(System.out::println);
  }

  private final static void custom1() {
    Gatherer<String, AtomicLong, String> rateLimitGatherer = Gatherer
        .ofSequential(() -> new AtomicLong(0), (lastTime, element, downstream) -> {
          long currentTime = System.currentTimeMillis();
          long elapsed = currentTime - lastTime.get();
          if (elapsed < 1000) {
            try {
              Thread.sleep(1000 - elapsed);
            }
            catch (InterruptedException e) {
              Thread.currentThread().interrupt();
              throw new RuntimeException(e);
            }
          }
          lastTime.set(System.currentTimeMillis());
          return downstream.push(element);
        });

    List<String> logs = List.of("Log 1", "Log 2", "Log 3", "Log 4", "Log 5");
    logs.stream().gather(rateLimitGatherer).forEach(System.out::println);
  }

  private final static void custom2() {
    class State {
      int first, second;
      boolean hasFirst, hasSecond;
    }
    // Emit elements only when three consecutive increasing values appear.
    Gatherer<Integer, State, Integer> increasingTriplet = Gatherer
        .ofSequential(State::new, (state, element, downstream) -> {
          if (state.hasFirst && state.hasSecond && element > state.second) {
            boolean canSendMore = downstream.push(state.first);
            if (!canSendMore) {
              return false;
            }
            canSendMore = downstream.push(state.second);
            if (!canSendMore) {
              return false;
            }
            canSendMore = downstream.push(element);
            if (!canSendMore) {
              return false;
            }

            state.first = state.second = 0;
            state.hasFirst = state.hasSecond = false;
          }
          else if (state.hasFirst && state.hasSecond && element <= state.second) {
            state.first = state.second;
            state.second = element;
          }
          else if (state.hasFirst && !state.hasSecond && element > state.first) {
            state.second = element;
            state.hasSecond = true;
          }
          else if (!state.hasFirst) {
            state.first = element;
            state.hasFirst = true;
          }
          return true;
        });

    List<Integer> triplets = Stream.of(2, 4, 5, 1, 1, 4).gather(increasingTriplet)
        .toList(); 
    System.out.println(triplets);
	// [2, 4, 5, 3, 4] (if 3 < 4 < next element...)
  }

  private final static void custom3() {
    Gatherer<Integer, Void, Integer> doubleValues = Gatherer.ofSequential(
        Gatherer.Integrator.ofGreedy((state, element, downstream) -> downstream
            .push(element * 2)));

    List<Integer> result = Stream.of(1, 2, 3).gather(doubleValues).toList();
    System.out.println(result);
	// [2, 4, 6]
  }

  private final static void custom4() {
    Gatherer<Integer, Void, Integer> doubleValuesNonGreedy = Gatherer.ofSequential(
        (state, element, downstream) -> downstream.push(element * 2));

    List<Integer> result = Stream.of(1, 2, 3).gather(doubleValuesNonGreedy).toList();
    System.out.println(result);
	// [2, 4, 6]
  }

  private final static void custom5() {
    class State {
      int runningTotal;
    }

    Gatherer<Integer, State, Integer> runningSumGreedy = Gatherer.ofSequential(
        State::new,
        Gatherer.Integrator.ofGreedy((state, element, downstream) -> {
          state.runningTotal += element;
          return downstream.push(state.runningTotal);
        }));

    List<Integer> result = Stream.of(1, 2, 3, 4).gather(runningSumGreedy).toList();
    System.out.println(result);
	// [1, 3, 6, 10]
  }

  private final static void custom6() {
    Gatherer<Integer, Void, String> labelValues = Gatherer.ofSequential(
        Gatherer.Integrator
            .ofGreedy((state, element, downstream) -> downstream.push("value=" + element)),
        (state, downstream) -> {
          downstream.push("END");
        });

    List<String> result = Stream.of(10, 20, 30).gather(labelValues).toList();
    List<String> noData = Stream.<Integer>empty().gather(labelValues).toList();
    System.out.println(result);
	// [value=10, value=20, value=30, END]
    System.out.println(noData);
	// [END] (no input elements)
  }

  private final static void custom7() {
    class State {
      Integer pending;
    }

    Gatherer<Integer, State, Integer> pairSums = Gatherer.ofSequential(State::new,
        Gatherer.Integrator.ofGreedy((state, element, downstream) -> {
          if (state.pending == null) {
            state.pending = element;
            return true;
          }
          int sum = state.pending + element;
          state.pending = null;
          return downstream.push(sum);
        }), (state, downstream) -> {
          if (state.pending != null) {
            downstream.push(state.pending);
          }
        });

    List<Integer> result = Stream.of(1, 2, 3, 4, 5).gather(pairSums).toList();
    System.out.println(result);
	// [3, 7, 5]
  }

  private static void quickRefresher() {
    List<String> result = Stream.of("the", "", "fox", "jumps").filter(s -> !s.isBlank())
        .map(String::toUpperCase).toList();
    System.out.println("Quick refresher: " + result);
  }

  private static void sequentialTemplateDemo() {
    Gatherer<Integer, int[], Integer> everySecondElement = Gatherer.ofSequential(
        () -> new int[] { 0 }, (state, element, downstream) -> {
          state[0]++;
          if (state[0] % 2 == 0) {
            return downstream.push(element);
          }
          return true;
        }, (state, downstream) -> {
        });

    List<Integer> result = Stream.of(1, 2, 3, 4, 5, 6).gather(everySecondElement).toList();
    System.out.println("sequential template demo: " + result);
  }

  private static void greedyVsNonGreedyDemo() {
    Gatherer<Integer, int[], Integer> greedy = Gatherer.ofSequential(() -> new int[] { 0 },
        Gatherer.Integrator.ofGreedy((state, element, downstream) -> downstream.push(element)));

    Gatherer<Integer, int[], Integer> nonGreedy = Gatherer.ofSequential(() -> new int[] { 0 },
        (state, element, downstream) -> {
          if (element >= 5) {
            downstream.push(element);
            return false;
          }
          return true;
        });

    List<Integer> greedyResult = Stream.iterate(1, i -> i + 1).gather(greedy).limit(3).toList();
    List<Integer> nonGreedyResult = Stream.of(1, 2, 3, 4, 5, 6, 7).gather(nonGreedy).toList();

    System.out.println("greedy integrator (stopped by downstream limit): " + greedyResult);
    System.out.println("non-greedy integrator (self short-circuit): " + nonGreedyResult);
  }

  private static void compositionDemo() {
    Gatherer<Integer, ?, Integer> running = Gatherers.scan(() -> 0, Integer::sum);

    Gatherer<Integer, ?, String> asCsv = Gatherers.fold(() -> "",
        (acc, n) -> acc.isEmpty() ? n.toString() : acc + ";" + n);

    String out = Stream.of(1, 2, 3, 4).gather(running.andThen(asCsv)).findFirst().orElse("");
    System.out.println("composition andThen: " + out);
  }

}