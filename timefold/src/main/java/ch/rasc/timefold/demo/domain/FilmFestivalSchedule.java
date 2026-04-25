package ch.rasc.timefold.demo.domain;

import java.util.List;

import ai.timefold.solver.core.api.domain.solution.PlanningEntityCollectionProperty;
import ai.timefold.solver.core.api.domain.solution.PlanningScore;
import ai.timefold.solver.core.api.domain.solution.PlanningSolution;
import ai.timefold.solver.core.api.domain.solution.ProblemFactCollectionProperty;
import ai.timefold.solver.core.api.domain.valuerange.ValueRangeProvider;
import ai.timefold.solver.core.api.score.HardSoftScore;

@PlanningSolution
public class FilmFestivalSchedule {

  @ProblemFactCollectionProperty
  @ValueRangeProvider(id = "timeslotRange")
  private List<Timeslot> timeslots;

  @ProblemFactCollectionProperty
  @ValueRangeProvider(id = "screenRange")
  private List<Screen> screens;

  @PlanningEntityCollectionProperty
  private List<Screening> screenings;

  @PlanningScore
  private HardSoftScore score;

  public FilmFestivalSchedule() {
  }

  public FilmFestivalSchedule(List<Timeslot> timeslots, List<Screen> screens, List<Screening> screenings) {
    this.timeslots = timeslots;
    this.screens = screens;
    this.screenings = screenings;
  }

  public List<Timeslot> getTimeslots() {
    return this.timeslots;
  }

  public List<Screen> getScreens() {
    return this.screens;
  }

  public List<Screening> getScreenings() {
    return this.screenings;
  }

  public HardSoftScore getScore() {
    return this.score;
  }

  public void setScore(HardSoftScore score) {
    this.score = score;
  }
}