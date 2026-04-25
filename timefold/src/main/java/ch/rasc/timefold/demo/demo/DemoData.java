package ch.rasc.timefold.demo.demo;

import java.time.LocalDateTime;
import java.util.List;
import java.util.Set;

import ch.rasc.timefold.demo.domain.FilmFestivalSchedule;
import ch.rasc.timefold.demo.domain.Screen;
import ch.rasc.timefold.demo.domain.Screening;
import ch.rasc.timefold.demo.domain.Timeslot;

public final class DemoData {

  private DemoData() {
  }

  public static FilmFestivalSchedule sampleFestival() {
    List<Timeslot> timeslots = List.of(slot("thu-18", 2026, 10, 8, 18, 0), slot("thu-20", 2026, 10, 8, 20, 30),
        slot("fri-16", 2026, 10, 9, 16, 0), slot("fri-18", 2026, 10, 9, 18, 0), slot("fri-20", 2026, 10, 9, 20, 30),
        slot("sat-17", 2026, 10, 10, 17, 30), slot("sat-20", 2026, 10, 10, 20, 15));

    List<Screen> screens = List.of(
        new Screen("grand", "Grand Theater", 220, true, true, Set.of("thu-18", "fri-18", "sat-17")),
        new Screen("warehouse", "Warehouse Screen", 180, true, false,
            Set.of("thu-18", "thu-20", "fri-16", "fri-18", "fri-20", "sat-17", "sat-20")),
        new Screen("rooftop", "Rooftop Cinema", 110, false, true,
            Set.of("thu-20", "fri-18", "fri-20", "sat-17", "sat-20")));

    List<Screening> screenings = List.of(
        new Screening("opening-night", "Opening Night: Glass Harbor", "Eva Stone", Set.of("Eva Stone", "Jun Park"),
            "arthouse", 210, 160, true, true, true),
        new Screening("quantum-hearts", "Quantum Hearts", "Mina Alvarez", Set.of("Mina Alvarez", "Leila Haddad"),
            "sci-fi", 150, 170, true, true, true),
        new Screening("ember-lake", "Ember Lake", "Daniel Sosa", Set.of("Daniel Sosa"), "drama", 185, 165, true, true,
            true),
        new Screening("after-the-storm", "After the Storm", "Lina Moreau", Set.of("Lina Moreau", "Jun Park"),
            "arthouse", 160, 175, true, true, true),
        new Screening("rust-stardust", "Rust and Stardust", "Noah Okafor", Set.of("Jun Park", "Noah Okafor"), "drama",
            170, 135, true, false, false),
        new Screening("neon-alley", "Neon Alley", "Samir Das", Set.of("Samir Das"), "action", 145, 115, true, false,
            false),
        new Screening("family-pixels", "Family Pixels", "Priya Raman", Set.of("Priya Raman"), "family", 105, 95, true,
            false, false),
        new Screening("paper-monument", "Paper Monument", "Hannah Weiss", Set.of("Hannah Weiss"), "documentary", 95,
            105, false, true, false),
        new Screening("silent-orbit", "Silent Orbit", "Ari Kim", Set.of("Ari Kim"), "arthouse", 85, 140, false, false,
            false),
        new Screening("midnight-syntax", "Midnight Syntax", "Leo Varga", Set.of("Leo Varga", "Leila Haddad"), "cult",
            120, 125, false, false, false),
        new Screening("city-of-sparks", "City of Sparks", "Rhea Banerjee", Set.of("Rhea Banerjee"), "documentary", 100,
            90, false, true, false));

    return new FilmFestivalSchedule(timeslots, screens, screenings);
  }

  private static Timeslot slot(String id, int year, int month, int day, int hour, int minute) {
    LocalDateTime start = LocalDateTime.of(year, month, day, hour, minute);
    return new Timeslot(id, start, start.plusMinutes(110));
  }
}