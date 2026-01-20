package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"swu-cyber-security-agent-go/internal/model"

	"github.com/joho/godotenv"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/genai"
)

func main() {
	godotenv.Load()
	apiKey := os.Getenv("OPENAI_API_KEY")
	baseURL := os.Getenv("OPENAI_BASE_URL")
	modelName := os.Getenv("OPENAI_MODEL_NAME")

	fmt.Printf("Testing LLM: %s at %s\n", modelName, baseURL)

	llm := model.NewOpenAIModel(baseURL, apiKey, modelName)
	ctx := context.Background()
	llmReq := &adkmodel.LLMRequest{
		Contents: []*genai.Content{{Parts: []*genai.Part{{Text: "Hello, say 'Ready' if you are working."}}}},
	}

	var sb strings.Builder
	for resp, err := range llm.GenerateContent(ctx, llmReq, false) {
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		if resp.Content != nil {
			for _, p := range resp.Content.Parts {
				fmt.Printf("Received part: %s\n", p.Text)
				sb.WriteString(p.Text)
			}
		} else {
			fmt.Println("Received empty content")
		}
	}

	fmt.Printf("Final result: [%s]\n", sb.String())
}
