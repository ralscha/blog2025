package ch.rasc.speldemo.fees;

public record FeeRule(String name, String conditionExpression,
		String feeExpression, String description) {
}
