package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	System     SystemConfig     `yaml:"system"`
	Agents     []AgentConfig    `yaml:"agents"`
	SuperAgent SuperAgentConfig `yaml:"super_agent"`
}

type SystemConfig struct {
	EmbeddingModel string `yaml:"embedding_model"`
	EmbeddingDim   int    `yaml:"embedding_dim"` // e.g., 384, 768, 1024
	VisionModel    string `yaml:"vision_model"`
}

type AgentConfig struct {
	Name           string   `yaml:"name"`
	ID             string   `yaml:"id"`
	Description    string   `yaml:"description"`
	Instruction    string   `yaml:"instruction"`
	KnowledgeBases []string `yaml:"knowledge_bases"`
}

type SuperAgentConfig struct {
	Name        string `yaml:"name"`
	Model       string `yaml:"model"`
	Instruction string `yaml:"instruction"`
}

func LoadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Config
	decoder := yaml.NewDecoder(f)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, err
	}
	// Set defaults if missing
	if cfg.System.EmbeddingModel == "" {
		cfg.System.EmbeddingModel = "all-minilm"
	}
	if cfg.System.EmbeddingDim == 0 {
		cfg.System.EmbeddingDim = 384
	}
	if cfg.System.VisionModel == "" {
		cfg.System.VisionModel = "llava" // Fallback
	}
	return &cfg, nil
}
