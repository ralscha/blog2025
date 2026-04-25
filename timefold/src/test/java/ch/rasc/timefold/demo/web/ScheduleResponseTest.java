package ch.rasc.timefold.demo.web;

import static org.assertj.core.api.Assertions.assertThat;

import java.time.LocalDateTime;
import java.util.List;
import java.util.Set;

import org.junit.jupiter.api.Test;

import ai.timefold.solver.core.api.score.HardSoftScore;
import ch.rasc.timefold.demo.domain.FilmFestivalSchedule;
import ch.rasc.timefold.demo.domain.Screen;
import ch.rasc.timefold.demo.domain.Screening;
import ch.rasc.timefold.demo.domain.Timeslot;

class ScheduleResponseTest {

  @Test
  void mapsAssignedAndUnassignedScreenings() {
    Timeslot timeslot = new Timeslot("fri-18", LocalDateTime.of(2026, 10, 9, 18, 0),
        LocalDateTime.of(2026, 10, 9, 18, 30));
    Screen screen = new Screen("grand", "Grand Theater", 220, true, true, Set.of("fri-18"));

    Screening assigned = new Screening("assigned", "Opening", "Eva Stone", Set.of("Eva Stone"), "arthouse", 200, 120,
        true, true, true);
    assigned.setTimeslot(timeslot);
    assigned.setScreen(screen);

    Screening unassigned = new Screening("unassigned", "Quantum Hearts", "Mina Alvarez", Set.of("Leila Haddad"),
        "sci-fi", 150, 170, true, true, true);

    FilmFestivalSchedule solution = new FilmFestivalSchedule(List.of(timeslot), List.of(screen),
        List.of(assigned, unassigned));
    solution.setScore(HardSoftScore.of(0, -10));

    ScheduleResponse response = ScheduleResponse.fromSolution(solution);

    assertThat(response.score()).isEqualTo("0hard/-10soft");
    assertThat(response.timeslots()).hasSize(1);
    assertThat(response.screens()).hasSize(1);
    assertThat(response.screenings()).hasSize(2);
    assertThat(response.screenings()).anySatisfy(screening -> {
      assertThat(screening.id()).isEqualTo("assigned");
      assertThat(screening.assigned()).isTrue();
      assertThat(screening.unassigned()).isFalse();
      assertThat(screening.screenName()).isEqualTo("Grand Theater");
    });
    assertThat(response.screenings()).anySatisfy(screening -> {
      assertThat(screening.id()).isEqualTo("unassigned");
      assertThat(screening.assigned()).isFalse();
      assertThat(screening.unassigned()).isTrue();
      assertThat(screening.screenName()).isNull();
    });
  }
}