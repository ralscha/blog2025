package ch.rasc.codemode.mcp;

import java.io.IOException;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.function.BiFunction;

import org.springframework.boot.web.servlet.FilterRegistrationBean;
import org.springframework.boot.web.servlet.ServletRegistrationBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.core.Ordered;

import ch.rasc.codemode.mcp.tools.DemoMcpTools;
import io.modelcontextprotocol.json.McpJsonMapper;
import io.modelcontextprotocol.json.jackson3.JacksonMcpJsonMapper;
import io.modelcontextprotocol.server.McpServer;
import io.modelcontextprotocol.server.McpServerFeatures;
import io.modelcontextprotocol.server.McpSyncServer;
import io.modelcontextprotocol.server.McpSyncServerExchange;
import io.modelcontextprotocol.server.transport.HttpServletStreamableServerTransportProvider;
import io.modelcontextprotocol.spec.McpSchema;
import tools.jackson.databind.ObjectMapper;
import tools.jackson.databind.json.JsonMapper;

@Configuration(proxyBeanMethods = false)
public class McpServerConfiguration {

	@Bean
	public McpJsonMapper mcpJsonMapper(ObjectMapper objectMapper) {
		JsonMapper jsonMapper = objectMapper instanceof JsonMapper mapper ? mapper : JsonMapper.builder().build();
		return new JacksonMcpJsonMapper(jsonMapper);
	}

	@Bean
	public HttpServletStreamableServerTransportProvider mcpTransportProvider(McpJsonMapper jsonMapper,
			DemoMcpServerProperties properties) {
		return HttpServletStreamableServerTransportProvider.builder()
			.jsonMapper(jsonMapper)
			.mcpEndpoint(properties.getEndpoint())
			.build();
	}

	@Bean
	public ServletRegistrationBean<HttpServletStreamableServerTransportProvider> mcpServlet(
			HttpServletStreamableServerTransportProvider transportProvider, DemoMcpServerProperties properties) {
		String endpoint = properties.getEndpoint();
		ServletRegistrationBean<HttpServletStreamableServerTransportProvider> registration = new ServletRegistrationBean<>(
				transportProvider, endpoint, endpoint + "/*");
		registration.setLoadOnStartup(1);
		registration.setName("demoMcpServlet");
		return registration;
	}

	@Bean
	public FilterRegistrationBean<ApiKeyFilter> mcpApiKeyFilter(DemoMcpServerProperties properties) {
		String endpoint = properties.getEndpoint();
		FilterRegistrationBean<ApiKeyFilter> registration = new FilterRegistrationBean<>();
		registration.setFilter(new ApiKeyFilter(properties.getApiKey()));
		registration.addUrlPatterns(endpoint, endpoint + "/*");
		registration.setOrder(Ordered.HIGHEST_PRECEDENCE);
		registration.setName("demoMcpApiKeyFilter");
		return registration;
	}

	@Bean(destroyMethod = "closeGracefully")
	public McpSyncServer mcpServer(HttpServletStreamableServerTransportProvider transportProvider,
			McpJsonMapper jsonMapper, DemoMcpTools tools, DemoMcpServerProperties properties) {

		return McpServer.sync(transportProvider)
			.jsonMapper(jsonMapper)
			.serverInfo(properties.getName(), properties.getVersion())
			.instructions(properties.getInstructions())
			.capabilities(McpSchema.ServerCapabilities.builder().tools(true).build())
			.tools(List.of(addNumbersTool(jsonMapper, tools), cityTimeTool(jsonMapper, tools),
					shiftTimeTool(jsonMapper, tools), listCarriersTool(jsonMapper, tools),
					quoteRateTool(jsonMapper, tools), estimateDeliveryTool(jsonMapper, tools),
					applySurchargeTool(jsonMapper, tools), quoteSummaryTool(jsonMapper, tools)))
			.build();
	}

	@SuppressWarnings("static-method")
	private McpServerFeatures.SyncToolSpecification addNumbersTool(McpJsonMapper jsonMapper, DemoMcpTools tools) {
		return toolSpecification(jsonMapper, "add_numbers", "Add two decimal numbers together.",
				objectSchema(Map.of("a", numberProperty("First number"), "b", numberProperty("Second number")), "a",
						"b"),
				objectSchemaMap(Map.of("a", numberProperty("The first input number."), "b",
						numberProperty("The second input number."), "sum", numberProperty("The sum of a and b.")), "a",
						"b", "sum"),
				(_, request) -> tools.addNumbers(requiredDouble(request, "a"), requiredDouble(request, "b")));
	}

