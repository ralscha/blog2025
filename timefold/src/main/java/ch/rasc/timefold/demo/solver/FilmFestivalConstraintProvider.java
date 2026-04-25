package ch.rasc.timefold.demo.solver;

import ai.timefold.solver.core.api.score.HardSoftScore;
import ai.timefold.solver.core.api.score.stream.Constraint;
import ai.timefold.solver.core.api.score.stream.ConstraintFactory;
import ai.timefold.solver.core.api.score.stream.ConstraintProvider;
import ai.timefold.solver.core.api.score.stream.Joiners;
import ch.rasc.timefold.demo.domain.Screening;

public class FilmFestivalConstraintProvider implements ConstraintProvider {

  @Override
  public Constraint[] defineConstraints(ConstraintFactory constraintFactory) {
    return new Constraint[]{partiallyAssigned(constraintFactory), screenConflict(constraintFactory),
        guestConflict(constraintFactory), unavailableScreen(constraintFactory), unsupportedScreen(constraintFactory),
        audienceOverflow(constraintFactory), unassignedScreening(constraintFactory),
        premiereOutsidePrimeTime(constraintFactory), audienceSegmentClash(constraintFactory),
        seatMismatch(constraintFactory)};
  }

  Constraint partiallyAssigned(ConstraintFactory constraintFactory) {
    return constraintFactory.forEachIncludingUnassigned(Screening.class).filter(Screening::isPartiallyAssigned)
        .penalize(HardSoftScore.ONE_HARD).asConstraint("partially assigned screening");
  }

  Constraint screenConflict(ConstraintFactory constraintFactory) {
    return constraintFactory.forEachUniquePair(Screening.class, Joiners.equal(Screening::getScreen))
        .filter((left, right) -> left.getScreen() != null && left.overlapsWith(right)).penalize(HardSoftScore.ONE_HARD)
        .asConstraint("screen conflict");
  }

  Constraint guestConflict(ConstraintFactory constraintFactory) {
    return constraintFactory.forEachUniquePair(Screening.class)
        .filter((left, right) -> left.overlapsWith(right) && left.sharesGuestWith(right))
        .penalize(HardSoftScore.ONE_HARD).asConstraint("guest conflict");
  }

  Constraint unavailableScreen(ConstraintFactory constraintFactory) {
    return constraintFactory.forEach(Screening.class)
        .filter(screening -> screening.isAssigned() && !screening.getScreen().isAvailable(screening.getTimeslot()))
        .penalize(HardSoftScore.ONE_HARD).asConstraint("screen unavailable");
  }

  Constraint unsupportedScreen(ConstraintFactory constraintFactory) {
    return constraintFactory.forEach(Screening.class)
        .filter(screening -> screening.getScreen() != null && !screening.canUse(screening.getScreen()))
        .penalize(HardSoftScore.ONE_HARD).asConstraint("screen unsupported");
  }

  Constraint audienceOverflow(ConstraintFactory constraintFactory) {
    return constraintFactory.forEach(Screening.class)
        .filter(screening -> screening.getScreen() != null && screening.overflow() > 0)
        .penalize(HardSoftScore.ONE_HARD, Screening::overflow).asConstraint("audience overflow");
  }

  Constraint unassignedScreening(ConstraintFactory constraintFactory) {
    return constraintFactory.forEachIncludingUnassigned(Screening.class).filter(Screening::isUnassigned)
        .penalize(HardSoftScore.ONE_SOFT, Screening::unassignedPenalty).asConstraint("unassigned screening");
  }

  Constraint premiereOutsidePrimeTime(ConstraintFactory constraintFactory) {
    return constraintFactory.forEach(Screening.class)
        .filter(screening -> screening.isPremiere() && screening.isAssigned() && !screening.getTimeslot().isPrimeTime())
        .penalize(HardSoftScore.ONE_SOFT, _ -> 20).asConstraint("premiere outside prime time");
  }

  Constraint audienceSegmentClash(ConstraintFactory constraintFactory) {
    return constraintFactory.forEachUniquePair(Screening.class, Joiners.equal(Screening::getAudienceSegment))
        .filter(Screening::overlapsWith).penalize(HardSoftScore.ONE_SOFT, (_, _) -> 12)
        .asConstraint("audience segment clash");
  }

  Constraint seatMismatch(ConstraintFactory constraintFactory) {
    return constraintFactory.forEach(Screening.class).filter(screening -> screening.getScreen() != null)
        .penalize(HardSoftScore.ONE_SOFT, screening -> screening.spareSeats() / 10).asConstraint("seat mismatch");
  }
}