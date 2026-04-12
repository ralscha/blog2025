package shared

import (
	"context"
	"fmt"
	"os"
	"strings"

	openai "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
)

func NewChatModel(ctx context.Context) (model.BaseChatModel, error) {
	apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	modelName := strings.TrimSpace(os.Getenv("OPENAI_MODEL"))
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is required")
	}
	if modelName == "" {
		return nil, fmt.Errorf("OPENAI_MODEL is required")
	}

	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:     apiKey,
		Model:      modelName,
		BaseURL:    strings.TrimSpace(os.Getenv("OPENAI_BASE_URL")),
		APIVersion: strings.TrimSpace(os.Getenv("OPENAI_API_VERSION")),
		ByAzure:    strings.EqualFold(strings.TrimSpace(os.Getenv("OPENAI_BY_AZURE")), "true"),
	})
	if err != nil {
		return nil, err
	}

	return chatModel, nil
}
