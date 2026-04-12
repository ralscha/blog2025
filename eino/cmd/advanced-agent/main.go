package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	"einoexamples/internal/shared"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/compose"
)

type graphInput struct {
	Question string
}

type routeDecision struct {
	Question string
	Route    string
}

type teamDraft struct {
	Question string
	Draft    string
}

func main() {
	question := flag.String("question", "Design a lightweight customer-support agent and explain when it should use a specialist team instead of answering directly.", "Question for the graph-based multi-agent example")
	flag.Parse()

	ctx := context.Background()
	chatModel, err := shared.NewChatModel(ctx)
	if err != nil {
		log.Fatal(err)
	}

	routerAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "router-agent",
		Description: "Chooses whether a request should go to a direct responder or to the full analysis team.",
		Instruction: "You route requests. Reply with exactly one lowercase word: direct or team. Use team for multi-step, architectural, comparative, or strategy questions. Use direct for simple factual questions or short explanations.",
		Model:       chatModel,
	})
	if err != nil {
		log.Fatal(err)
	}

	directAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "direct-agent",
		Description: "Answers straightforward requests directly.",
		Instruction: "You are the fast-response agent. Answer directly in one or two tight paragraphs. Do not mention other agents.",
		Model:       chatModel,
	})
	if err != nil {
		log.Fatal(err)
	}

	plannerAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "planner-agent",
		Description: "Breaks a complex request into a concrete plan.",
		Instruction: "You are the planning agent. Do not answer the user directly. Produce a compact plan with sections for objectives, approach, and assumptions.",
		Model:       chatModel,
	})
	if err != nil {
		log.Fatal(err)
	}

	researcherAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "researcher-agent",
		Description: "Finds supporting points, examples, and implementation ideas.",
		Instruction: "You are the research agent. Use the existing conversation as context. Add supporting details, examples, and practical implementation suggestions. Do not write the final answer.",
		Model:       chatModel,
	})
	if err != nil {
		log.Fatal(err)
	}

	criticAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "critic-agent",
		Description: "Challenges weak assumptions and identifies tradeoffs.",
		Instruction: "You are the critic agent. Use the existing conversation as context. Identify risks, tradeoffs, missing assumptions, and edge cases. Do not write the final answer.",
		Model:       chatModel,
	})
	if err != nil {
		log.Fatal(err)
	}

	drafterAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "drafter-agent",
		Description: "Synthesizes the planning and review outputs into a strong draft.",
		Instruction: "You are the synthesis agent. Use the prior agent outputs in the conversation history to draft a substantial answer. Include assumptions when certainty is low.",
		Model:       chatModel,
	})
	if err != nil {
		log.Fatal(err)
	}

	writerAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "writer-agent",
		Description: "Polishes a draft into the final answer.",
		Instruction: "You are the final writer. Rewrite the provided draft into a polished answer with sections titled Answer, Why, and Next Steps. Keep it concise but useful.",
		Model:       chatModel,
	})
	if err != nil {
		log.Fatal(err)
	}

	reviewTeam, err := adk.NewParallelAgent(ctx, &adk.ParallelAgentConfig{
		Name:        "review-team",
		Description: "Runs the researcher and critic in parallel so the draft gets both supporting detail and pushback.",
		SubAgents:   []adk.Agent{researcherAgent, criticAgent},
	})
	if err != nil {
		log.Fatal(err)
	}

	analysisTeam, err := adk.NewSequentialAgent(ctx, &adk.SequentialAgentConfig{
		Name:        "analysis-team",
		Description: "For complex requests, plan the work, review it in parallel, and synthesize a draft.",
		SubAgents:   []adk.Agent{plannerAgent, reviewTeam, drafterAgent},
	})
	if err != nil {
		log.Fatal(err)
	}

	graph, err := buildAdvancedGraph(ctx, routerAgent, directAgent, analysisTeam, writerAgent)
	if err != nil {
		log.Fatal(err)
	}

	printTopology()

	answer, err := graph.Invoke(ctx, graphInput{Question: strings.TrimSpace(*question)})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("\n[final answer]")
	fmt.Println(answer)
}

