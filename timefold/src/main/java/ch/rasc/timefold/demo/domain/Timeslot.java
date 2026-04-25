package ch.rasc.timefold.demo.domain;

import java.time.LocalDateTime;
import java.time.format.DateTimeFormatter;
import java.util.Locale;
import java.util.Objects;

public class Timeslot {

  private static final DateTimeFormatter LABEL_FORMATTER = DateTimeFormatter.ofPattern("EEE HH:mm", Locale.ENGLISH);

  private String id;
  private LocalDateTime start;
  private LocalDateTime end;

  public Timeslot() {
  }

  public Timeslot(String id, LocalDateTime start, LocalDateTime end) {
    this.id = id;
    this.start = start;
    this.end = end;
  }

  public String getId() {
    return this.id;
  }

  public LocalDateTime getStart() {
    return this.start;
  }

  public LocalDateTime getEnd() {
    return this.end;
  }

  public boolean isPrimeTime() {
    return this.start != null && this.start.getHour() >= 18;
  }

  public String startLabel() {
    return LABEL_FORMATTER.format(this.start);
  }

  public String label() {
    return startLabel() + "-" + this.end.toLocalTime();
  }

  @Override
  public boolean equals(Object o) {
    if (this == o) {
      return true;
    }
    if (!(o instanceof Timeslot timeslot)) {
      return false;
    }
    return Objects.equals(this.id, timeslot.id);
  }

  @Override
  public int hashCode() {
    return Objects.hash(this.id);
  }

  @Override
  public String toString() {
    return label();
  }
}