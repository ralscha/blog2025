package ch.rasc.speldemo.fees;

import java.math.BigDecimal;
import java.math.RoundingMode;

public final class FeeMath {

	private FeeMath() {
	}

	public static double riskMultiplier(double chargebackRate) {
		if (chargebackRate >= 0.03) {
			return 1.6;
		}
		if (chargebackRate >= 0.02) {
			return 1.35;
		}
		if (chargebackRate >= 0.01) {
			return 1.15;
		}
		return 1.0;
	}

	public static double roundMoney(double amount) {
		return BigDecimal.valueOf(amount).setScale(2, RoundingMode.HALF_UP)
				.doubleValue();
	}
}
