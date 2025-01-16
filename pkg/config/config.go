package config

import (
	// stdlib
	"fmt"
	"github.com/pelletier/go-toml/v2"
	"os"
)

type ConfigFile struct {
	Model     ModelConfig
	Backend   BackendConfig
	Webserver WebserverConfig
	Logging   LoggingConfig
	Input     InputConfig
}

type ModelConfig struct {
	Format     string
	Path       string
	ConfigPath string
}

type BackendConfig struct {
	Device string
}

type InputConfig struct {
	Type string
	Path string
}

type WebserverConfig struct {
	Port               uint
	ReadTimeoutSec     uint
	WriteTimeoutSec    uint
	ShutdownTimeoutSec uint
}

type LoggingConfig struct {
	Level         string
	StatPeriodSec uint
}

func Unmarshal(file_path string) (*ConfigFile, error) {
	config_file := new(ConfigFile)
	data, err := os.ReadFile(file_path)
	if err != nil {
		return nil,
			fmt.Errorf("Unable to read %s error: %w", file_path, err)
	}
	err = toml.Unmarshal(data, config_file)
	if err != nil {
		return nil,
			fmt.Errorf("Unable to unmarshal %s error: %w", file_path, err)
	}
	return config_file, nil
}
