package model

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type VLMClient struct {
	BaseURL string
	Model   string
	Client  *http.Client
}

func NewVLMClient(baseURL, modelName string) *VLMClient {
	return &VLMClient{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Model:   modelName,
		Client:  &http.Client{Timeout: 60 * time.Second},
	}
}

type OllamaGenerateRequest struct {
	Model  string   `json:"model"`
	Prompt string   `json:"prompt"`
	Images []string `json:"images"` // Base64 encoded images
	Stream bool     `json:"stream"`
}

type OllamaGenerateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

func (v *VLMClient) AnalyzeImage(ctx context.Context, prompt, imagePath string) (string, error) {
	// 1. Read and encode image to base64
	imgData, err := os.ReadFile(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to read image: %w", err)
	}
	encodedImg := base64.StdEncoding.EncodeToString(imgData)

	// 2. Prepare request
	reqBody := OllamaGenerateRequest{
		Model:  v.Model,
		Prompt: prompt,
		Images: []string{encodedImg},
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	// 3. Send to Ollama
	url := fmt.Sprintf("%s/api/generate", v.BaseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := v.Client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama error: %d - %s", resp.StatusCode, string(body))
	}

	// 4. Parse response
	var ollamaResp OllamaGenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "", err
	}

	return ollamaResp.Response, nil
}
