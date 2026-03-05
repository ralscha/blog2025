package streamgatherers;

import java.util.List;
import java.util.function.Function;
import java.util.stream.Gatherer;

public class ParallelGathererDemo {

  class WeightedAverageState {
    double weightedSum = 0.0;
    double totalWeight = 0.0;
  }

  record StudentGrade(double grade, double creditHours) {
  }

  <TR extends StudentGrade> Gatherer<TR, WeightedAverageState, Double> weightedAverage(
      Function<TR, Double> weightFunction) {

    return Gatherer.of(
        /* Initializer */
        WeightedAverageState::new,
        /* Integrator */
        Gatherer.Integrator.ofGreedy((state, element, _) -> {
          double weight = weightFunction.apply(element);
          state.weightedSum += element.grade() * weight;
          state.totalWeight += weight;
          return true;
        }),
        /* Combiner */
        (leftState, rightState) -> {
          leftState.weightedSum += rightState.weightedSum;
          leftState.totalWeight += rightState.totalWeight;
          return leftState;
        },
        /* Finisher */
        (state, downstream) -> {
          if (state.totalWeight > 0) {
            double weightedAverage = state.weightedSum / state.totalWeight;
            downstream.push(weightedAverage);
          }
        });
  }

  public static void main(String[] args) {
    ParallelGathererDemo demo = new ParallelGathererDemo();
    Function<StudentGrade, Double> weightFunction = sg -> sg.creditHours();

    List<StudentGrade> grades = List.of(new StudentGrade(90, 3), new StudentGrade(80, 4),
        new StudentGrade(85, 2), new StudentGrade(70, 3), new StudentGrade(95, 1));

    double weightedAverage = grades.stream().parallel()
      .gather(demo.weightedAverage(weightFunction)).findFirst().get();
    System.out.println("Weighted Average: " + weightedAverage);
    // Weighted Average: 81.92307692307692

    grades = List.of();
    grades.stream().parallel().gather(demo.weightedAverage(weightFunction)).findFirst()
        .ifPresentOrElse(wa -> {
          System.out.println("Weighted Averages: " + wa);
        }, () -> {
          System.out.println("No data");
        });
    // No data

  }
}
