package shared

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

func PrintAgentEvents(events *adk.AsyncIterator[*adk.AgentEvent]) (string, error) {
	return printAgentEvents(events, nil)
}

func PrintQueryAgentEvents(query string, events *adk.AsyncIterator[*adk.AgentEvent]) (string, error) {
	trace := newConversationTrace(query)
	if trace != nil {
		trace.printInitialRequest()
	}
	return printAgentEvents(events, trace)
}

func printAgentEvents(events *adk.AsyncIterator[*adk.AgentEvent], trace *conversationTrace) (string, error) {
	var assistantText strings.Builder

	for {
		event, ok := events.Next()
		if !ok {
			return strings.TrimSpace(assistantText.String()), nil
		}
		if event.Err != nil {
			return "", event.Err
		}

		if event.Output != nil {
			if err := printAgentOutput(event, &assistantText, trace); err != nil {
				return "", err
			}
		}

		if event.Action != nil {
			printAgentAction(event)
		}
	}
}

type conversationTrace struct {
	query                    string
	nextRequestIndex         int
	pendingAssistantToolCall *schema.Message
	pendingToolResults       []*schema.Message
	expectedToolResults      int
}

func newConversationTrace(query string) *conversationTrace {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return nil
	}

	return &conversationTrace{query: trimmed, nextRequestIndex: 1}
}

func (t *conversationTrace) printInitialRequest() {
	if t == nil || t.query == "" {
		return
	}

	t.printRequestBlock(schema.UserMessage(t.query))
}

func (t *conversationTrace) printRequestBlock(messages ...*schema.Message) {
	if t == nil || len(messages) == 0 {
		return
	}

	fmt.Printf("[llm request %d]\n", t.nextRequestIndex)
	t.nextRequestIndex++
	t.printMessages(messages...)
}

func (t *conversationTrace) printAssistantResponseBlock(message *schema.Message, final bool) {
	if t == nil || message == nil {
		return
	}

	label := "assistant response"
	if final {
		label = "assistant final response"
	}

	fmt.Printf("[%s]\n", label)
	t.printMessages(message)
}

func (t *conversationTrace) printMessages(messages ...*schema.Message) {
	printed := false
	for index, message := range messages {
		if message == nil {
			continue
		}
		if printed || index > 0 {
			fmt.Println()
		}
		fmt.Println(strings.TrimSpace(formatTraceMessage(message)))
		printed = true
	}
	if printed {
		fmt.Println()
	}
}

func (t *conversationTrace) recordAssistantToolCall(message *schema.Message) {
	if t == nil || message == nil || len(message.ToolCalls) == 0 {
		return
	}

	t.pendingAssistantToolCall = message
	t.pendingToolResults = nil
	t.expectedToolResults = len(message.ToolCalls)
	t.printAssistantResponseBlock(message, false)
}

func (t *conversationTrace) recordToolResult(message *schema.Message) {
	if t == nil || t.expectedToolResults == 0 || message == nil {
		return
	}

	t.pendingToolResults = append(t.pendingToolResults, message)
	if len(t.pendingToolResults) < t.expectedToolResults {
		return
	}

	requestMessages := make([]*schema.Message, 0, 1+len(t.pendingToolResults))
	requestMessages = append(requestMessages, t.pendingAssistantToolCall)
	requestMessages = append(requestMessages, t.pendingToolResults...)
	t.printRequestBlock(requestMessages...)
	t.pendingAssistantToolCall = nil
	t.pendingToolResults = nil
	t.expectedToolResults = 0
}

func (t *conversationTrace) recordAssistantFinal(message *schema.Message) {
	if t == nil || message == nil {
		return
	}

	t.printAssistantResponseBlock(message, true)
}

func printAgentOutput(event *adk.AgentEvent, assistantText *strings.Builder, trace *conversationTrace) error {
	if event.Output == nil {
		return nil
	}

	if event.Output.MessageOutput != nil {
		if err := printMessageOutput(event, event.Output.MessageOutput, assistantText, trace); err != nil {
			return err
		}
	}

	if event.Output.CustomizedOutput != nil {
		fmt.Printf("%s %v\n", eventLabel(event, "custom output"), event.Output.CustomizedOutput)
	}

	return nil
}

