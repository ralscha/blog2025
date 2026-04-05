package ch.rasc.codemode.demo;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.ai.chat.messages.AssistantMessage;
import org.springframework.ai.chat.messages.Message;
import org.springframework.ai.chat.messages.SystemMessage;
import org.springframework.ai.chat.messages.ToolResponseMessage;
import org.springframework.ai.chat.messages.UserMessage;
import org.springframework.ai.chat.model.ChatResponse;
import org.springframework.ai.chat.model.Generation;
import org.springframework.ai.chat.prompt.Prompt;
import org.springframework.ai.openai.OpenAiChatModel;
import org.springframework.ai.openai.OpenAiChatOptions;
import org.springframework.ai.openai.api.OpenAiApi;
import org.springframework.ai.tool.ToolCallback;
import org.springframework.ai.tool.definition.ToolDefinition;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;

import tools.jackson.databind.ObjectMapper;

@Service
public class CodeModeService {

	private static final Logger log = LoggerFactory.getLogger(CodeModeService.class);

	private static final String SEARCH_INPUT_SCHEMA = """
			{
			  "type": "object",
			  "properties": {
			    "query": {
			      "type": "string",
			      "description": "A concise natural-language summary of the task or the helpers you need to find."
			    },
			    "limit": {
			      "type": "integer",
			      "description": "Maximum number of results to return."
			    }
			  },
			  "required": ["query"]
			}
			""";

	private static final String EXECUTE_INPUT_SCHEMA = """
			{
			  "type": "object",
			  "properties": {
			    "code": {
			      "type": "string",
			      "description": "JavaScript body to execute inside a synchronous IIFE. End by returning a value. Do not write comments."
			    }
			  },
			  "required": ["code"]
			}
			""";

	private final OpenAiChatModel chatModel;

	private final ToolCatalog catalog;

	private final JsSandbox sandbox;

	private final ObjectMapper objectMapper;

	@Value("${codemode.max-turns:10}")
	private int maxTurns;

	public CodeModeService(OpenAiChatModel chatModel, ToolCatalog catalog, JsSandbox sandbox,
			ObjectMapper objectMapper) {
		this.chatModel = chatModel;
		this.catalog = catalog;
		this.sandbox = sandbox;
		this.objectMapper = objectMapper;
	}

	public String run(String prompt) {
		List<Message> messages = new ArrayList<>();
		messages.add(new UserMessage(prompt));

		for (int turn = 0; turn < this.maxTurns; turn++) {
			Prompt aiPrompt = buildPrompt(messages, turn);

			log.debug("[turn {}] calling model with {} messages", turn + 1, messages.size());
			ChatResponse response = this.chatModel.call(aiPrompt);

			AssistantMessage assistant = response.getResults().getLast().getOutput();
			messages.addAll(response.getResults().stream().map(Generation::getOutput).toList());

			List<AssistantMessage.ToolCall> toolCalls = assistant.getToolCalls();

			if (toolCalls == null || toolCalls.isEmpty()) {
				log.debug("[turn {}] no tool calls, returning assistant text", turn + 1);
				return assistant.getText();
			}

			List<ToolResponseMessage.ToolResponse> toolResponses = new ArrayList<>();
			for (AssistantMessage.ToolCall call : toolCalls) {
				log.debug("[turn {} tool] name={} input={}", turn + 1, call.name(), call.arguments());
				String result = dispatch(call.name(), call.arguments());
				log.debug("[turn {} result] {}", turn + 1, result);
				toolResponses.add(new ToolResponseMessage.ToolResponse(call.id(), call.name(), result));
			}
			messages.add(ToolResponseMessage.builder().responses(toolResponses).build());
		}

		throw new IllegalStateException("Tool loop exceeded max turns (" + this.maxTurns + ")");
	}

	private static Prompt buildPrompt(List<Message> messages, int turn) {
		boolean firstTurn = turn == 0;
		List<Message> allMessages = new ArrayList<>();
		allMessages.add(new SystemMessage(systemPromptForTurn(turn)));

		allMessages.addAll(messages);

		List<ToolCallback> toolCallbacks = firstTurn ? List.of(definitionOnly(searchToolDefinition()))
				: List.of(definitionOnly(searchToolDefinition()), definitionOnly(executeToolDefinition()));

		OpenAiChatOptions options = OpenAiChatOptions.builder()
			.toolCallbacks(toolCallbacks)
			.toolChoice(firstTurn ? OpenAiApi.ChatCompletionRequest.ToolChoiceBuilder.function("search")
					: OpenAiApi.ChatCompletionRequest.ToolChoiceBuilder.AUTO)
			.internalToolExecutionEnabled(false)
			.build();

		return new Prompt(allMessages, options);
	}

