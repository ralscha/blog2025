package ch.rasc.codemode.mcp.tools;

import java.math.BigDecimal;
import java.math.RoundingMode;
import java.time.OffsetDateTime;
import java.time.ZoneId;
import java.time.ZoneOffset;
import java.time.format.DateTimeFormatter;
import java.util.List;
import java.util.Map;

import org.springframework.stereotype.Component;

@Component
public class DemoMcpTools {

	private static final Map<String, ZoneId> CITY_ZONES = Map.of("Zurich", ZoneId.of("Europe/Zurich"), "Amsterdam",
			ZoneId.of("Europe/Amsterdam"), "Berlin", ZoneId.of("Europe/Berlin"), "Madrid", ZoneId.of("Europe/Madrid"),
			"New York", ZoneId.of("America/New_York"));

	public AddNumbersResult addNumbers(double a, double b) {
		return new AddNumbersResult(a, b, round(a + b));
	}

	public CityTimeResult cityTime(String city) {
		ZoneId zoneId = CITY_ZONES.getOrDefault(city, ZoneOffset.UTC);
		OffsetDateTime now = OffsetDateTime.now(zoneId);
		return new CityTimeResult(city, zoneId.toString(), now.format(DateTimeFormatter.ISO_OFFSET_DATE_TIME),
				now.toEpochSecond());
	}

	public ShiftTimeResult shiftTime(String rfc3339, double hours) {
		OffsetDateTime shifted = OffsetDateTime.parse(rfc3339).plusMinutes(Math.round(hours * 60));
		return new ShiftTimeResult(rfc3339, hours, shifted.format(DateTimeFormatter.ISO_OFFSET_DATE_TIME));
	}

	public ListCarriersResult listCarriers(String originCountry, String destinationCountry) {
		return new ListCarriersResult(originCountry, destinationCountry,
				List.of("correos_priority", "dhl_economy", "gls_euro_business", "ups_standard"));
	}

	public QuoteRateResult quoteRate(String carrier, String originCountry, String destinationCountry, double weightKg) {

		double base = switch (carrier) {
			case "correos_priority" -> 9.75;
			case "dhl_economy" -> 11.40;
			case "gls_euro_business" -> 10.90;
			case "ups_standard" -> 12.20;
			default -> 14.00;
		};
		double total = round(base + weightKg * 1.005);
		return new QuoteRateResult(carrier, originCountry, destinationCountry, weightKg, total, "EUR");
	}

	public EstimateDeliveryResult estimateDelivery(String carrier, String originCountry, String destinationCountry) {
		return switch (carrier) {
			case "correos_priority" -> new EstimateDeliveryResult(carrier, originCountry, destinationCountry, 3, 4);
			case "dhl_economy" -> new EstimateDeliveryResult(carrier, originCountry, destinationCountry, 4, 5);
			case "gls_euro_business" -> new EstimateDeliveryResult(carrier, originCountry, destinationCountry, 3, 5);
			case "ups_standard" -> new EstimateDeliveryResult(carrier, originCountry, destinationCountry, 2, 4);
			default -> new EstimateDeliveryResult(carrier, originCountry, destinationCountry, 5, 7);
		};
	}

	public ApplySurchargeResult applySurcharge(String carrier, double weightKg, boolean isRemoteArea,
			boolean isFragile) {

		double remote = isRemoteArea ? 2.40 : 0.0;
		double fragile = isFragile ? 1.60 : 0.0;
		double heavy = weightKg > 2.0 ? 0.75 : 0.0;
		return new ApplySurchargeResult(carrier, round(remote), round(fragile), round(heavy),
				round(remote + fragile + heavy));
	}

	public QuoteSummaryResult quoteSummary(String carrier, double basePriceEur, double surchargeEur, int minDays,
			int maxDays) {

		double totalPrice = round(basePriceEur + surchargeEur);
		return new QuoteSummaryResult(carrier, round(basePriceEur), round(surchargeEur), totalPrice, minDays, maxDays,
				minDays + "-" + maxDays + " business days", "EUR");
	}

	private static double round(double value) {
		return BigDecimal.valueOf(value).setScale(2, RoundingMode.HALF_UP).doubleValue();
	}

	public record AddNumbersResult(double a, double b, double sum) {
	}

	public record CityTimeResult(String city, String timezone, String rfc3339, long unix) {
	}

	public record ShiftTimeResult(String original, double hours, String shifted) {
	}

	public record ListCarriersResult(String originCountry, String destinationCountry, List<String> carriers) {
	}

	public record QuoteRateResult(String carrier, String originCountry, String destinationCountry, double weightKg,
			double basePriceEur, String currency) {
	}

	public record EstimateDeliveryResult(String carrier, String originCountry, String destinationCountry, int minDays,
			int maxDays) {
	}

	public record ApplySurchargeResult(String carrier, double remoteAreaSurchargeEur, double fragileSurchargeEur,
			double heavyWeightSurchargeEur, double totalSurchargeEur) {
	}

	public record QuoteSummaryResult(String carrier, double basePriceEur, double surchargeEur, double totalPriceEur,
			int minDays, int maxDays, String deliveryWindow, String currency) {
	}

}