package ch.rasc.codemode.demo;

import java.time.Duration;
import java.util.List;
import java.util.Map;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

import io.modelcontextprotocol.client.McpClient;
import io.modelcontextprotocol.client.McpSyncClient;
import io.modelcontextprotocol.client.transport.HttpClientStreamableHttpTransport;
import io.modelcontextprotocol.spec.McpSchema;
import jakarta.annotation.PreDestroy;

@Component
public class DemoMcpClient {

	private final McpSyncClient client;

	public DemoMcpClient(@Value("${codemode.mcp.base-url}") String baseUrl,
			@Value("${codemode.mcp.endpoint:/mcp}") String endpoint, @Value("${codemode.mcp.api-key:}") String apiKey) {
		HttpClientStreamableHttpTransport transport = HttpClientStreamableHttpTransport.builder(baseUrl)
			.endpoint(endpoint)
			.httpRequestCustomizer((requestBuilder, _, _, _, _) -> {
				if (apiKey != null && !apiKey.isBlank()) {
					requestBuilder.header("X-api-key", apiKey);
				}
			})
			.build();

		this.client = McpClient.sync(transport)
			.clientInfo(new McpSchema.Implementation("codemode-demo", "0.0.1"))
			.requestTimeout(Duration.ofSeconds(30))
			.build();
		this.client.initialize();
	}

	public List<McpSchema.Tool> listTools() {
		McpSchema.ListToolsResult result = this.client.listTools();
		return result == null || result.tools() == null ? List.of() : result.tools();
	}

	public McpSchema.CallToolResult callTool(String toolName, Map<String, Object> arguments) {
		return this.client.callTool(new McpSchema.CallToolRequest(toolName, arguments == null ? Map.of() : arguments));
	}

	@PreDestroy
	void close() {
		this.client.close();
	}

}