	private static String systemPromptForTurn(int turn) {
		if (turn == 0) {
			return searchOnlySystemPrompt();
		}
		if (turn == 1) {
			return programWritingSystemPrompt();
		}
		return answerOrContinueSystemPrompt();
	}

	private static ToolCallback definitionOnly(ToolDefinition def) {
		return new ToolCallback() {
			@Override
			public ToolDefinition getToolDefinition() {
				return def;
			}

			@Override
			public String call(String input) {
				throw new UnsupportedOperationException("Manual execution only");
			}
		};
	}

	private static ToolDefinition searchToolDefinition() {
		return ToolDefinition.builder()
			.name("search")
			.description("Call this to discover relevant helpers for the user's task. "
					+ "The result includes JavaScript helper definitions you should use in execute(code).")
			.inputSchema(SEARCH_INPUT_SCHEMA)
			.build();
	}

	private static ToolDefinition executeToolDefinition() {
		return ToolDefinition.builder()
			.name("execute")
			.description(
					"""
							Execute synchronous JavaScript only. The sandbox supports ECMAScript 2023 (ES14) and is purely synchronous \
							(no async/await, no Web APIs). Use the helper functions from search, e.g. demo_add_numbers({a:1,b:2}). \
							Keep the code minimal and do not write comments.""")
			.inputSchema(EXECUTE_INPUT_SCHEMA)
			.build();
	}

	private String dispatch(String toolName, String argsJson) {
		return switch (toolName) {
			case "search" -> handleSearch(argsJson);
			case "execute" -> handleExecute(argsJson);
			default -> errorJson("Unknown tool: " + toolName);
		};
	}

	private String handleSearch(String argsJson) {
		try {
			Map<String, Object> args = this.objectMapper.readValue(argsJson, Map.class);
			String query = (String) args.getOrDefault("query", "");
			int limit = args.containsKey("limit") ? ((Number) args.get("limit")).intValue() : 8;
			String apiDefs = this.catalog.search(query, limit);
			return this.objectMapper.writeValueAsString(Map.of("api_definition", apiDefs));
		}
		catch (Exception e) {
			return errorJson("search failed: " + e.getMessage());
		}
	}

	private String handleExecute(String argsJson) {
		try {
			Map<String, Object> args = this.objectMapper.readValue(argsJson, Map.class);
			String code = (String) args.get("code");
			if (code == null || code.isBlank()) {
				return errorJson("'code' is required");
			}
			log.debug("code: {}", code);
			JsSandbox.ExecutionResult result = this.sandbox.execute(code);
			log.debug("result: {}", result);
			return this.objectMapper.writeValueAsString(Map.of("logs", result.logs(), "value", result.value()));
		}
		catch (Exception e) {
			return errorJson("execute failed: " + e.getMessage());
		}
	}

	private String errorJson(String message) {
		try {
			return this.objectMapper.writeValueAsString(Map.of("error", message));
		}
		catch (Exception e2) {
			return "{\"error\":\"serialization error\"}";
		}
	}

	private static String searchOnlySystemPrompt() {
		return """
				You are a helpful assistant. You have access to helpers that can help you answer the user's question.
				Use them if it helps you answer better. Use search to discover the relevant helpers.""".stripIndent()
			.strip();
	}

	private static String programWritingSystemPrompt() {
		return """
				You are a helpful assistant. You have access to helpers that can help you answer the user's question.
				Use them if it helps you answer better.
				Use search to discover the relevant helpers.

				After search returns helper definitions, prefer a single execute(code) call that completes the full computation.
				If one helper's output can be passed directly into another helper inside the same JavaScript snippet,
				do that instead of making multiple execute calls with intermediate results.

				Also when you need certain parts of the response of one helper to decide how to call another helper,
				it's better to do that orchestration in a single execute call with JavaScript, rather than making multiple
				tool calls. For example if the helper returns a list of items and you want to call another helper on each item, it's better to do that
				iteration within the same JavaScript snippet.

				You have a ECMAScript 2023 environment to your disposal in the execute(code) helper, and you can use it to orchestrate
				calls to other helpers as needed.
				"""
			.stripIndent()
			.strip();
	}

	private static String answerOrContinueSystemPrompt() {
		return """
				You are a helpful assistant. You have access to helpers that can help you answer the user's question.
				Use them if it helps you answer better. Use search to discover the relevant helpers.
				""".stripIndent().strip();
	}

}
