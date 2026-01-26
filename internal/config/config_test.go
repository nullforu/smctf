package config

import (
	"os"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func TestLoadConfig_Defaults(t *testing.T) {
	os.Clearenv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.AppEnv != "local" {
		t.Errorf("expected AppEnv local, got %s", cfg.AppEnv)
	}

	if cfg.HTTPAddr != ":8080" {
		t.Errorf("expected HTTPAddr :8080, got %s", cfg.HTTPAddr)
	}

	if cfg.AutoMigrate != true {
		t.Error("expected AutoMigrate true")
	}

	if cfg.PasswordBcryptCost != 12 {
		t.Errorf("expected PasswordBcryptCost 12, got %d", cfg.PasswordBcryptCost)
	}

	if cfg.DB.Host != "localhost" {
		t.Errorf("expected DB.Host localhost, got %s", cfg.DB.Host)
	}

	if cfg.DB.Port != 5432 {
		t.Errorf("expected DB.Port 5432, got %d", cfg.DB.Port)
	}

	if cfg.JWT.Issuer != "smctf" {
		t.Errorf("expected JWT.Issuer smctf, got %s", cfg.JWT.Issuer)
	}

	if cfg.JWT.AccessTTL != 24*time.Hour {
		t.Errorf("expected JWT.AccessTTL 24h, got %v", cfg.JWT.AccessTTL)
	}
}

func TestLoadConfig_CustomValues(t *testing.T) {
	os.Clearenv()

	os.Setenv("APP_ENV", "production")
	os.Setenv("HTTP_ADDR", ":9000")
	os.Setenv("AUTO_MIGRATE", "false")
	os.Setenv("BCRYPT_COST", "10")
	os.Setenv("DB_HOST", "db.example.com")
	os.Setenv("DB_PORT", "5433")
	os.Setenv("DB_USER", "custom_user")
	os.Setenv("DB_PASSWORD", "custom_pass")
	os.Setenv("DB_NAME", "custom_db")
	os.Setenv("JWT_SECRET", "custom-secret")
	os.Setenv("JWT_ISSUER", "custom-issuer")
	os.Setenv("JWT_ACCESS_TTL", "2h")
	os.Setenv("JWT_REFRESH_TTL", "48h")
	os.Setenv("FLAG_HMAC_SECRET", "custom-flag-secret")
	os.Setenv("SUBMIT_WINDOW", "30s")
	os.Setenv("SUBMIT_MAX", "5")

	defer os.Clearenv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.AppEnv != "production" {
		t.Errorf("expected AppEnv production, got %s", cfg.AppEnv)
	}

	if cfg.HTTPAddr != ":9000" {
		t.Errorf("expected HTTPAddr :9000, got %s", cfg.HTTPAddr)
	}

	if cfg.AutoMigrate != false {
		t.Error("expected AutoMigrate false")
	}

	if cfg.PasswordBcryptCost != 10 {
		t.Errorf("expected PasswordBcryptCost 10, got %d", cfg.PasswordBcryptCost)
	}

	if cfg.DB.Host != "db.example.com" {
		t.Errorf("expected DB.Host db.example.com, got %s", cfg.DB.Host)
	}

	if cfg.DB.Port != 5433 {
		t.Errorf("expected DB.Port 5433, got %d", cfg.DB.Port)
	}

	if cfg.JWT.Secret != "custom-secret" {
		t.Errorf("expected JWT.Secret custom-secret, got %s", cfg.JWT.Secret)
	}

	if cfg.JWT.AccessTTL != 2*time.Hour {
		t.Errorf("expected JWT.AccessTTL 2h, got %v", cfg.JWT.AccessTTL)
	}

	if cfg.Security.SubmissionWindow != 30*time.Second {
		t.Errorf("expected Security.SubmissionWindow 30s, got %v", cfg.Security.SubmissionWindow)
	}

	if cfg.Security.SubmissionMax != 5 {
		t.Errorf("expected Security.SubmissionMax 5, got %d", cfg.Security.SubmissionMax)
	}
}

func TestLoadConfig_InvalidValues(t *testing.T) {
	tests := []struct {
		name   string
		envKey string
		envVal string
	}{
		{"invalid int", "DB_PORT", "not-a-number"},
		{"invalid bool", "AUTO_MIGRATE", "not-a-bool"},
		{"invalid duration", "JWT_ACCESS_TTL", "invalid-duration"},
		{"bcrypt cost too low", "BCRYPT_COST", "3"},
		{"bcrypt cost too high", "BCRYPT_COST", "32"},
		{"negative db port", "DB_PORT", "-1"},
		{"zero db port", "DB_PORT", "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			os.Setenv(tt.envKey, tt.envVal)
			defer os.Clearenv()

			_, err := Load()
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestLoadConfig_ProductionValidation(t *testing.T) {
	os.Clearenv()
	os.Setenv("APP_ENV", "production")
	defer os.Clearenv()

	_, err := Load()
	if err == nil {
		t.Error("expected error for default secrets in production, got nil")
	}

	os.Setenv("JWT_SECRET", "production-secret-123")
	os.Setenv("FLAG_HMAC_SECRET", "production-flag-secret-456")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed with valid production config: %v", err)
	}

	if cfg.AppEnv != "production" {
		t.Errorf("expected AppEnv production, got %s", cfg.AppEnv)
	}
}

func TestGetEnv(t *testing.T) {
	os.Clearenv()
	defer os.Clearenv()

	if got := getEnv("NONEXISTENT_KEY", "default"); got != "default" {
		t.Errorf("expected default, got %s", got)
	}

	os.Setenv("TEST_KEY", "test_value")
	if got := getEnv("TEST_KEY", "default"); got != "test_value" {
		t.Errorf("expected test_value, got %s", got)
	}
}

func TestGetEnvInt(t *testing.T) {
	os.Clearenv()
	defer os.Clearenv()

	val, err := getEnvInt("NONEXISTENT_KEY", 42)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if val != 42 {
		t.Errorf("expected 42, got %d", val)
	}

	os.Setenv("TEST_INT", "123")
	val, err = getEnvInt("TEST_INT", 42)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if val != 123 {
		t.Errorf("expected 123, got %d", val)
	}

	os.Setenv("TEST_INT", "not-a-number")
	_, err = getEnvInt("TEST_INT", 42)
	if err == nil {
		t.Error("expected error for invalid integer")
	}
}

func TestGetEnvBool(t *testing.T) {
	os.Clearenv()
	defer os.Clearenv()

	val, err := getEnvBool("NONEXISTENT_KEY", true)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if val != true {
		t.Error("expected true")
	}

	tests := []struct {
		input    string
		expected bool
	}{
		{"true", true},
		{"false", false},
		{"1", true},
		{"0", false},
		{"t", true},
		{"f", false},
	}

	for _, tt := range tests {
		os.Setenv("TEST_BOOL", tt.input)
		val, err := getEnvBool("TEST_BOOL", false)
		if err != nil {
			t.Errorf("unexpected error for input %s: %v", tt.input, err)
		}
		if val != tt.expected {
			t.Errorf("input %s: expected %v, got %v", tt.input, tt.expected, val)
		}
	}

	os.Setenv("TEST_BOOL", "not-a-bool")
	_, err = getEnvBool("TEST_BOOL", true)
	if err == nil {
		t.Error("expected error for invalid boolean")
	}
}

func TestGetDuration(t *testing.T) {
	os.Clearenv()
	defer os.Clearenv()

	val, err := getDuration("NONEXISTENT_KEY", 5*time.Minute)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if val != 5*time.Minute {
		t.Errorf("expected 5m, got %v", val)
	}

	os.Setenv("TEST_DUR", "2h30m")
	val, err = getDuration("TEST_DUR", 5*time.Minute)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if val != 2*time.Hour+30*time.Minute {
		t.Errorf("expected 2h30m, got %v", val)
	}

	os.Setenv("TEST_DUR", "invalid")
	_, err = getDuration("TEST_DUR", 5*time.Minute)
	if err == nil {
		t.Error("expected error for invalid duration")
	}
}

func TestValidateConfig_EmptyValues(t *testing.T) {
	cfg := Config{
		HTTPAddr:           "",
		PasswordBcryptCost: bcrypt.DefaultCost,
		DB: DBConfig{
			Host:            "localhost",
			Port:            5432,
			User:            "user",
			Name:            "db",
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: 10 * time.Minute,
		},
		Redis: RedisConfig{
			Addr:     "localhost:6379",
			PoolSize: 10,
		},
		JWT: JWTConfig{
			Secret:     "secret",
			Issuer:     "issuer",
			AccessTTL:  time.Hour,
			RefreshTTL: 24 * time.Hour,
		},
		Security: SecurityConfig{
			FlagHMACSecret:   "flag-secret",
			SubmissionWindow: time.Minute,
			SubmissionMax:    10,
		},
	}

	err := validateConfig(cfg)
	if err == nil {
		t.Error("expected error for empty HTTPAddr")
	}
}

func TestValidateConfig_InvalidDBConfig(t *testing.T) {
	cfg := Config{
		HTTPAddr:           ":8080",
		PasswordBcryptCost: bcrypt.DefaultCost,
		DB: DBConfig{
			Host:            "",
			Port:            0,
			User:            "",
			Name:            "",
			MaxOpenConns:    0,
			MaxIdleConns:    0,
			ConnMaxLifetime: 0,
		},
		Redis: RedisConfig{
			Addr:     "localhost:6379",
			PoolSize: 10,
		},
		JWT: JWTConfig{
			Secret:     "secret",
			Issuer:     "issuer",
			AccessTTL:  time.Hour,
			RefreshTTL: 24 * time.Hour,
		},
		Security: SecurityConfig{
			FlagHMACSecret:   "flag-secret",
			SubmissionWindow: time.Minute,
			SubmissionMax:    10,
		},
	}

	err := validateConfig(cfg)
	if err == nil {
		t.Error("expected error for invalid DB config")
	}
}
