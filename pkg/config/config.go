package config

import (
	"fmt"

	"github.com/spf13/viper"
	v1 "k8s.io/api/core/v1"
)

type Config struct {
	ScriptPath           []string          `mapstructure:"scriptPath"`
	MaxAffectedNodeCount int               `mapstructure:"maxAffectedNodeCount"`
	TargetNodeLabels     map[string]string `mapstructure:"targetNodeLabels"`
	Labels               map[string]string `mapstructure:"labels"`
	Taints               []v1.Taint        `mapstructure:"taints"`
}

func LoadConfig(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// Set default values
	viper.SetDefault("maxAffectedNodeCount", 1)

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unable to decode into struct: %w", err)
	}

	return &cfg, nil
}