func printMessageOutput(event *adk.AgentEvent, messageOutput *adk.MessageVariant, assistantText *strings.Builder, trace *conversationTrace) error {
	if messageOutput == nil {
		return nil
	}

	if messageOutput.IsStreaming {
		return printStreamingMessageOutput(event, messageOutput, assistantText, trace)
	}

	return printFinalMessage(event, messageOutput, messageOutput.Message, assistantText, false, trace)
}

func printStreamingMessageOutput(event *adk.AgentEvent, messageOutput *adk.MessageVariant, assistantText *strings.Builder, trace *conversationTrace) error {
	messageOutput.MessageStream.SetAutomaticClose()

	frames := make([]*schema.Message, 0, 8)
	printedAssistantContent := false

	for {
		frame, err := messageOutput.MessageStream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if frame == nil {
			continue
		}

		frames = append(frames, frame)

		if messageOutput.Role == schema.Assistant && frame.Content != "" {
			assistantText.WriteString(frame.Content)
			if trace == nil {
				fmt.Print(frame.Content)
				printedAssistantContent = true
			}
		}
	}

	if printedAssistantContent {
		fmt.Println()
	}

	if len(frames) == 0 {
		return nil
	}

	message, err := schema.ConcatMessages(frames)
	if err != nil {
		return err
	}

	return printFinalMessage(event, messageOutput, message, assistantText, true, trace)
}

func printFinalMessage(event *adk.AgentEvent, messageOutput *adk.MessageVariant, message *schema.Message, assistantText *strings.Builder, contentAlreadyPrinted bool, trace *conversationTrace) error {
	if message == nil {
		return nil
	}

	if messageOutput.Role == schema.Assistant && !contentAlreadyPrinted && message.Content != "" {
		assistantText.WriteString(message.Content)
	}

	if trace != nil {
		traceMessage := messageForTrace(messageOutput, message)
		switch {
		case len(traceMessage.ToolCalls) > 0:
			trace.recordAssistantToolCall(traceMessage)
		case messageOutput.Role == schema.Tool:
			trace.recordToolResult(traceMessage)
		case messageOutput.Role == schema.Assistant:
			trace.recordAssistantFinal(traceMessage)
		}

		if messageOutput.Role == schema.Assistant || messageOutput.Role == schema.Tool {
			return nil
		}
	}

	for _, toolCall := range message.ToolCalls {
		fmt.Printf("%s %s(%s)\n", eventLabel(event, "tool call"), toolCall.Function.Name, toolCall.Function.Arguments)
	}

	switch messageOutput.Role {
	case schema.Assistant:
		if !contentAlreadyPrinted && message.Content != "" {
			fmt.Println(message.Content)
		}
		if message.ReasoningContent != "" {
			fmt.Printf("%s %s\n", eventLabel(event, "reasoning"), message.ReasoningContent)
		}
	case schema.Tool:
		fmt.Printf("%s %s\n", eventLabel(event, toolResultLabel(messageOutput, message)), formatMessageDetails(message))
	default:
		fmt.Printf("%s %s\n", eventLabel(event, string(messageOutput.Role)), formatMessageDetails(message))
	}

	return nil
}

func messageForTrace(messageOutput *adk.MessageVariant, message *schema.Message) *schema.Message {
	if message == nil {
		return nil
	}

	clone := *message
	if clone.ToolName == "" {
		clone.ToolName = strings.TrimSpace(messageOutput.ToolName)
	}

	return &clone

}

func printAgentAction(event *adk.AgentEvent) {
	action := event.Action
	if action == nil {
		return
	}

	if action.TransferToAgent != nil {
		fmt.Printf("%s %s\n", eventLabel(event, "transfer"), action.TransferToAgent.DestAgentName)
	}
	if action.Interrupted != nil {
		fmt.Printf("%s %v\n", eventLabel(event, "interrupt"), action.Interrupted.Data)
	}
	if action.BreakLoop != nil {
		fmt.Printf("%s from=%s iteration=%d\n", eventLabel(event, "break-loop"), action.BreakLoop.From, action.BreakLoop.CurrentIterations)
	}
	if action.Exit {
		fmt.Printf("%s\n", eventLabel(event, "exit"))
	}
	if action.CustomizedAction != nil {
		fmt.Printf("%s %v\n", eventLabel(event, "custom action"), action.CustomizedAction)
	}
}

