package main

import (
	"fmt"
	"os"

	"github.com/josexy/mini-ss/config"
	"gopkg.in/yaml.v3"
)

func writeYaml(path string, cfg *config.Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func formatBytes(bytesPerSecond float64) string {
	units := []string{"B/s", "KB/s", "MB/s", "GB/s", "TB/s"}
	index := 0
	for bytesPerSecond >= 1024 && index < len(units)-1 {
		bytesPerSecond /= 1024
		index++
	}
	return fmt.Sprintf("%.2f %s", bytesPerSecond, units[index])
}
