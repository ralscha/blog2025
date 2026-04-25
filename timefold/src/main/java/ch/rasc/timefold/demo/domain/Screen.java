package ch.rasc.timefold.demo.domain;

import java.util.Objects;
import java.util.Set;

public class Screen {

  private String id;
  private String name;
  private int capacity;
  private boolean supports4k;
  private boolean hasStage;
  private final Set<String> availableTimeslotIds;

  public Screen() {
    this.availableTimeslotIds = Set.of();
  }

  public Screen(String id, String name, int capacity, boolean supports4k, boolean hasStage,
      Set<String> availableTimeslotIds) {
    this.id = id;
    this.name = name;
    this.capacity = capacity;
    this.supports4k = supports4k;
    this.hasStage = hasStage;
    this.availableTimeslotIds = availableTimeslotIds;
  }

  public String getId() {
    return this.id;
  }

  public String getName() {
    return this.name;
  }

  public int getCapacity() {
    return this.capacity;
  }

  public boolean isSupports4k() {
    return this.supports4k;
  }

  public boolean isHasStage() {
    return this.hasStage;
  }

  public boolean isAvailable(Timeslot timeslot) {
    return timeslot != null && this.availableTimeslotIds.contains(timeslot.getId());
  }

  @Override
  public boolean equals(Object o) {
    if (this == o) {
      return true;
    }
    if (!(o instanceof Screen screen)) {
      return false;
    }
    return Objects.equals(this.id, screen.id);
  }

  @Override
  public int hashCode() {
    return Objects.hash(this.id);
  }

  @Override
  public String toString() {
    return this.name;
  }
}