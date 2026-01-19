package model

import (
	"context"
	"iter"

	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

type MockModel struct {
	FixedResponse string
}

func (m *MockModel) Name() string {
	return "mock-model"
}

func (m *MockModel) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		resp := &model.LLMResponse{
			Content: &genai.Content{
				Parts: []*genai.Part{
					{Text: m.FixedResponse},
				},
			},
		}
		yield(resp, nil)
	}
}
