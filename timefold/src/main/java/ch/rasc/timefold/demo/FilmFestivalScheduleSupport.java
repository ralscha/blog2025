package ch.rasc.timefold.demo;

import java.time.Duration;
import java.util.Comparator;
import java.util.List;

import ai.timefold.solver.core.api.solver.SolverFactory;
import ai.timefold.solver.core.config.solver.SolverConfig;
import ai.timefold.solver.core.config.solver.termination.TerminationConfig;
import ch.rasc.timefold.demo.demo.DemoData;
import ch.rasc.timefold.demo.domain.FilmFestivalSchedule;
import ch.rasc.timefold.demo.domain.Screen;
import ch.rasc.timefold.demo.domain.Screening;
import ch.rasc.timefold.demo.domain.Timeslot;
import ch.rasc.timefold.demo.solver.FilmFestivalConstraintProvider;

public final class FilmFestivalScheduleSupport {

  private FilmFestivalScheduleSupport() {
  }

  public static FilmFestivalSchedule solveSampleFestival() {
    FilmFestivalSchedule problem = DemoData.sampleFestival();
    SolverFactory<FilmFestivalSchedule> solverFactory = SolverFactory
        .create(new SolverConfig().withSolutionClass(FilmFestivalSchedule.class).withEntityClasses(Screening.class)
            .withConstraintProviderClass(FilmFestivalConstraintProvider.class)
            .withTerminationConfig(new TerminationConfig().withSpentLimit(Duration.ofSeconds(2))));
    return solverFactory.buildSolver().solve(problem);
  }

  public static List<Screening> sortedScreenings(FilmFestivalSchedule solution) {
    return solution.getScreenings().stream()
        .sorted(
            Comparator.comparing(Screening::getTimeslot, Comparator.nullsLast(Comparator.comparing(Timeslot::getStart)))
                .thenComparing(Screening::isUnassigned)
                .thenComparing(Screening::getScreen, Comparator.nullsLast(Comparator.comparing(Screen::getName))))
        .toList();
  }

  public static void printSolution(FilmFestivalSchedule solution) {
    System.out.println("Final score: " + solution.getScore());
    sortedScreenings(solution)
        .forEach(screening -> System.out.printf("%s | %-18s | %-26s | duration=%3d | audience=%3d | segment=%s%n",
            screening.scheduleLabel(), screening.getScreen() == null ? "UNASSIGNED" : screening.getScreen().getName(),
            screening.getTitle(), screening.getDurationMinutes(), screening.getExpectedAudience(),
            screening.getAudienceSegment()));
  }
}