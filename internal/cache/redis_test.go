package cache

import (
	"testing"

	"smctf/internal/config"
)

func TestNew(t *testing.T) {
	cfg := config.RedisConfig{
		Addr:     "localhost:6379",
		Password: "test-password",
		DB:       1,
		PoolSize: 10,
	}

	client := New(cfg)
	if client == nil {
		t.Fatal("expected non-nil client")
	}

	opts := client.Options()
	if opts.Addr != cfg.Addr {
		t.Errorf("expected Addr %s, got %s", cfg.Addr, opts.Addr)
	}

	if opts.Password != cfg.Password {
		t.Errorf("expected Password %s, got %s", cfg.Password, opts.Password)
	}

	if opts.DB != cfg.DB {
		t.Errorf("expected DB %d, got %d", cfg.DB, opts.DB)
	}

	if opts.PoolSize != cfg.PoolSize {
		t.Errorf("expected PoolSize %d, got %d", cfg.PoolSize, opts.PoolSize)
	}
}

func TestNew_DefaultValues(t *testing.T) {
	cfg := config.RedisConfig{
		Addr: "localhost:6379",
	}

	client := New(cfg)
	if client == nil {
		t.Fatal("expected non-nil client")
	}

	opts := client.Options()
	if opts.Addr != cfg.Addr {
		t.Errorf("expected Addr %s, got %s", cfg.Addr, opts.Addr)
	}

	if opts.DB != 0 {
		t.Errorf("expected default DB 0, got %d", opts.DB)
	}
}
