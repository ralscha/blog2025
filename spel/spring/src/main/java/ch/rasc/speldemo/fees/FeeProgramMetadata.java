package ch.rasc.speldemo.fees;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

@Component
public class FeeProgramMetadata {

	@Value("#{systemProperties['user.name'] ?: 'unknown'}")
	private String operator;

	@Value("#{T(java.time.Year).now().value}")
	private int year;

	@Value("#{'${fee.rules.file}'.toUpperCase()}")
	private String configuredRulesFile;

	public String operator() {
		return this.operator;
	}

	public int year() {
		return this.year;
	}

	public String configuredRulesFile() {
		return this.configuredRulesFile;
	}
}
