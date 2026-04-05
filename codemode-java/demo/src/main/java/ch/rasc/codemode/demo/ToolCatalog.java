package ch.rasc.codemode.demo;

import java.util.ArrayList;
import java.util.Collections;
import java.util.Comparator;
import java.util.HashSet;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.Set;

import org.springaicommunity.tool.search.ToolReference;
import org.springaicommunity.tool.search.ToolSearchRequest;
import org.springaicommunity.tool.search.ToolSearcher;
import org.springframework.stereotype.Component;

import io.modelcontextprotocol.spec.McpSchema;
import jakarta.annotation.PostConstruct;
import tools.jackson.core.type.TypeReference;
import tools.jackson.databind.ObjectMapper;

@Component
public class ToolCatalog {

	private static final String SESSION = "codemode";

	private final DemoMcpClient mcpClient;

	private final ToolSearcher toolSearcher;

	private final ObjectMapper objectMapper;

	private final Map<String, ToolEntry> catalog = new LinkedHashMap<>();

	public ToolCatalog(DemoMcpClient mcpClient, ToolSearcher toolSearcher, ObjectMapper objectMapper) {
		this.mcpClient = mcpClient;
		this.toolSearcher = toolSearcher;
		this.objectMapper = objectMapper;
	}

	@PostConstruct
	void init() {
		List<ToolEntry> entries = new ArrayList<>();

		for (McpSchema.Tool tool : this.mcpClient.listTools()) {
			String serverPrefix = "demo";
			String callable = serverPrefix + "_" + tool.name();
			Map<String, Object> inputSchema = toSchemaMap(tool.inputSchema());
			Map<String, Object> outputSchema = tool.outputSchema() == null ? Map.of() : tool.outputSchema();
			entries.add(new ToolEntry(tool.name(), callable, tool.description(), inputSchema, outputSchema));
		}

		entries.sort(Comparator.comparing(ToolEntry::name));

		for (ToolEntry entry : entries) {
			this.catalog.put(entry.callable(), entry);
			this.toolSearcher.indexTool(SESSION,
					ToolReference.builder()
						.toolName(entry.callable())
						.summary(entry.name() + ": " + entry.description())
						.build());
		}
	}

	public String search(String query, int limit) {
		var response = this.toolSearcher.search(new ToolSearchRequest(SESSION, query, limit <= 0 ? 8 : limit, null));
		List<ToolEntry> matched = new ArrayList<>();
		for (var ref : response.toolReferences()) {
			ToolEntry entry = this.catalog.get(ref.toolName());
			if (entry != null) {
				matched.add(entry);
			}
		}
		return helperDefinitions(matched);
	}

	public String allDefinitions() {
		return helperDefinitions(new ArrayList<>(this.catalog.values()));
	}

	public List<ToolEntry> all() {
		return new ArrayList<>(this.catalog.values());
	}

	private String helperDefinitions(List<ToolEntry> entries) {
		if (entries.isEmpty()) {
			return "// No helper functions matched this search.\n";
		}
		var sb = new StringBuilder();
		for (ToolEntry entry : entries) {
			sb.append("/**\n");
			String description = sanitizeJSDocText(entry.description());
			if (!description.isEmpty()) {
				sb.append(" * ").append(description).append("\n");
			}
			sb.append(" * @param ").append(jsDocTypeTag(entry.inputSchema())).append(" args\n");
			writeSchemaFieldDescriptions(sb, "Input fields", "args", entry.inputSchema());
			sb.append(" * @returns ").append(jsDocTypeTag(entry.outputSchema())).append(" result\n");
			writeSchemaFieldDescriptions(sb, "Output fields", "result", entry.outputSchema());
			sb.append(" */\n");
			sb.append("function ").append(entry.callable()).append("(args) {}\n\n");
		}
		return sb.toString();
	}

	private void writeSchemaFieldDescriptions(StringBuilder sb, String heading, String root,
			Map<String, Object> schema) {
		List<String> lines = schemaFieldDescriptions(root, schema);
		if (lines.isEmpty()) {
			return;
		}
		sb.append(" * ").append(heading).append(":\n");
		for (String line : lines) {
			sb.append(" *   - ").append(line).append("\n");
		}
	}

	@SuppressWarnings("unchecked")
	private List<String> schemaFieldDescriptions(String root, Map<String, Object> schema) {
		if (schema == null || schema.isEmpty()) {
			return List.of();
		}
		Map<String, Object> properties = (Map<String, Object>) schema.get("properties");
		if (properties == null || properties.isEmpty()) {
			return List.of();
		}

		List<String> names = new ArrayList<>(properties.keySet());
		Collections.sort(names);

		List<String> lines = new ArrayList<>();
		for (String name : names) {
			Map<String, Object> propertySchema = (Map<String, Object>) properties.get(name);
			if (propertySchema == null) {
				propertySchema = Map.of();
			}
			String path = jsPropertyPath(root, name);
			String description = sanitizeJSDocText(schemaDescription(propertySchema));
			if (!description.isEmpty()) {
				lines.add(path + ": " + description);
			}
			lines.addAll(schemaFieldDescriptionsForProperty(path, propertySchema));
		}
		return lines;
	}

	@SuppressWarnings("unchecked")
	private List<String> schemaFieldDescriptionsForProperty(String path, Map<String, Object> schema) {
		if (schema == null || schema.isEmpty()) {
			return List.of();
		}

		String typeName = schemaTypeName(schema);
		if ("object".equals(typeName)) {
			return schemaFieldDescriptions(path, schema);
		}
		if ("array".equals(typeName)) {
			Map<String, Object> itemSchema = (Map<String, Object>) schema.get("items");
			if (itemSchema == null) {
				itemSchema = Map.of();
			}
			String itemPath = path + "[]";
			List<String> lines = new ArrayList<>();
			String description = sanitizeJSDocText(schemaDescription(itemSchema));
			if (!description.isEmpty()) {
				lines.add(itemPath + ": " + description);
			}
			lines.addAll(schemaFieldDescriptionsForProperty(itemPath, itemSchema));
			return lines;
		}
		return List.of();
	}