	@SuppressWarnings("static-method")
	private McpServerFeatures.SyncToolSpecification cityTimeTool(McpJsonMapper jsonMapper, DemoMcpTools tools) {
		return toolSpecification(jsonMapper, "city_time", "Get the current time for a supported city.",
				objectSchema(
						Map.of("city",
								stringProperty(
										"Supported city name such as Zurich, Amsterdam, Berlin, Madrid, or New York")),
						"city"),
				objectSchemaMap(
						Map.of("city", stringProperty("The requested city name."), "timezone",
								stringProperty("The resolved IANA timezone identifier."), "rfc3339",
								stringProperty("The current local time formatted as an RFC3339 timestamp."), "unix",
								integerProperty("The current local time as a Unix timestamp in seconds.")),
						"city", "timezone", "rfc3339", "unix"),
				(_, request) -> tools.cityTime(requiredString(request, "city")));
	}

	@SuppressWarnings("static-method")
	private McpServerFeatures.SyncToolSpecification shiftTimeTool(McpJsonMapper jsonMapper, DemoMcpTools tools) {
		return toolSpecification(jsonMapper, "shift_time", "Shift an RFC3339 timestamp by a number of hours.",
				objectSchema(Map.of("rfc3339", stringProperty("Timestamp in RFC3339 / ISO-8601 format"), "hours",
						numberProperty("Hours to add or subtract")), "rfc3339", "hours"),
				objectSchemaMap(
						Map.of("original", stringProperty("The original RFC3339 timestamp that was provided."), "hours",
								numberProperty("The number of hours applied to the timestamp."), "shifted",
								stringProperty("The shifted RFC3339 timestamp after applying the hour offset.")),
						"original", "hours", "shifted"),
				(_, request) -> tools.shiftTime(requiredString(request, "rfc3339"), requiredDouble(request, "hours")));
	}

	@SuppressWarnings("static-method")
	private McpServerFeatures.SyncToolSpecification listCarriersTool(McpJsonMapper jsonMapper, DemoMcpTools tools) {
		return toolSpecification(jsonMapper, "list_carriers",
				"List deterministic carriers for a shipping route between two countries.",
				objectSchema(
						Map.of("originCountry", stringProperty("Origin ISO 3166-1 alpha-2 country code"),
								"destinationCountry", stringProperty("Destination ISO 3166-1 alpha-2 country code")),
						"originCountry", "destinationCountry"),
				objectSchemaMap(Map.of("originCountry",
						stringProperty("The origin ISO 3166-1 alpha-2 country code echoed by the server."),
						"destinationCountry",
						stringProperty("The destination ISO 3166-1 alpha-2 country code echoed by the server."),
						"carriers",
						arrayProperty("The list of supported carrier identifiers for the requested route.",
								stringProperty(null))),
						"originCountry", "destinationCountry", "carriers"),
				(_, request) -> tools.listCarriers(requiredString(request, "originCountry"),
						requiredString(request, "destinationCountry")));
	}

	@SuppressWarnings("static-method")
	private McpServerFeatures.SyncToolSpecification quoteRateTool(McpJsonMapper jsonMapper, DemoMcpTools tools) {
		return toolSpecification(jsonMapper, "quote_rate",
				"Get a deterministic base shipping quote for a carrier and package weight.",
				objectSchema(
						Map.of("carrier", stringProperty("Carrier identifier"), "originCountry",
								stringProperty("Origin ISO country code"), "destinationCountry",
								stringProperty("Destination ISO country code"), "weightKg",
								numberProperty("Weight in kilograms")),
						"carrier", "originCountry", "destinationCountry", "weightKg"),
				objectSchemaMap(
						Map.of("carrier", stringProperty("The carrier identifier used for the quote."), "originCountry",
								stringProperty("The origin ISO 3166-1 alpha-2 country code."), "destinationCountry",
								stringProperty("The destination ISO 3166-1 alpha-2 country code."), "weightKg",
								numberProperty("The package weight in kilograms used for pricing."), "basePriceEur",
								numberProperty("The deterministic base shipping price in euros."), "currency",
								stringProperty("The currency of the quote.")),
						"carrier", "originCountry", "destinationCountry", "weightKg", "basePriceEur", "currency"),
				(_, request) -> tools.quoteRate(requiredString(request, "carrier"),
						requiredString(request, "originCountry"), requiredString(request, "destinationCountry"),
						requiredDouble(request, "weightKg")));
	}

