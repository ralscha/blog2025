package ch.rasc.codemode.demo;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;

import org.graalvm.polyglot.Context;
import org.graalvm.polyglot.PolyglotException;
import org.graalvm.polyglot.Value;
import org.springframework.stereotype.Component;

import io.modelcontextprotocol.spec.McpSchema;
import tools.jackson.databind.ObjectMapper;

@Component
public class JsSandbox {

	private final ToolCatalog catalog;

	private final DemoMcpClient mcpClient;

	private final ObjectMapper objectMapper;

	public JsSandbox(ToolCatalog catalog, DemoMcpClient mcpClient, ObjectMapper objectMapper) {
		this.catalog = catalog;
		this.mcpClient = mcpClient;
		this.objectMapper = objectMapper;
	}

	public record ExecutionResult(List<String> logs, Object value) {
	}

	public ExecutionResult execute(String code) {
		List<String> logs = new ArrayList<>();

		try (Context ctx = Context.newBuilder("js").allowAllAccess(false).option("js.strict", "false").build()) {

			Value bindings = ctx.getBindings("js");

			// console.log bridge
			bindings.putMember("__host_log", (org.graalvm.polyglot.proxy.ProxyExecutable) args -> {
				StringBuilder sb = new StringBuilder();
				for (Value arg : args) {
					sb.append(arg.toString());
				}
				logs.add(sb.toString());
				return null;
			});

			// Register each MCP tool as a synchronous JS callable
			for (ToolCatalog.ToolEntry entry : this.catalog.all()) {
				String callable = entry.callable();

				bindings.putMember("__bridge_" + callable, (org.graalvm.polyglot.proxy.ProxyExecutable) args -> {
					String argsJson = args.length > 0 ? args[0].asString() : "{}";
					return invokeTool(entry.name(), argsJson);
				});
			}

			String prelude = buildPrelude();
			String wrapped = prelude + "\n(() => {\n" + code + "\n})()";
			Value result = ctx.eval("js", wrapped);

			Object value = toJavaValue(result);
			return new ExecutionResult(logs, value);
		}
		catch (PolyglotException ex) {
			throw new RuntimeException("JavaScript execution failed: " + ex.getMessage(), ex);
		}
	}

	private String buildPrelude() {
		var sb = new StringBuilder();
		sb.append("const console = {\n");
		sb.append("  log: (...args) => __host_log(args.map(a => JSON.stringify(a)).join(' ')),\n");
		sb.append("};\n");

		for (ToolCatalog.ToolEntry entry : this.catalog.all()) {
			String callable = entry.callable();
			sb.append("function ")
				.append(callable)
				.append("(args) { const r = JSON.parse(__bridge_")
				.append(callable)
				.append("(JSON.stringify(args || {}))); if (r.error) { throw new Error(r.error); } return r.value; }\n");
		}
		return sb.toString();
	}

	private String invokeTool(String toolName, String argsJson) {
		try {
			Map<String, Object> arguments = this.objectMapper.readValue(argsJson, Map.class);
			McpSchema.CallToolResult result = this.mcpClient.callTool(toolName, arguments);
			Object parsed = extractValue(result);
			if (Boolean.TRUE.equals(result.isError())) {
				String message = parsed == null ? "Tool execution failed"
						: this.objectMapper.writeValueAsString(parsed);
				return this.objectMapper.writeValueAsString(Map.of("error", message));
			}
			return this.objectMapper.writeValueAsString(Map.of("value", parsed));
		}
		catch (RuntimeException ex) {
			try {
				return this.objectMapper.writeValueAsString(Map.of("error", ex.getMessage()));
			}
			catch (RuntimeException e2) {
				return "{\"error\":\"serialization error\"}";
			}
		}
	}

	private Object extractValue(McpSchema.CallToolResult result) {
		if (result == null) {
			return null;
		}
		if (result.structuredContent() != null) {
			return result.structuredContent();
		}

		List<McpSchema.Content> content = result.content();
		if (content == null || content.isEmpty()) {
			return null;
		}

		List<Object> values = new ArrayList<>(content.size());
		for (McpSchema.Content item : content) {
			values.add(extractContent(item));
		}
		return values.size() == 1 ? values.getFirst() : values;
	}

	private Object extractContent(McpSchema.Content content) {
		if (content instanceof McpSchema.TextContent textContent) {
			return parseMaybeJson(textContent.text());
		}
		return this.objectMapper.convertValue(content, Object.class);
	}

	private Object parseMaybeJson(String value) {
		if (value == null || value.isBlank()) {
			return value;
		}
		try {
			return this.objectMapper.readValue(value, Object.class);
		}
		catch (RuntimeException ex) {
			return value;
		}
	}

	private static Object toJavaValue(Value v) {
		if (v == null || v.isNull()) {
			return null;
		}
		if (v.isBoolean()) {
			return v.asBoolean();
		}
		if (v.isNumber()) {
			if (v.fitsInInt()) {
				return v.asInt();
			}
			if (v.fitsInLong()) {
				return v.asLong();
			}
			return v.asDouble();
		}
		if (v.isString()) {
			return v.asString();
		}

		try {
			return v.toString();
		}
		catch (Exception e) {
			return null;
		}
	}

}