func toolResultLabel(messageOutput *adk.MessageVariant, message *schema.Message) string {
	toolName := strings.TrimSpace(message.ToolName)
	if toolName == "" {
		toolName = strings.TrimSpace(messageOutput.ToolName)
	}
	if toolName == "" {
		return "tool result"
	}
	return fmt.Sprintf("tool result:%s", toolName)
}

func formatMessageDetails(message *schema.Message) string {
	parts := make([]string, 0, 5)

	if message.Content != "" {
		parts = append(parts, message.Content)
	}
	if message.ReasoningContent != "" {
		parts = append(parts, "reasoning: "+message.ReasoningContent)
	}
	if message.ToolCallID != "" {
		parts = append(parts, "tool_call_id="+message.ToolCallID)
	}
	if message.ToolName != "" {
		parts = append(parts, "tool_name="+message.ToolName)
	}
	if message.ResponseMeta != nil {
		parts = append(parts, fmt.Sprintf("finish_reason=%s", message.ResponseMeta.FinishReason))
		if message.ResponseMeta.Usage != nil {
			parts = append(parts, "usage={"+formatUsageDetails(message.ResponseMeta.Usage)+"}")
		}
	}

	formatted := strings.Join(parts, " | ")
	if formatted != "" {
		return formatted
	}

	return strings.TrimSpace(message.String())
}

func formatTraceMessage(message *schema.Message) string {
	if message == nil {
		return ""
	}

	sb := &strings.Builder{}
	sb.WriteString(fmt.Sprintf("%s: %s", message.Role, message.Content))

	if message.ReasoningContent != "" {
		sb.WriteString("\nreasoning content:\n")
		sb.WriteString(message.ReasoningContent)
	}
	if len(message.ToolCalls) > 0 {
		sb.WriteString("\ntool_calls:\n")
		for _, toolCall := range message.ToolCalls {
			sb.WriteString(fmt.Sprintf("%+v\n", toolCall))
		}
	}
	if message.ToolCallID != "" {
		sb.WriteString(fmt.Sprintf("\ntool_call_id: %s", message.ToolCallID))
	}
	if message.ToolName != "" {
		sb.WriteString(fmt.Sprintf("\ntool_call_name: %s", message.ToolName))
	}
	if message.ResponseMeta != nil {
		sb.WriteString(fmt.Sprintf("\nfinish_reason: %s", message.ResponseMeta.FinishReason))
		if message.ResponseMeta.Usage != nil {
			sb.WriteString("\nusage: ")
			sb.WriteString(formatUsageDetails(message.ResponseMeta.Usage))
		}
	}

	return sb.String()
}

func formatUsageDetails(usage *schema.TokenUsage) string {
	if usage == nil {
		return ""
	}

	return fmt.Sprintf(
		"prompt_tokens=%d, prompt_cached_tokens=%d, completion_tokens=%d, completion_reasoning_tokens=%d, total_tokens=%d",
		usage.PromptTokens,
		usage.PromptTokenDetails.CachedTokens,
		usage.CompletionTokens,
		usage.CompletionTokensDetails.ReasoningTokens,
		usage.TotalTokens,
	)
}

func eventLabel(event *adk.AgentEvent, kind string) string {
	context := eventContext(event)
	if context == "" {
		return fmt.Sprintf("[%s]", kind)
	}
	return fmt.Sprintf("[%s %s]", context, kind)
}

func eventContext(event *adk.AgentEvent) string {
	if event == nil {
		return ""
	}

	steps := make([]string, 0, len(event.RunPath))
	for _, step := range event.RunPath {
		name := strings.TrimSpace(step.String())
		if name != "" {
			steps = append(steps, name)
		}
	}
	if len(steps) > 0 {
		return strings.Join(steps, " > ")
	}

	return strings.TrimSpace(event.AgentName)
}
