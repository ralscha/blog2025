package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"einoexamples/internal/shared"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/middlewares/skill"
	goyaml "gopkg.in/yaml.v3"
)

func main() {
	question := flag.String("question", "Use the skill tool with skill=\"incident-triage\" and produce a first-response plan for a p95 latency spike on the checkout API right after today's deploy.", "Question for the skill-enabled agent")
	skillsDirFlag := flag.String("skills-dir", filepath.Join("skills"), "Directory containing skill subdirectories")
	flag.Parse()

	ctx := context.Background()
	chatModel, err := shared.NewChatModel(ctx)
	if err != nil {
		log.Fatal(err)
	}

	skillsDir, err := filepath.Abs(strings.TrimSpace(*skillsDirFlag))
	if err != nil {
		log.Fatal(err)
	}

	backend := &diskSkillBackend{baseDir: skillsDir}
	skillMiddleware, err := skill.NewMiddleware(ctx, &skill.Config{Backend: backend})
	if err != nil {
		log.Fatal(err)
	}

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "skill-agent",
		Description: "An agent that can discover and load reusable Eino skills from SKILL.md files.",
		Instruction: "You are a helpful assistant. When a matching skill exists, use it before answering.",
		Model:       chatModel,
		Handlers:    []adk.ChatModelAgentMiddleware{skillMiddleware},
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Skills dir: %s\n", skillsDir)

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	prompt := strings.TrimSpace(*question)
	if _, err := shared.PrintQueryAgentEvents(prompt, runner.Query(ctx, prompt)); err != nil {
		log.Fatal(err)
	}
}

type diskSkillBackend struct {
	baseDir string
}

func (b *diskSkillBackend) List(ctx context.Context) ([]skill.FrontMatter, error) {
	_ = ctx

	entries, err := os.ReadDir(b.baseDir)
	if err != nil {
		return nil, fmt.Errorf("read skills directory: %w", err)
	}

	matters := make([]skill.FrontMatter, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillPath := filepath.Join(b.baseDir, entry.Name(), "SKILL.md")
		sk, err := loadSkillFile(skillPath)
		if err != nil {
			return nil, err
		}
		matters = append(matters, sk.FrontMatter)
	}

	return matters, nil
}

func (b *diskSkillBackend) Get(ctx context.Context, name string) (skill.Skill, error) {
	_ = ctx

	entries, err := os.ReadDir(b.baseDir)
	if err != nil {
		return skill.Skill{}, fmt.Errorf("read skills directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillPath := filepath.Join(b.baseDir, entry.Name(), "SKILL.md")
		sk, err := loadSkillFile(skillPath)
		if err != nil {
			return skill.Skill{}, err
		}
		if sk.Name == name {
			return sk, nil
		}
	}

	return skill.Skill{}, fmt.Errorf("skill not found: %s", name)
}

func loadSkillFile(skillPath string) (skill.Skill, error) {
	content, err := os.ReadFile(skillPath)
	if err != nil {
		return skill.Skill{}, fmt.Errorf("read %s: %w", skillPath, err)
	}

	frontMatter, body, err := splitFrontMatter(string(content))
	if err != nil {
		return skill.Skill{}, fmt.Errorf("parse %s: %w", skillPath, err)
	}

	var matter skill.FrontMatter
	if err := goyaml.Unmarshal([]byte(frontMatter), &matter); err != nil {
		return skill.Skill{}, fmt.Errorf("unmarshal front matter for %s: %w", skillPath, err)
	}

	return skill.Skill{
		FrontMatter:   matter,
		Content:       strings.TrimSpace(body),
		BaseDirectory: filepath.Dir(skillPath),
	}, nil
}

func splitFrontMatter(content string) (string, string, error) {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	if !strings.HasPrefix(content, "---\n") {
		return "", "", errors.New("missing YAML front matter")
	}

	rest := strings.TrimPrefix(content, "---\n")
	before, after, ok := strings.Cut(rest, "\n---\n")
	if !ok {
		return "", "", errors.New("missing closing front matter delimiter")
	}

	frontMatter := before
	body := after
	return frontMatter, body, nil
}
