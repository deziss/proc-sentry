package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port           string
	TopN           int
	UpdateInterval time.Duration
	EnableDiskIO   bool
	EnablePorts    bool
	ProcFSPath     string
	Hostname       string
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:           "9105",
		TopN:           40,
		UpdateInterval: 5 * time.Second,
		EnableDiskIO:   true,
		EnablePorts:    true,
		ProcFSPath:     "/proc",
	}

	if v := os.Getenv("METRICS_PORT"); v != "" {
		cfg.Port = v
	}

	if v := os.Getenv("TOP_N"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("invalid TOP_N value %q: %w", v, err)
		}
		if n < 1 || n > 500 {
			return nil, fmt.Errorf("TOP_N must be between 1 and 500, got %d", n)
		}
		cfg.TopN = n
	}

	if v := os.Getenv("SCRAPE_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return nil, fmt.Errorf("invalid SCRAPE_INTERVAL %q: %w", v, err)
		}
		if d < 1*time.Second || d > 5*time.Minute {
			return nil, fmt.Errorf("SCRAPE_INTERVAL must be between 1s and 5m, got %s", d)
		}
		cfg.UpdateInterval = d
	}

	if v := os.Getenv("ENABLE_DISK_IO"); v == "false" {
		cfg.EnableDiskIO = false
	}

	if v := os.Getenv("ENABLE_PORTS"); v == "false" {
		cfg.EnablePorts = false
	}

	if v := os.Getenv("PROCFS_PATH"); v != "" {
		cfg.ProcFSPath = v
	}

	if v := os.Getenv("PROC_HOSTNAME"); v != "" {
		cfg.Hostname = v
	} else {
		h, err := os.Hostname()
		if err != nil {
			cfg.Hostname = "unknown"
		} else {
			cfg.Hostname = h
		}
	}

	return cfg, nil
}
