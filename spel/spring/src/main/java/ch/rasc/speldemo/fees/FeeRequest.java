package ch.rasc.speldemo.fees;

public record FeeRequest(double amount, String currency, String paymentMethod,
		String merchantTier, String channel, boolean crossBorder, int installments,
		double monthlyVolume, int riskScore, double chargebackRate) {

	public double riskMultiplier() {
		return FeeMath.riskMultiplier(this.chargebackRate);
	}

	public int riskScoreDistanceFrom50() {
		return Math.abs(this.riskScore - 50);
	}
}
