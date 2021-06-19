package config

import (
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/BurntSushi/toml"
)

type DBType string

const (
	DBPostgres   DBType = "postgres"
	DBClickHouse DBType = "clickhouse"
)

type FlagType string

const (
	FlagTimerange FlagType = "timerange"
)

type FlagConfig struct {
	Required    bool
	Default     *string
	Description string
	Type        FlagType
}

type SelectorConfig struct {
	Query       string                `toml:"query"`
	Flags       map[string]FlagConfig `toml:"flag"`
	Description string                `toml:"description"`
	Include     string                `toml:"include"`
	UseUTC      bool                  `toml:"use_utc"`
}

type EnvConfig struct {
	IsDefault bool   `toml:"is_default"`
	Endpoint  string `toml:"endpoint"`
	Type      DBType `toml:"type"`
	Database  string `toml:"database"`
	User      string `toml:"user"`
	Password  string `toml:"password"`
	Include   string `toml:"include"`
}

type SelektorConfig struct {
	Selectors map[string]SelectorConfig `toml:"selector"`
	Envs      map[string]EnvConfig      `toml:"env"`
}

func (ec EnvConfig) String() string {
	bytes, err := json.Marshal(ec)
	if err != nil {
		return fmt.Sprintf("Failed to form JSON for EnvConfig: %s", err.Error())
	}
	return string(bytes)
}

func (sc SelectorConfig) String() string {
	bytes, err := json.Marshal(sc)
	if err != nil {
		return fmt.Sprintf("Failed to form JSON for SelectorConfig: %s", err.Error())
	}
	return string(bytes)
}

func (ec SelektorConfig) String() string {
	bytes, err := json.MarshalIndent(ec, "", "    ")
	if err != nil {
		return fmt.Sprintf("Failed to form JSON for EGConfig: %s", err.Error())
	}
	return string(bytes)
}

func processIncludes(cfg *SelektorConfig, cfgPath string) error {
	cfgDir := path.Dir(cfgPath)

	for k, v := range cfg.Selectors {
		if v.Include != "" {
			includedConfig := SelectorConfig{}
			includePath := v.Include
			if !path.IsAbs(includePath) {
				includePath = path.Join(cfgDir, includePath)
			}

			_, err := toml.DecodeFile(includePath, &includedConfig)
			if err != nil {
				return fmt.Errorf("failed to read include for selector='%s': %w", k, err)
			}

			cfg.Selectors[k] = includedConfig
		}
	}

	for k, v := range cfg.Envs {
		if v.Include != "" {
			includedConfig := EnvConfig{}
			includePath := v.Include
			if !path.IsAbs(includePath) {
				includePath = path.Join(cfgDir, includePath)
			}

			_, err := toml.DecodeFile(includePath, &includedConfig)
			if err != nil {
				return fmt.Errorf("failed to read include for env='%s': %w", k, err)
			}

			cfg.Envs[k] = includedConfig
		}
	}

	return nil
}

func verifyConfig(cfg *SelektorConfig) error {
	defaults := []string{}
	for k, v := range cfg.Envs {
		if v.IsDefault {
			defaults = append(defaults, k)
		}
	}

	if len(defaults) > 1 {
		return fmt.Errorf("it is allowed to have only one default env, found multiple: %s", strings.Join(defaults, ", "))
	}

	return nil
}

func ReadConfig(cfgPath string) (*SelektorConfig, error) {
	cfg := &SelektorConfig{}

	if _, err := toml.DecodeFile(cfgPath, cfg); err != nil {
		return nil, err
	}

	if err := processIncludes(cfg, cfgPath); err != nil {
		return nil, err
	}

	if err := verifyConfig(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
