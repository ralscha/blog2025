package ch.rasc.timefold.demo.solver;

import java.time.LocalDateTime;
import java.util.Set;

import org.junit.jupiter.api.Test;

import ai.timefold.solver.core.api.score.stream.test.ConstraintVerifier;
import ch.rasc.timefold.demo.domain.FilmFestivalSchedule;
import ch.rasc.timefold.demo.domain.Screen;
import ch.rasc.timefold.demo.domain.Screening;
import ch.rasc.timefold.demo.domain.Timeslot;

class FilmFestivalConstraintProviderTest {

  private final ConstraintVerifier<FilmFestivalConstraintProvider, FilmFestivalSchedule> constraintVerifier = ConstraintVerifier
      .build(new FilmFestivalConstraintProvider(), FilmFestivalSchedule.class, Screening.class);

  @Test
  void screenConflictIsHard() {
    Timeslot firstStart = new Timeslot("fri-18", LocalDateTime.of(2026, 10, 9, 18, 0),
        LocalDateTime.of(2026, 10, 9, 18, 30));
    Timeslot secondStart = new Timeslot("fri-20", LocalDateTime.of(2026, 10, 9, 20, 0),
        LocalDateTime.of(2026, 10, 9, 20, 30));
    Screen screen = new Screen("grand", "Grand Theater", 220, true, true, Set.of("fri-18", "fri-20"));

    Screening left = new Screening("a", "Opening", "Eva Stone", Set.of("Eva Stone"), "arthouse", 200, 170, true, true,
        true);
    left.setTimeslot(firstStart);
    left.setScreen(screen);

    Screening right = new Screening("b", "Quantum Hearts", "Mina Alvarez", Set.of("Leila Haddad"), "sci-fi", 150, 110,
        true, true, true);
    right.setTimeslot(secondStart);
    right.setScreen(screen);

    this.constraintVerifier.verifyThat(FilmFestivalConstraintProvider::screenConflict).given(left, right)
        .penalizesBy(1);
  }

  @Test
  void premiereOutsidePrimeTimeIsSoft() {
    Timeslot matinee = new Timeslot("fri-16", LocalDateTime.of(2026, 10, 9, 16, 0),
        LocalDateTime.of(2026, 10, 9, 17, 50));
    Screen screen = new Screen("grand", "Grand Theater", 220, true, true, Set.of("fri-16"));

    Screening premiere = new Screening("q", "Quantum Hearts", "Mina Alvarez", Set.of("Leila Haddad"), "sci-fi", 150,
        110, true, true, true);
    premiere.setTimeslot(matinee);
    premiere.setScreen(screen);

    this.constraintVerifier.verifyThat(FilmFestivalConstraintProvider::premiereOutsidePrimeTime).given(premiere)
        .penalizesBy(20);
  }

  @Test
  void unassignedScreeningIsSoft() {
    Screening unassigned = new Screening("u", "After the Storm", "Lina Moreau", Set.of("Lina Moreau"), "arthouse", 160,
        175, true, true, true);

    this.constraintVerifier.verifyThat(FilmFestivalConstraintProvider::unassignedScreening).given(unassigned)
        .penalizesBy(unassigned.unassignedPenalty());
  }
}