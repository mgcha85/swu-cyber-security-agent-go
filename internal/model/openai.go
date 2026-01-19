package model

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"
	"strings"

	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

type OpenAIMod struct {
	BaseURL string
	APIKey  string
	Model   string
	Client  *http.Client
}

func NewOpenAIModel(baseURL, apiKey, modelName string) *OpenAIMod {
	return &OpenAIMod{
		BaseURL: strings.TrimRight(baseURL, "/"),
		APIKey:  apiKey,
		Model:   modelName,
		Client:  &http.Client{},
	}
}

func (m *OpenAIMod) Name() string {
	return m.Model
}

// Simple request structure for OpenAI Chat Completions
type OpenAIChatRequest struct {
	Model    string          `json:"model"`
	Messages []OpenAIMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (m *OpenAIMod) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		// 1. Convert ADK Request to OpenAI Messages
		var messages []OpenAIMessage

		// Typically ADK sends a history + new message.
		// For simplicity, we'll flatten the Parts into a single USER message if it's just text.
		// A full implementation would map roles properly.
		var combinedText strings.Builder
		if req.Contents != nil {
			for _, content := range req.Contents {
				for _, part := range content.Parts {
					if part.Text != "" {
						combinedText.WriteString(part.Text + "\n")
					}
				}
			}
		}

		messages = append(messages, OpenAIMessage{
			Role:    "user",
			Content: combinedText.String(),
		})

		// 2. Prepare Request
		openAIReq := OpenAIChatRequest{
			Model:    m.Model,
			Messages: messages,
			Stream:   false, // Streaming not fully implemented in this adapter yet
		}

		jsonData, err := json.Marshal(openAIReq)
		if err != nil {
			yield(nil, err)
			return
		}

		// 3. Send HTTP Request
		url := fmt.Sprintf("%s/v1/chat/completions", m.BaseURL)
		// Handle if user provided full v1 URL or base
		if strings.HasSuffix(m.BaseURL, "/v1") {
			url = fmt.Sprintf("%s/chat/completions", m.BaseURL)
		}

		httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			yield(nil, err)
			return
		}

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", m.APIKey))

		resp, err := m.Client.Do(httpReq)
		if err != nil {
			yield(nil, err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			yield(nil, fmt.Errorf("OpenAI API error: %s - %s", resp.Status, string(bodyBytes)))
			return
		}

		// 4. Parse Response
		var openAIResp OpenAIChatResponse
		if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
			yield(nil, err)
			return
		}

		if len(openAIResp.Choices) == 0 {
			yield(nil, fmt.Errorf("no choices in response"))
			return
		}

		// 5. Convert back to ADK Response
		adkResp := &model.LLMResponse{
			Content: &genai.Content{
				Parts: []*genai.Part{
					{Text: openAIResp.Choices[0].Message.Content},
				},
			},
		}

		yield(adkResp, nil)
	}
}
