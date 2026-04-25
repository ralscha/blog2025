package ch.rasc.timefold.demo.web;

import java.util.Comparator;
import java.util.List;

import ch.rasc.timefold.demo.FilmFestivalScheduleSupport;
import ch.rasc.timefold.demo.domain.FilmFestivalSchedule;
import ch.rasc.timefold.demo.domain.Screening;
import ch.rasc.timefold.demo.domain.Timeslot;

public record ScheduleResponse(String score, long hardScore, long softScore, List<TimeslotView> timeslots,
    List<ScreenView> screens, List<ScreeningView> screenings) {

  public static ScheduleResponse fromSolution(FilmFestivalSchedule solution) {
    List<TimeslotView> timeslotViews = solution.getTimeslots().stream().sorted(Comparator.comparing(Timeslot::getStart))
        .map(timeslot -> new TimeslotView(timeslot.getId(), timeslot.startLabel(), timeslot.getStart().toString(),
            timeslot.getEnd().toString()))
        .toList();

    List<ScreenView> screenViews = solution.getScreens().stream()
        .map(screen -> new ScreenView(screen.getId(), screen.getName(), screen.getCapacity())).toList();

    List<ScreeningView> screeningViews = FilmFestivalScheduleSupport.sortedScreenings(solution).stream()
        .map(ScheduleResponse::toView).toList();

    return new ScheduleResponse(solution.getScore().toString(), solution.getScore().hardScore(),
        solution.getScore().softScore(), timeslotViews, screenViews, screeningViews);
  }

  private static ScreeningView toView(Screening screening) {
    return new ScreeningView(screening.getId(), screening.getTitle(), screening.getDirector(),
        screening.getScreen() == null ? null : screening.getScreen().getId(),
        screening.getScreen() == null ? null : screening.getScreen().getName(),
        screening.getTimeslot() == null ? null : screening.getTimeslot().getId(), screening.scheduleLabel(),
        screening.getStart() == null ? null : screening.getStart().toString(),
        screening.getEnd() == null ? null : screening.getEnd().toString(), screening.getDurationMinutes(),
        screening.getExpectedAudience(), screening.getAudienceSegment(), screening.isPremiere(), screening.isAssigned(),
        screening.isUnassigned());
  }

  public record TimeslotView(String id, String label, String start, String end) {
  }

  public record ScreenView(String id, String name, int capacity) {
  }

  public record ScreeningView(String id, String title, String director, String screenId, String screenName,
      String startSlotId, String scheduleLabel, String start, String end, int durationMinutes, int expectedAudience,
      String audienceSegment, boolean premiere, boolean assigned, boolean unassigned) {
  }
}