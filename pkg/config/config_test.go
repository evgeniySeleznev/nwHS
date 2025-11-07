package config

import (
	"os"
	"path/filepath"
	"testing"
)

type testConfig struct {
	HTTP struct {
		Port int `yaml:"port" mapstructure:"port"`
	} `yaml:"http" mapstructure:"http"`
	FeatureFlag bool `yaml:"feature_flag" mapstructure:"feature_flag"`
}

func TestLoaderLoad(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("http:\n  port: 8080\nfeature_flag: false\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := os.Setenv("FEATURE_FLAG", "true"); err != nil {
		t.Fatalf("set env: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Unsetenv("FEATURE_FLAG")
	})

	var cfg testConfig
	loader := New(WithConfigPaths(tmpDir))
	if err := loader.Load(&cfg); err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.HTTP.Port != 8080 {
		t.Fatalf("unexpected port: %d", cfg.HTTP.Port)
	}
	if !cfg.FeatureFlag {
		t.Fatalf("expected feature flag to override from env")
	}
}

func TestLoaderLoadNil(t *testing.T) {
	loader := New()
	if err := loader.Load(nil); err == nil {
		t.Fatalf("expected error for nil dst")
	}
}
