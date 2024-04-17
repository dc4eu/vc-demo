package configuration

import (
	"context"
	"errors"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"vcweb1/pkg/helpers"
	"vcweb1/pkg/logger"
	"vcweb1/pkg/model"

	"github.com/kelseyhightower/envconfig"
)

type envVars struct {
	ConfigYAML string `envconfig:"VC_CONFIG_YAML" required:"true"`
}

// Parse parses config file from VC_CONFIG_YAML environment variable
func Parse(ctx context.Context, logger *logger.Log) (*model.Cfg, error) {
	logger.Info("Read environmental variable")
	env := envVars{}
	if err := envconfig.Process("", &env); err != nil {
		return nil, err
	}

	configPath := env.ConfigYAML

	cfg := &model.Cfg{}

	configFile, err := os.ReadFile(filepath.Clean(configPath))
	if err != nil {
		return nil, err
	}

	fileInfo, err := os.Stat(configPath)
	if err != nil {
		return nil, err
	}

	if fileInfo.IsDir() {
		return nil, errors.New("config is a folder")
	}

	if err := yaml.Unmarshal(configFile, cfg); err != nil {
		return nil, err
	}

	if err := helpers.Check(ctx, cfg, cfg, logger); err != nil {
		return nil, err
	}

	return cfg, nil
}
