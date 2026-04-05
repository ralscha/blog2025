package ch.rasc.codemode.demo;

import org.springaicommunity.tool.search.ToolSearcher;
import org.springaicommunity.tool.searcher.LuceneToolSearcher;
import org.springframework.boot.CommandLineRunner;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.context.annotation.Bean;

@SpringBootApplication
public class DemoApplication {

	public static void main(String[] args) {
		SpringApplication.run(DemoApplication.class, args);
	}

	@Bean
	ToolSearcher toolSearcher() {
		return new LuceneToolSearcher(0.25f);
	}

	@Bean
	CommandLineRunner runner(CodeModeService codeModeService) {
		return _ -> {
			String prompt = """
					Compare all shipping options for a 2.4kg parcel from Germany to Spain.
					Exclude any option slower than 4 business days and return the cheapest option.
					Nothing else.
					""".stripIndent().strip();
			String answer = codeModeService.run(prompt);
			System.out.println("\n=== Answer ===");
			System.out.println(answer);
		};
	}

}
