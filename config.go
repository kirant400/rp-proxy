package main

import (
	"encoding/json"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// ===== Config Structs =====
type AllowedPath struct {
	Path    string   `json:"path" yaml:"path"`
	Methods []string `json:"methods" yaml:"methods"` // e.g. ["GET","POST"]
}

type Endpoint struct {
	Name        string        `json:"name" yaml:"name"`
	TargetURL   string        `json:"target_url" yaml:"target_url"`
	AuthType    string        `json:"auth_type" yaml:"auth_type"` // "bearer" or "none"
	AuthAPI     string        `json:"auth_api" yaml:"auth_api"`
	Username    string        `json:"username" yaml:"username"`
	PasswordEnc string        `json:"password_enc" yaml:"password_enc"`
	Allowed     []AllowedPath `json:"allowed" yaml:"allowed"`
}

type Config struct {
	Endpoints []Endpoint `json:"endpoints" yaml:"endpoints"`
}

// ===== Load Config =====
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if strings.HasSuffix(path, ".json") {
		err = json.Unmarshal(data, &cfg)
	} else {
		err = yaml.Unmarshal(data, &cfg)
	}
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
