package ch.rasc.speldemo.fees;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.List;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

@Component
public class ExternalFeeRuleLoader {

	private final Path rulesFile;

	public ExternalFeeRuleLoader(
			@Value("${fee.rules.file:config/fee-rules.txt}") String rulesFile) {
		this.rulesFile = Path.of(rulesFile);
	}

	public List<FeeRule> load() {
		if (!Files.exists(this.rulesFile)) {
			throw new IllegalStateException(
					"Rules file not found: " + this.rulesFile.toAbsolutePath());
		}

		try {
			return Files.readAllLines(this.rulesFile).stream().map(String::trim)
					.filter(line -> !line.isEmpty() && !line.startsWith("#"))
					.map(this::parseLine).toList();
		}
		catch (IOException e) {
			throw new IllegalStateException(
					"Could not read rules file: " + this.rulesFile.toAbsolutePath(),
					e);
		}
	}

	private FeeRule parseLine(String line) {
		String[] parts = line.split("\\|", 4);
		if (parts.length != 4) {
			throw new IllegalArgumentException(
					"Invalid rule format. Expected name|condition|fee|description but got: "
							+ line);
		}
		return new FeeRule(parts[0].trim(), parts[1].trim(), parts[2].trim(),
				parts[3].trim());
	}
}