func buildAdvancedGraph(
	ctx context.Context,
	routerAgent adk.Agent,
	directAgent adk.Agent,
	analysisTeam adk.Agent,
	writerAgent adk.Agent,
) (compose.Runnable[graphInput, string], error) {
	graph := compose.NewGraph[graphInput, string]()

	routerNode := compose.InvokableLambda(func(ctx context.Context, input graphInput) (routeDecision, error) {
		runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: routerAgent, EnableStreaming: true})
		reply, err := runAgentNode(ctx, "router-agent", runner, input.Question)
		if err != nil {
			return routeDecision{}, err
		}

		return routeDecision{
			Question: input.Question,
			Route:    normalizeRoute(reply),
		}, nil
	})

	directNode := compose.InvokableLambda(func(ctx context.Context, input routeDecision) (string, error) {
		runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: directAgent, EnableStreaming: true})
		return runAgentNode(ctx, "direct-agent", runner, input.Question)
	})

	analysisNode := compose.InvokableLambda(func(ctx context.Context, input routeDecision) (teamDraft, error) {
		runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: analysisTeam, EnableStreaming: true})
		draft, err := runAgentNode(ctx, "analysis-team", runner, input.Question)
		if err != nil {
			return teamDraft{}, err
		}

		return teamDraft{
			Question: input.Question,
			Draft:    draft,
		}, nil
	})

	finalWriterNode := compose.InvokableLambda(func(ctx context.Context, input teamDraft) (string, error) {
		runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: writerAgent, EnableStreaming: true})
		prompt := fmt.Sprintf("Original question:\n%s\n\nDraft to polish:\n%s", input.Question, input.Draft)
		return runAgentNode(ctx, "writer-agent", runner, prompt)
	})

	if err := graph.AddLambdaNode("router", routerNode); err != nil {
		return nil, err
	}
	if err := graph.AddLambdaNode("direct_answer", directNode); err != nil {
		return nil, err
	}
	if err := graph.AddLambdaNode("analysis_team", analysisNode); err != nil {
		return nil, err
	}
	if err := graph.AddLambdaNode("final_writer", finalWriterNode); err != nil {
		return nil, err
	}

	if err := graph.AddEdge(compose.START, "router"); err != nil {
		return nil, err
	}
	if err := graph.AddBranch("router", compose.NewGraphBranch(func(_ context.Context, decision routeDecision) (string, error) {
		if decision.Route == "direct" {
			return "direct_answer", nil
		}
		return "analysis_team", nil
	}, map[string]bool{
		"direct_answer": true,
		"analysis_team": true,
	})); err != nil {
		return nil, err
	}
	if err := graph.AddEdge("direct_answer", compose.END); err != nil {
		return nil, err
	}
	if err := graph.AddEdge("analysis_team", "final_writer"); err != nil {
		return nil, err
	}
	if err := graph.AddEdge("final_writer", compose.END); err != nil {
		return nil, err
	}

	return graph.Compile(ctx, compose.WithGraphName("advanced-agent-graph"))
}

func runAgentNode(ctx context.Context, name string, runner *adk.Runner, prompt string) (string, error) {
	fmt.Printf("\n[%s]\n", name)
	return shared.PrintAgentEvents(runner.Query(ctx, prompt))
}

func normalizeRoute(reply string) string {
	compact := strings.ToLower(strings.TrimSpace(reply))
	compact = strings.Trim(compact, ".!?:; \t\n\r")
	if strings.Contains(compact, "direct") {
		return "direct"
	}
	return "team"
}

func printTopology() {
	fmt.Println("Graph topology:")
	fmt.Println("  start -> router")
	fmt.Println("  router -> direct_answer -> end")
	fmt.Println("  router -> analysis_team -> final_writer -> end")
	fmt.Println()
	fmt.Println("Analysis team topology:")
	fmt.Println("  planner-agent -> review-team(parallel: researcher-agent, critic-agent) -> drafter-agent")
	fmt.Println()
}
