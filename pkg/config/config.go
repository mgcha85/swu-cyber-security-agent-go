package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Agents     []AgentConfig    `yaml:"agents"`
	SuperAgent SuperAgentConfig `yaml:"super_agent"`
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
	return &cfg, nil
}