	private static String schemaDescription(Map<String, Object> schema) {
		if (schema == null || schema.isEmpty()) {
			return "";
		}
		Object description = schema.get("description");
		return description instanceof String text ? text : "";
	}

	private String jsDocTypeTag(Map<String, Object> schema) {
		if (schema == null || schema.isEmpty()) {
			return "{Record<string, any>}";
		}
		return "{" + jsDocType(schema) + "}";
	}

	@SuppressWarnings("unchecked")
	private String jsDocObjectLiteral(Map<String, Object> schema) {
		if (schema == null || schema.isEmpty()) {
			return "Record<string, any>";
		}
		Map<String, Object> properties = (Map<String, Object>) schema.get("properties");
		if (properties == null || properties.isEmpty()) {
			return "Record<string, any>";
		}

		Set<String> required = new HashSet<>();
		Object requiredValues = schema.get("required");
		if (requiredValues instanceof List<?> values) {
			for (Object value : values) {
				if (value instanceof String name) {
					required.add(name);
				}
			}
		}

		List<String> names = new ArrayList<>(properties.keySet());
		Collections.sort(names);

		List<String> parts = new ArrayList<>(names.size());
		for (String name : names) {
			Map<String, Object> propertySchema = (Map<String, Object>) properties.get(name);
			if (propertySchema == null) {
				propertySchema = Map.of();
			}
			String optional = required.contains(name) ? "" : "?";
			parts.add(jsPropertyName(name) + optional + ": " + jsDocType(propertySchema));
		}
		return "{ " + String.join(", ", parts) + " }";
	}

	@SuppressWarnings("unchecked")
	private String jsDocType(Map<String, Object> schema) {
		if (schema == null || schema.isEmpty()) {
			return "any";
		}

		Object enumValues = schema.get("enum");
		if (enumValues instanceof List<?> values && !values.isEmpty()) {
			List<String> literals = new ArrayList<>(values.size());
			for (Object value : values) {
				literals.add(jsDocLiteral(value));
			}
			return String.join(" | ", literals);
		}

		Object unionTypes = schema.get("type");
		if (unionTypes instanceof List<?> values && !values.isEmpty()) {
			List<String> parts = new ArrayList<>(values.size());
			for (Object value : values) {
				if (value instanceof String name) {
					parts.add(jsDocType(Map.of("type", name)));
				}
			}
			if (!parts.isEmpty()) {
				return String.join(" | ", parts);
			}
		}

		return switch (schemaTypeName(schema)) {
			case "string" -> "string";
			case "integer", "number" -> "number";
			case "boolean" -> "boolean";
			case "null" -> "null";
			case "array" -> {
				Map<String, Object> itemSchema = (Map<String, Object>) schema.get("items");
				yield "Array<" + jsDocType(itemSchema == null ? Map.of() : itemSchema) + ">";
			}
			case "object" -> jsDocObjectLiteral(schema);
			default -> "any";
		};
	}

	private static String jsDocLiteral(Object value) {
		if (value instanceof String text) {
			return quote(text);
		}
		if (value instanceof Number number) {
			return formatNumber(number);
		}
		if (value instanceof Boolean bool) {
			return bool.toString();
		}
		return "any";
	}

	private static String formatNumber(Number number) {
		if (number == null) {
			return "0";
		}
		if (number instanceof Byte || number instanceof Short || number instanceof Integer || number instanceof Long) {
			return number.toString();
		}
		double value = number.doubleValue();
		if (Math.rint(value) == value) {
			return Long.toString((long) value);
		}
		return Double.toString(value);
	}

	private static String schemaTypeName(Map<String, Object> schema) {
		Object type = schema.get("type");
		return type instanceof String name ? name : "";
	}

	private static String jsPropertyName(String name) {
		if (isValidJSIdentifier(name)) {
			return name;
		}
		return quote(name);
	}

	private static String jsPropertyPath(String root, String name) {
		if (root == null || root.isEmpty()) {
			return jsPropertyName(name);
		}
		if (isValidJSIdentifier(name)) {
			return root + "." + name;
		}
		return root + "[" + quote(name) + "]";
	}

	private static boolean isValidJSIdentifier(String name) {
		if (name == null || name.isEmpty()) {
			return false;
		}
		for (int i = 0; i < name.length(); i++) {
			char ch = name.charAt(i);
			if (i == 0) {
				if (ch != '_' && ch != '$' && !Character.isLetter(ch)) {
					return false;
				}
				continue;
			}
			if (ch != '_' && ch != '$' && !Character.isLetterOrDigit(ch)) {
				return false;
			}
		}
		return true;
	}

	private static String quote(String text) {
		return '"' + text.replace("\\", "\\\\").replace("\"", "\\\"") + '"';
	}

	private static String sanitizeJSDocText(String text) {
		if (text == null) {
			return "";
		}
		return text.replace("*/", "* /").replace("\n", " ").trim();
	}

	private Map<String, Object> toSchemaMap(McpSchema.JsonSchema schema) {
		if (schema == null) {
			return Map.of();
		}
		try {
			return this.objectMapper.convertValue(schema, new TypeReference<Map<String, Object>>() {
				// empty
			});
		}
		catch (IllegalArgumentException e) {
			return Map.of();
		}
	}

	public record ToolEntry(String name, String callable, String description, Map<String, Object> inputSchema,
			Map<String, Object> outputSchema) {
	}

}
