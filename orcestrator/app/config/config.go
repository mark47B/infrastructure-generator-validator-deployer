package config

import "time"

type Config struct {
	Server   HTTPServerConfig `json:"server"`
	LLM      LLMConfig        `json:"llm"`
	Mongo    MongoConfig
	FileRepo FileRepoConfig
}

type HTTPServerConfig struct {
	Host         string        `json:"host" default:"0.0.0.0"`
	Port         int           `json:"port" default:"8080"`
	ReadTimeout  time.Duration `json:"read_timeout" default:"120s"`
	WriteTimeout time.Duration `json:"write_timeout" default:"120s"`
}

type LLMConfig struct {
	APIKey    string        `json:"api_key" required:"true"`
	BaseURL   string        `json:"base_url" default:"https://kong-proxy.yc.amvera.ru/api/v1/models/gpt"`
	Model     string        `json:"model" default:"gpt-5"`
	MaxTokens int           `json:"max_tokens" default:"4000"`
	Timeout   time.Duration `json:"timeout" default:"60s"`
}

type MongoConfig struct {
	URI      string `json:"uri" required:"true"`
	Database string `json:"database" required:"true"`
}

type FileRepoConfig struct {
	ConfigDir string `json:"config_dir" default:"./deployments"`
}
