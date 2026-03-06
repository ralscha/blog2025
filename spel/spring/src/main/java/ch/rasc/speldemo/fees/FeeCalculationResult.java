package ch.rasc.speldemo.fees;

import java.util.List;

public record FeeCalculationResult(FeeRequest request, List<FeeLineItem> lineItems,
		double totalFee) {
}