	@SuppressWarnings("static-method")
	private McpServerFeatures.SyncToolSpecification estimateDeliveryTool(McpJsonMapper jsonMapper, DemoMcpTools tools) {
		return toolSpecification(jsonMapper, "estimate_delivery",
				"Estimate a deterministic business-day delivery window for a carrier and route.",
				objectSchema(
						Map.of("carrier", stringProperty("Carrier identifier"), "originCountry",
								stringProperty("Origin ISO country code"), "destinationCountry",
								stringProperty("Destination ISO country code")),
						"carrier", "originCountry", "destinationCountry"),
				objectSchemaMap(Map.of("carrier", stringProperty("The carrier identifier used for the estimate."),
						"originCountry", stringProperty("The origin ISO 3166-1 alpha-2 country code."),
						"destinationCountry", stringProperty("The destination ISO 3166-1 alpha-2 country code."),
						"minDays", integerProperty("The minimum estimated delivery time in business days."), "maxDays",
						integerProperty("The maximum estimated delivery time in business days.")), "carrier",
						"originCountry", "destinationCountry", "minDays", "maxDays"),
				(_, request) -> tools.estimateDelivery(requiredString(request, "carrier"),
						requiredString(request, "originCountry"), requiredString(request, "destinationCountry")));
	}

	@SuppressWarnings("static-method")
	private McpServerFeatures.SyncToolSpecification applySurchargeTool(McpJsonMapper jsonMapper, DemoMcpTools tools) {
		return toolSpecification(jsonMapper, "apply_surcharge",
				"Calculate deterministic surcharges for package traits such as weight, remote area, and fragile handling.",
				objectSchema(
						Map.of("carrier", stringProperty("Carrier identifier"), "weightKg",
								numberProperty("Weight in kilograms"), "isRemoteArea",
								booleanProperty("Whether the destination is a remote area"), "isFragile",
								booleanProperty("Whether the package is fragile")),
						"carrier", "weightKg", "isRemoteArea", "isFragile"),
				objectSchemaMap(
						Map.of("carrier", stringProperty("The carrier identifier used for surcharge calculation."),
								"remoteAreaSurchargeEur", numberProperty("The remote-area surcharge in euros."),
								"fragileSurchargeEur", numberProperty("The fragile-handling surcharge in euros."),
								"heavyWeightSurchargeEur",
								numberProperty("The heavy-weight handling surcharge in euros."), "totalSurchargeEur",
								numberProperty("The total surcharge amount in euros.")),
						"carrier", "remoteAreaSurchargeEur", "fragileSurchargeEur", "heavyWeightSurchargeEur",
						"totalSurchargeEur"),
				(_, request) -> tools.applySurcharge(requiredString(request, "carrier"),
						requiredDouble(request, "weightKg"), requiredBoolean(request, "isRemoteArea"),
						requiredBoolean(request, "isFragile")));
	}

	@SuppressWarnings("static-method")
	private McpServerFeatures.SyncToolSpecification quoteSummaryTool(McpJsonMapper jsonMapper, DemoMcpTools tools) {
		return toolSpecification(jsonMapper, "quote_summary",
				"Normalize a shipping quote into a sortable final summary.",
				objectSchema(Map.of("carrier", stringProperty("Carrier identifier"), "basePriceEur",
						numberProperty("Base price in EUR"), "surchargeEur", numberProperty("Surcharge total in EUR"),
						"minDays", integerProperty("Minimum business days"), "maxDays",
						integerProperty("Maximum business days")), "carrier", "basePriceEur", "surchargeEur", "minDays",
						"maxDays"),
				objectSchemaMap(
						Map.of("carrier", stringProperty("The carrier identifier for the summarized quote."),
								"basePriceEur", numberProperty("The base shipping price in euros before surcharges."),
								"surchargeEur", numberProperty("The total surcharge amount in euros."), "totalPriceEur",
								numberProperty("The final shipping price in euros after surcharges."), "minDays",
								integerProperty("The minimum estimated delivery time in business days."), "maxDays",
								integerProperty("The maximum estimated delivery time in business days."),
								"deliveryWindow", stringProperty("A formatted delivery window summary."), "currency",
								stringProperty("The currency of the summarized quote.")),
						"carrier", "basePriceEur", "surchargeEur", "totalPriceEur", "minDays", "maxDays",
						"deliveryWindow", "currency"),
				(_, request) -> tools.quoteSummary(requiredString(request, "carrier"),
						requiredDouble(request, "basePriceEur"), requiredDouble(request, "surchargeEur"),
						requiredInt(request, "minDays"), requiredInt(request, "maxDays")));
	}

