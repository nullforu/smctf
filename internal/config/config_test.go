package config

import (
	"os"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func TestLoadConfig_Defaults(t *testing.T) {
	os.Clearenv()
	os.Setenv("STACKS_PROVISIONER_API_KEY", "test-key")
	defer os.Clearenv()

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

	if cfg.S3.Enabled {
		t.Errorf("expected S3.Enabled false by default")
	}

	if cfg.S3.Region != "us-east-1" {
		t.Errorf("expected S3.Region us-east-1, got %s", cfg.S3.Region)
	}

	if cfg.Stack.CreateWindow != time.Minute {
		t.Errorf("expected Stack.CreateWindow 1m, got %v", cfg.Stack.CreateWindow)
	}

	if cfg.Stack.CreateMax != 1 {
		t.Errorf("expected Stack.CreateMax 1, got %d", cfg.Stack.CreateMax)
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
	os.Setenv("LOG_DIR", "logs-test")
	os.Setenv("LOG_FILE_PREFIX", "app-test")
	os.Setenv("LOG_DISCORD_WEBHOOK_URL", "https://discord.example/hook")
	os.Setenv("LOG_SLACK_WEBHOOK_URL", "https://slack.example/hook")
	os.Setenv("LOG_MAX_BODY_BYTES", "2048")
	os.Setenv("LOG_WEBHOOK_QUEUE_SIZE", "250")
	os.Setenv("LOG_WEBHOOK_TIMEOUT", "3s")
	os.Setenv("LOG_WEBHOOK_BATCH_SIZE", "5")
	os.Setenv("LOG_WEBHOOK_BATCH_WAIT", "1s")
	os.Setenv("LOG_WEBHOOK_MAX_CHARS", "1900")
	os.Setenv("S3_ENABLED", "true")
	os.Setenv("S3_REGION", "ap-northeast-2")
	os.Setenv("S3_BUCKET", "smctf-test")
	os.Setenv("S3_ACCESS_KEY_ID", "access-key")
	os.Setenv("S3_SECRET_ACCESS_KEY", "secret-key")
	os.Setenv("S3_ENDPOINT", "https://s3.example.com")
	os.Setenv("S3_FORCE_PATH_STYLE", "true")
	os.Setenv("S3_PRESIGN_TTL", "20m")
	os.Setenv("STACKS_ENABLED", "true")
	os.Setenv("STACKS_MAX_PER_USER", "5")
	os.Setenv("STACKS_PROVISIONER_BASE_URL", "http://localhost:18081")
	os.Setenv("STACKS_PROVISIONER_API_KEY", "custom-key")
	os.Setenv("STACKS_PROVISIONER_TIMEOUT", "9s")
	os.Setenv("STACKS_CREATE_WINDOW", "2m")
	os.Setenv("STACKS_CREATE_MAX", "2")

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
	if cfg.Logging.Dir != "logs-test" {
		t.Errorf("expected Logging.Dir logs-test, got %s", cfg.Logging.Dir)
	}
	if cfg.Logging.FilePrefix != "app-test" {
		t.Errorf("expected Logging.FilePrefix app-test, got %s", cfg.Logging.FilePrefix)
	}
	if cfg.Logging.DiscordWebhookURL != "https://discord.example/hook" {
		t.Errorf("expected Logging.DiscordWebhookURL, got %s", cfg.Logging.DiscordWebhookURL)
	}
	if cfg.Logging.SlackWebhookURL != "https://slack.example/hook" {
		t.Errorf("expected Logging.SlackWebhookURL, got %s", cfg.Logging.SlackWebhookURL)
	}

	if !cfg.S3.Enabled {
		t.Errorf("expected S3.Enabled true")
	}
	if cfg.S3.Region != "ap-northeast-2" {
		t.Errorf("expected S3.Region ap-northeast-2, got %s", cfg.S3.Region)
	}
	if cfg.S3.Bucket != "smctf-test" {
		t.Errorf("expected S3.Bucket smctf-test, got %s", cfg.S3.Bucket)
	}
	if cfg.S3.AccessKeyID != "access-key" {
		t.Errorf("expected S3.AccessKeyID access-key, got %s", cfg.S3.AccessKeyID)
	}
	if cfg.S3.SecretAccessKey != "secret-key" {
		t.Errorf("expected S3.SecretAccessKey secret-key, got %s", cfg.S3.SecretAccessKey)
	}
	if cfg.S3.Endpoint != "https://s3.example.com" {
		t.Errorf("expected S3.Endpoint https://s3.example.com, got %s", cfg.S3.Endpoint)
	}
	if !cfg.S3.ForcePathStyle {
		t.Errorf("expected S3.ForcePathStyle true")
	}
	if cfg.S3.PresignTTL != 20*time.Minute {
		t.Errorf("expected S3.PresignTTL 20m, got %v", cfg.S3.PresignTTL)
	}
	if cfg.Logging.MaxBodyBytes != 2048 {
		t.Errorf("expected Logging.MaxBodyBytes 2048, got %d", cfg.Logging.MaxBodyBytes)
	}
	if cfg.Logging.WebhookQueueSize != 250 {
		t.Errorf("expected Logging.WebhookQueueSize 250, got %d", cfg.Logging.WebhookQueueSize)
	}
	if cfg.Logging.WebhookTimeout != 3*time.Second {
		t.Errorf("expected Logging.WebhookTimeout 3s, got %s", cfg.Logging.WebhookTimeout)
	}
	if cfg.Logging.WebhookBatchSize != 5 {
		t.Errorf("expected Logging.WebhookBatchSize 5, got %d", cfg.Logging.WebhookBatchSize)
	}
	if cfg.Logging.WebhookBatchWait != time.Second {
		t.Errorf("expected Logging.WebhookBatchWait 1s, got %s", cfg.Logging.WebhookBatchWait)
	}
	if cfg.Logging.WebhookMaxChars != 1900 {
		t.Errorf("expected Logging.WebhookMaxChars 1900, got %d", cfg.Logging.WebhookMaxChars)
	}
	if cfg.Stack.CreateWindow != 2*time.Minute {
		t.Errorf("expected Stack.CreateWindow 2m, got %v", cfg.Stack.CreateWindow)
	}
	if cfg.Stack.CreateMax != 2 {
		t.Errorf("expected Stack.CreateMax 2, got %d", cfg.Stack.CreateMax)
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
		{"invalid log max body", "LOG_MAX_BODY_BYTES", "nope"},
		{"invalid log queue size", "LOG_WEBHOOK_QUEUE_SIZE", "bad"},
		{"invalid log timeout", "LOG_WEBHOOK_TIMEOUT", "bad"},
		{"invalid log batch size", "LOG_WEBHOOK_BATCH_SIZE", "bad"},
		{"invalid log batch wait", "LOG_WEBHOOK_BATCH_WAIT", "bad"},
		{"invalid log max chars", "LOG_WEBHOOK_MAX_CHARS", "bad"},
		{"invalid s3 enabled", "S3_ENABLED", "not-a-bool"},
		{"invalid s3 presign ttl", "S3_PRESIGN_TTL", "bad-duration"},
		{"invalid s3 force path", "S3_FORCE_PATH_STYLE", "bad-bool"},
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

func TestLoadConfig_S3ValidationErrors(t *testing.T) {
	tests := []struct {
		name  string
		setup func()
	}{
		{
			name: "missing bucket",
			setup: func() {
				os.Setenv("S3_ENABLED", "true")
				os.Setenv("S3_REGION", "us-east-1")
				os.Setenv("S3_BUCKET", "")
			},
		},
		{
			name: "partial credentials",
			setup: func() {
				os.Setenv("S3_ENABLED", "true")
				os.Setenv("S3_REGION", "us-east-1")
				os.Setenv("S3_BUCKET", "bucket")
				os.Setenv("S3_ACCESS_KEY_ID", "access")
				os.Setenv("S3_SECRET_ACCESS_KEY", "")
			},
		},
		{
			name: "invalid presign ttl",
			setup: func() {
				os.Setenv("S3_ENABLED", "true")
				os.Setenv("S3_REGION", "us-east-1")
				os.Setenv("S3_BUCKET", "bucket")
				os.Setenv("S3_PRESIGN_TTL", "0s")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			tt.setup()
			defer os.Clearenv()

			_, err := Load()
			if err == nil {
				t.Fatalf("expected error")
			}
		})
	}
}

func TestValidateConfig_InvalidS3(t *testing.T) {
	cfg := Config{
		HTTPAddr:           ":8080",
		PasswordBcryptCost: bcrypt.DefaultCost,
		DB: DBConfig{
			Host:            "localhost",
			Port:            5432,
			User:            "user",
			Name:            "db",
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: time.Minute,
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
		Logging: LoggingConfig{
			Dir:              "logs",
			FilePrefix:       "app",
			MaxBodyBytes:     1024,
			WebhookQueueSize: 10,
			WebhookTimeout:   time.Second,
			WebhookBatchSize: 5,
			WebhookBatchWait: time.Second,
			WebhookMaxChars:  100,
		},
		S3: S3Config{
			Enabled:         true,
			Region:          "",
			Bucket:          "",
			AccessKeyID:     "access",
			SecretAccessKey: "",
			PresignTTL:      0,
		},
	}

	if err := validateConfig(cfg); err == nil {
		t.Fatalf("expected s3 validation error")
	}
}

func TestLoadConfig_ProductionValidation(t *testing.T) {
	os.Clearenv()
	os.Setenv("APP_ENV", "production")
	os.Setenv("STACKS_PROVISIONER_API_KEY", "test-key")
	defer os.Clearenv()

	_, err := Load()
	if err == nil {
		t.Error("expected error for default secrets in production, got nil")
	}

	os.Setenv("JWT_SECRET", "production-secret-123")
	os.Setenv("FLAG_HMAC_SECRET", "production-flag-secret-456")
	os.Setenv("STACKS_PROVISIONER_API_KEY", "test-key")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed with valid production config: %v", err)
	}

	if cfg.AppEnv != "production" {
		t.Errorf("expected AppEnv production, got %s", cfg.AppEnv)
	}
}

func TestValidateConfig_InvalidLogging(t *testing.T) {
	cfg := Config{
		HTTPAddr:           ":8080",
		PasswordBcryptCost: bcrypt.DefaultCost,
		DB: DBConfig{
			Host:            "localhost",
			Port:            5432,
			User:            "user",
			Name:            "db",
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: time.Minute,
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
		Logging: LoggingConfig{
			Dir:              "",
			FilePrefix:       "",
			MaxBodyBytes:     0,
			WebhookQueueSize: 0,
			WebhookTimeout:   0,
			WebhookBatchSize: 0,
			WebhookBatchWait: 0,
			WebhookMaxChars:  0,
		},
	}

	if err := validateConfig(cfg); err == nil {
		t.Fatal("expected logging validation errors")
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
		Logging: LoggingConfig{
			Dir:              "logs",
			FilePrefix:       "app",
			MaxBodyBytes:     1024,
			WebhookQueueSize: 10,
			WebhookTimeout:   time.Second,
			WebhookBatchSize: 5,
			WebhookBatchWait: time.Second,
			WebhookMaxChars:  100,
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
		Logging: LoggingConfig{
			Dir:              "logs",
			FilePrefix:       "app",
			MaxBodyBytes:     1024,
			WebhookQueueSize: 10,
			WebhookTimeout:   time.Second,
			WebhookBatchSize: 5,
			WebhookBatchWait: time.Second,
			WebhookMaxChars:  100,
		},
	}

	err := validateConfig(cfg)
	if err == nil {
		t.Error("expected error for invalid DB config")
	}
}

func TestValidateConfig_AdditionalValidation(t *testing.T) {
	cfg := Config{
		AppEnv:             "local",
		HTTPAddr:           ":8080",
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
			Addr:     "",
			PoolSize: 0,
		},
		JWT: JWTConfig{
			Secret:     "",
			Issuer:     "",
			AccessTTL:  0,
			RefreshTTL: 0,
		},
		Security: SecurityConfig{
			FlagHMACSecret:   "",
			SubmissionWindow: 0,
			SubmissionMax:    0,
		},
		Cache: CacheConfig{
			TimelineTTL:    time.Minute,
			LeaderboardTTL: time.Minute,
		},
		Logging: LoggingConfig{
			Dir:              "logs",
			FilePrefix:       "app",
			MaxBodyBytes:     1024,
			WebhookQueueSize: 10,
			WebhookTimeout:   time.Second,
			WebhookBatchSize: 5,
			WebhookBatchWait: time.Second,
		},
	}

	if err := validateConfig(cfg); err == nil {
		t.Error("expected error for redis/jwt/security validation")
	}
}

func TestRedact(t *testing.T) {
	cfg := Config{
		DB: DBConfig{Password: "dbpass"},
		Redis: RedisConfig{
			Password: "redispass",
		},
		JWT: JWTConfig{
			Secret: "jwtsecret",
		},
		Security: SecurityConfig{
			FlagHMACSecret: "flagsecret",
		},
		Logging: LoggingConfig{
			DiscordWebhookURL: "https://discord.example/hook",
			SlackWebhookURL:   "https://slack.example/hook",
		},
		S3: S3Config{
			AccessKeyID:     "access-key",
			SecretAccessKey: "secret-key",
		},
		Stack: StackConfig{
			ProvisionerAPIKey: "stack-key",
		},
	}

	redacted := Redact(cfg)
	if redacted.DB.Password == cfg.DB.Password {
		t.Fatalf("expected db password redacted")
	}

	if redacted.Redis.Password == cfg.Redis.Password {
		t.Fatalf("expected redis password redacted")
	}

	if redacted.JWT.Secret == cfg.JWT.Secret {
		t.Fatalf("expected jwt secret redacted")
	}

	if redacted.Security.FlagHMACSecret == cfg.Security.FlagHMACSecret {
		t.Fatalf("expected flag secret redacted")
	}

	if redacted.Logging.DiscordWebhookURL == cfg.Logging.DiscordWebhookURL {
		t.Fatalf("expected discord webhook redacted")
	}

	if redacted.Logging.SlackWebhookURL == cfg.Logging.SlackWebhookURL {
		t.Fatalf("expected slack webhook redacted")
	}

	if redacted.S3.AccessKeyID == cfg.S3.AccessKeyID {
		t.Fatalf("expected s3 access key redacted")
	}

	if redacted.S3.SecretAccessKey == cfg.S3.SecretAccessKey {
		t.Fatalf("expected s3 secret key redacted")
	}

	if redacted.Stack.ProvisionerAPIKey == cfg.Stack.ProvisionerAPIKey {
		t.Fatalf("expected stack api key redacted")
	}
}

func TestRedactValueEdgeCases(t *testing.T) {
	if got := redact(""); got != "" {
		t.Fatalf("expected empty value, got %s", got)
	}

	if got := redact("ab"); got != "***" {
		t.Fatalf("expected short value to be masked, got %s", got)
	}

	if got := redact("abcd"); got != "***" {
		t.Fatalf("expected short value to be masked, got %s", got)
	}

	if got := redact("abcdef"); got != "ab***ef" {
		t.Fatalf("unexpected redaction: %s", got)
	}
}

func TestFormatForLog(t *testing.T) {
	cfg := Config{
		AppEnv:             "local",
		HTTPAddr:           ":8080",
		ShutdownTimeout:    5 * time.Second,
		AutoMigrate:        true,
		PasswordBcryptCost: 10,
		DB: DBConfig{
			Host:            "localhost",
			Port:            5432,
			User:            "user",
			Password:        "dbpass",
			Name:            "db",
			SSLMode:         "disable",
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: time.Minute,
		},
		Redis: RedisConfig{
			Addr:     "localhost:6379",
			Password: "redispass",
			DB:       0,
			PoolSize: 5,
		},
		JWT: JWTConfig{
			Secret:     "jwtsecret",
			Issuer:     "issuer",
			AccessTTL:  time.Hour,
			RefreshTTL: 2 * time.Hour,
		},
		Security: SecurityConfig{
			FlagHMACSecret:   "flagsecret",
			SubmissionWindow: time.Minute,
			SubmissionMax:    10,
		},
		Cache: CacheConfig{
			TimelineTTL:    time.Minute,
			LeaderboardTTL: time.Minute,
		},
		Logging: LoggingConfig{
			Dir:               "logs",
			FilePrefix:        "app",
			DiscordWebhookURL: "https://discord.example/hook",
			SlackWebhookURL:   "https://slack.example/hook",
			MaxBodyBytes:      1024,
			WebhookQueueSize:  10,
			WebhookTimeout:    time.Second,
			WebhookBatchSize:  5,
			WebhookBatchWait:  time.Second,
			WebhookMaxChars:   100,
		},
		S3: S3Config{
			Enabled:         true,
			Region:          "us-east-1",
			Bucket:          "smctf",
			AccessKeyID:     "access-key",
			SecretAccessKey: "secret-key",
			Endpoint:        "https://s3.example.com",
			ForcePathStyle:  true,
			PresignTTL:      10 * time.Minute,
		},
		Stack: StackConfig{
			Enabled:            true,
			MaxPerUser:         3,
			ProvisionerBaseURL: "http://localhost:8081",
			ProvisionerAPIKey:  "stack-key",
			ProvisionerTimeout: 5 * time.Second,
		},
	}

	out := FormatForLog(cfg)
	if out == "" {
		t.Fatalf("expected output")
	}

	if strings.Contains(out, "dbpass") || strings.Contains(out, "redispass") || strings.Contains(out, "jwtsecret") || strings.Contains(out, "flagsecret") || strings.Contains(out, "stack-key") {
		t.Fatalf("expected secrets redacted")
	}

	if !strings.Contains(out, "AppEnv=local") || !strings.Contains(out, "DB:") || !strings.Contains(out, "Cache:") {
		t.Fatalf("expected formatted config fields")
	}
}
