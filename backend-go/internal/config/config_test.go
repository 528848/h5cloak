package config

import (
	"os"
	"testing"
)

func TestLoadDefaultsWithoutFile(t *testing.T) {
	cfg, err := Load("gateway", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTP.Addr != "127.0.0.1:8090" {
		t.Fatalf("addr = %s", cfg.HTTP.Addr)
	}
	if cfg.App.Name != "cloak-gateway" {
		t.Fatalf("name = %s", cfg.App.Name)
	}
}

func TestEnvOverride(t *testing.T) {
	t.Setenv("CLOAK_HTTP_ADDR", "0.0.0.0:1234")
	t.Setenv("CLOAK_REDIS_ADDR", "redis:6379")
	cfg, err := Load("admin", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTP.Addr != "0.0.0.0:1234" || cfg.Redis.Addr != "redis:6379" {
		t.Fatalf("env override failed: %+v", cfg.HTTP)
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	yaml := "app:\n  env: test\nhttp:\n  addr: 127.0.0.1:9999\nlog:\n  level: warn\n"
	if err := os.WriteFile(dir+"/worker.yaml", []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load("worker", dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.App.Env != "test" || cfg.HTTP.Addr != "127.0.0.1:9999" || cfg.Log.Level != "warn" {
		t.Fatalf("unexpected %+v", cfg)
	}
}
