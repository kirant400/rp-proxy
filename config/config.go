// config/config.go
package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

var Global Config

func Load(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("Cannot read config file %s: %v", path, err)
	}

	err = yaml.Unmarshal(data, &Global)
	if err != nil {
		log.Fatalf("Invalid YAML in config file: %v", err)
	}
}