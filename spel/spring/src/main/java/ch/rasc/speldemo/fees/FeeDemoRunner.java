package ch.rasc.speldemo.fees;

import java.util.List;

import org.springframework.boot.CommandLineRunner;
import org.springframework.stereotype.Component;

@Component
public class FeeDemoRunner implements CommandLineRunner {

	private final SpelFeeEngine feeEngine;
	private final FeeProgramMetadata metadata;

	public FeeDemoRunner(SpelFeeEngine feeEngine, FeeProgramMetadata metadata) {
		this.feeEngine = feeEngine;
		this.metadata = metadata;
	}

	@Override
	public void run(String... args) {
		System.out.println("\n=== SpEL Fee Rule Engine Demo ===");
		System.out.printf("Operator: %s | Year: %d | Rules: %s%n",
				this.metadata.operator(), this.metadata.year(),
				this.metadata.configuredRulesFile());
		List<FeeRequest> scenarios = List.of(
				new FeeRequest(3200.00, "EUR", "CARD", "SCALE", "CHECKOUT", true, 1,
						92000, 42, 0.009),
				new FeeRequest(18500.00, "USD", "CARD", "ENTERPRISE",
						"INSTANT_PAYOUT", true, 3, 460000, 88, 0.024),
				new FeeRequest(120.00, "CHF", "ACH", "STARTER", "BATCH", false, 1,
						12000, 35, 0.003));

		for (FeeRequest request : scenarios) {
			printResult(this.feeEngine.calculate(request));
		}
	}

	private void printResult(FeeCalculationResult result) {
		FeeRequest request = result.request();
		System.out.printf("\nCase: %,.2f %s | method=%s | tier=%s | channel=%s%n",
				request.amount(), request.currency(),
				request.paymentMethod(), request.merchantTier(), request.channel());

		for (FeeLineItem lineItem : result.lineItems()) {
			System.out.printf("  %-22s %8.2f   %s%n", lineItem.ruleName(),
					lineItem.amount(), lineItem.description());
		}

		System.out.printf("  %-22s %8.2f%n", "TOTAL", result.totalFee());
	}
}