	private static McpServerFeatures.SyncToolSpecification toolSpecification(McpJsonMapper jsonMapper, String name,
			String description, McpSchema.JsonSchema inputSchema, Map<String, Object> outputSchema,
			BiFunction<McpSyncServerExchange, McpSchema.CallToolRequest, Object> handler) {

		McpSchema.Tool tool = McpSchema.Tool.builder()
			.name(name)
			.description(description)
			.inputSchema(inputSchema)
			.outputSchema(outputSchema)
			.build();

		return McpServerFeatures.SyncToolSpecification.builder().tool(tool).callHandler((exchange, request) -> {
			try {
				Object result = handler.apply(exchange, request);
				String text = jsonMapper.writeValueAsString(result);
				return McpSchema.CallToolResult.builder().structuredContent(result).addTextContent(text).build();
			}
			catch (RuntimeException | IOException ex) {
				return McpSchema.CallToolResult.builder().isError(true).addTextContent(ex.getMessage()).build();
			}
		}).build();
	}

	private static McpSchema.JsonSchema objectSchema(Map<String, Object> properties, String... required) {
		return new McpSchema.JsonSchema("object", properties, List.of(required), false, null, null);
	}

	private static Map<String, Object> objectSchemaMap(Map<String, Object> properties, String... required) {
		Map<String, Object> schema = new LinkedHashMap<>();
		schema.put("type", "object");
		schema.put("properties", properties);
		schema.put("required", List.of(required));
		schema.put("additionalProperties", false);
		return Map.copyOf(schema);
	}

	private static Map<String, Object> stringProperty(String description) {
		return scalarProperty("string", description);
	}

	private static Map<String, Object> numberProperty(String description) {
		return scalarProperty("number", description);
	}

	private static Map<String, Object> integerProperty(String description) {
		return scalarProperty("integer", description);
	}

	private static Map<String, Object> booleanProperty(String description) {
		return scalarProperty("boolean", description);
	}

	private static Map<String, Object> arrayProperty(String description, Map<String, Object> items) {
		Map<String, Object> schema = new LinkedHashMap<>();
		schema.put("type", "array");
		if (description != null && !description.isBlank()) {
			schema.put("description", description);
		}
		schema.put("items", items);
		return Map.copyOf(schema);
	}

	private static Map<String, Object> scalarProperty(String type, String description) {
		Map<String, Object> schema = new LinkedHashMap<>();
		schema.put("type", type);
		if (description != null && !description.isBlank()) {
			schema.put("description", description);
		}
		return Map.copyOf(schema);
	}

	private static String requiredString(McpSchema.CallToolRequest request, String key) {
		Object value = requiredArgument(request, key);
		if (value instanceof String text) {
			return text;
		}
		return String.valueOf(value);
	}

	private static double requiredDouble(McpSchema.CallToolRequest request, String key) {
		Object value = requiredArgument(request, key);
		if (value instanceof Number number) {
			return number.doubleValue();
		}
		return Double.parseDouble(String.valueOf(value));
	}

	private static int requiredInt(McpSchema.CallToolRequest request, String key) {
		Object value = requiredArgument(request, key);
		if (value instanceof Number number) {
			return number.intValue();
		}
		return Integer.parseInt(String.valueOf(value));
	}

	private static boolean requiredBoolean(McpSchema.CallToolRequest request, String key) {
		Object value = requiredArgument(request, key);
		if (value instanceof Boolean bool) {
			return bool;
		}
		return Boolean.parseBoolean(String.valueOf(value));
	}

	private static Object requiredArgument(McpSchema.CallToolRequest request, String key) {
		Map<String, Object> arguments = request.arguments();
		if (arguments == null || !arguments.containsKey(key)) {
			throw new IllegalArgumentException("Missing required argument: " + key);
		}
		Object value = arguments.get(key);
		if (value == null) {
			throw new IllegalArgumentException("Argument must not be null: " + key);
		}
		return value;
	}

}