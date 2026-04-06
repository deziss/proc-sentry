package config

import (
	"os"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	// Clear any env vars that could interfere
	for _, k := range []string{"METRICS_PORT", "TOP_N", "SCRAPE_INTERVAL", "ENABLE_DISK_IO", "ENABLE_PORTS", "PROCFS_PATH", "PROC_HOSTNAME"} {
		os.Unsetenv(k)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != "9105" {
		t.Errorf("expected port 9105, got %s", cfg.Port)
	}
	if cfg.TopN != 40 {
		t.Errorf("expected TopN 40, got %d", cfg.TopN)
	}
	if !cfg.EnableDiskIO {
		t.Error("expected EnableDiskIO true")
	}
	if !cfg.EnablePorts {
		t.Error("expected EnablePorts true")
	}
	if cfg.ProcFSPath != "/proc" {
		t.Errorf("expected ProcFSPath /proc, got %s", cfg.ProcFSPath)
	}
}

func TestLoadTopNValidation(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid", "10", false},
		{"min", "1", false},
		{"max", "500", false},
		{"zero", "0", true},
		{"negative", "-5", true},
		{"too_large", "501", true},
		{"non_numeric", "abc", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("TOP_N", tt.value)
			defer os.Unsetenv("TOP_N")

			_, err := Load()
			if (err != nil) != tt.wantErr {
				t.Errorf("TOP_N=%q: wantErr=%v, got err=%v", tt.value, tt.wantErr, err)
			}
		})
	}
}

func TestLoadScrapeInterval(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid_5s", "5s", false},
		{"valid_1m", "1m", false},
		{"too_short", "500ms", true},
		{"too_long", "10m", true},
		{"invalid", "not-a-duration", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv("TOP_N")
			os.Setenv("SCRAPE_INTERVAL", tt.value)
			defer os.Unsetenv("SCRAPE_INTERVAL")

			_, err := Load()
			if (err != nil) != tt.wantErr {
				t.Errorf("SCRAPE_INTERVAL=%q: wantErr=%v, got err=%v", tt.value, tt.wantErr, err)
			}
		})
	}
}

func TestLoadDisableFeatures(t *testing.T) {
	os.Unsetenv("TOP_N")
	os.Unsetenv("SCRAPE_INTERVAL")
	os.Setenv("ENABLE_DISK_IO", "false")
	os.Setenv("ENABLE_PORTS", "false")
	defer os.Unsetenv("ENABLE_DISK_IO")
	defer os.Unsetenv("ENABLE_PORTS")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.EnableDiskIO {
		t.Error("expected EnableDiskIO false")
	}
	if cfg.EnablePorts {
		t.Error("expected EnablePorts false")
	}
}

func TestLoadHostnameOverride(t *testing.T) {
	os.Unsetenv("TOP_N")
	os.Unsetenv("SCRAPE_INTERVAL")
	os.Setenv("PROC_HOSTNAME", "test-node-01")
	defer os.Unsetenv("PROC_HOSTNAME")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Hostname != "test-node-01" {
		t.Errorf("expected hostname test-node-01, got %s", cfg.Hostname)
	}
}
