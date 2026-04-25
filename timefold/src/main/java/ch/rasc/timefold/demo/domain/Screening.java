package ch.rasc.timefold.demo.domain;

import java.time.LocalDateTime;
import java.util.Objects;
import java.util.Set;

import ai.timefold.solver.core.api.domain.common.PlanningId;
import ai.timefold.solver.core.api.domain.entity.PlanningEntity;
import ai.timefold.solver.core.api.domain.variable.PlanningVariable;

@PlanningEntity
public class Screening {

  @PlanningId
  private String id;
  private String title;
  private String director;
  private final Set<String> guests;
  private String audienceSegment;
  private int expectedAudience;
  private int durationMinutes;
  private boolean requires4k;
  private boolean requiresStage;
  private boolean premiere;

  @PlanningVariable(valueRangeProviderRefs = "timeslotRange", allowsUnassigned = true)
  private Timeslot timeslot;

  @PlanningVariable(valueRangeProviderRefs = "screenRange", allowsUnassigned = true)
  private Screen screen;

  public Screening() {
    this.guests = Set.of();
  }

  public Screening(String id, String title, String director, Set<String> guests, String audienceSegment,
      int expectedAudience, int durationMinutes, boolean requires4k, boolean requiresStage, boolean premiere) {
    this.id = id;
    this.title = title;
    this.director = director;
    this.guests = guests;
    this.audienceSegment = audienceSegment;
    this.expectedAudience = expectedAudience;
    this.durationMinutes = durationMinutes;
    this.requires4k = requires4k;
    this.requiresStage = requiresStage;
    this.premiere = premiere;
  }

  public String getId() {
    return this.id;
  }

  public String getTitle() {
    return this.title;
  }

  public String getDirector() {
    return this.director;
  }

  public Set<String> getGuests() {
    return this.guests;
  }

  public String getAudienceSegment() {
    return this.audienceSegment;
  }

  public int getExpectedAudience() {
    return this.expectedAudience;
  }

  public int getDurationMinutes() {
    return this.durationMinutes;
  }

  public boolean isRequires4k() {
    return this.requires4k;
  }

  public boolean isRequiresStage() {
    return this.requiresStage;
  }

  public boolean isPremiere() {
    return this.premiere;
  }

  public Timeslot getTimeslot() {
    return this.timeslot;
  }

  public void setTimeslot(Timeslot timeslot) {
    this.timeslot = timeslot;
  }

  public Screen getScreen() {
    return this.screen;
  }

  public void setScreen(Screen screen) {
    this.screen = screen;
  }

  public boolean sharesGuestWith(Screening other) {
    return this.guests.stream().anyMatch(other.guests::contains);
  }

  public boolean isAssigned() {
    return this.timeslot != null && this.screen != null;
  }

  public boolean isPartiallyAssigned() {
    return this.timeslot == null != (this.screen == null);
  }

  public boolean isUnassigned() {
    return this.timeslot == null && this.screen == null;
  }

  public LocalDateTime getStart() {
    return this.timeslot == null ? null : this.timeslot.getStart();
  }

  public LocalDateTime getEnd() {
    return this.timeslot == null ? null : this.timeslot.getStart().plusMinutes(this.durationMinutes);
  }

  public boolean overlapsWith(Screening other) {
    if (!isAssigned() || !other.isAssigned()) {
      return false;
    }
    return getStart().isBefore(other.getEnd()) && other.getStart().isBefore(getEnd());
  }

  public String scheduleLabel() {
    if (this.timeslot == null) {
      return "UNASSIGNED";
    }
    return this.timeslot.startLabel() + "-" + getEnd().toLocalTime();
  }

  public int unassignedPenalty() {
    int basePenalty = 40 + this.expectedAudience / 10;
    if (this.premiere) {
      basePenalty += 25;
    }
    if (this.requiresStage && this.requires4k) {
      basePenalty += 20;
    }
    return basePenalty;
  }

  public boolean canUse(Screen candidateScreen) {
    if ((candidateScreen == null) || (this.requires4k && !candidateScreen.isSupports4k())) {
      return false;
    }
    return !this.requiresStage || candidateScreen.isHasStage();
  }

  public int overflow() {
    if (this.screen == null) {
      return 0;
    }
    return Math.max(0, this.expectedAudience - this.screen.getCapacity());
  }

  public int spareSeats() {
    if (this.screen == null) {
      return 0;
    }
    return Math.max(0, this.screen.getCapacity() - this.expectedAudience);
  }

  @Override
  public boolean equals(Object o) {
    if (this == o) {
      return true;
    }
    if (!(o instanceof Screening screening)) {
      return false;
    }
    return Objects.equals(this.id, screening.id);
  }

  @Override
  public int hashCode() {
    return Objects.hash(this.id);
  }

  @Override
  public String toString() {
    return this.title;
  }
}