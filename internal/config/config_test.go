package config

import (
	"os"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func TestLoadConfigDefaults(t *testing.T) {
	os.Clearenv()
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

	if cfg.BcryptCost != 12 {
		t.Errorf("expected BcryptCost 12, got %d", cfg.BcryptCost)
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

	if cfg.VM.CreateWindow != time.Minute {
		t.Errorf("expected VM.CreateWindow 1m, got %v", cfg.VM.CreateWindow)
	}

	if cfg.VM.CreateMax != 1 {
		t.Errorf("expected VM.CreateMax 1, got %d", cfg.VM.CreateMax)
	}
}

func TestLoadConfigCustomValues(t *testing.T) {
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
	os.Setenv("SUBMIT_WINDOW", "30s")
	os.Setenv("SUBMIT_MAX", "5")
	os.Setenv("LOG_DIR", "logs-test")
	os.Setenv("LOG_FILE_PREFIX", "app-test")
	os.Setenv("LOG_MAX_BODY_BYTES", "2048")
	os.Setenv("S3_ENABLED", "true")
	os.Setenv("S3_REGION", "ap-northeast-2")
	os.Setenv("S3_BUCKET", "smctf-test")
	os.Setenv("S3_ACCESS_KEY_ID", "access-key")
	os.Setenv("S3_SECRET_ACCESS_KEY", "secret-key")
	os.Setenv("S3_ENDPOINT", "https://s3.example.com")
	os.Setenv("S3_FORCE_PATH_STYLE", "true")
	os.Setenv("S3_PRESIGN_TTL", "20m")
	os.Setenv("VMS_ENABLED", "true")
	os.Setenv("VMS_MAX_SCOPE", "team")
	os.Setenv("VMS_MAX_PER", "5")
	os.Setenv("VMS_ORCHESTRATOR_BASE_URL", "http://localhost:18081")
	os.Setenv("VMS_ORCHESTRATOR_TIMEOUT", "9s")
	os.Setenv("VMS_CREATE_WINDOW", "2m")
	os.Setenv("VMS_CREATE_MAX", "2")

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

	if cfg.BcryptCost != 10 {
		t.Errorf("expected BcryptCost 10, got %d", cfg.BcryptCost)
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
	if cfg.VM.CreateWindow != 2*time.Minute {
		t.Errorf("expected VM.CreateWindow 2m, got %v", cfg.VM.CreateWindow)
	}
	if cfg.VM.CreateMax != 2 {
		t.Errorf("expected VM.CreateMax 2, got %d", cfg.VM.CreateMax)
	}
	if cfg.VM.MaxScope != "team" {
		t.Errorf("expected VM.MaxScope team, got %s", cfg.VM.MaxScope)
	}
	if cfg.VM.MaxPer != 5 {
		t.Errorf("expected VM.MaxPer 5, got %d", cfg.VM.MaxPer)
	}
}

func TestLoadConfigInvalidValues(t *testing.T) {
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
		{"invalid s3 enabled", "S3_ENABLED", "not-a-bool"},
		{"invalid s3 presign ttl", "S3_PRESIGN_TTL", "bad-duration"},
		{"invalid s3 force path", "S3_FORCE_PATH_STYLE", "bad-bool"},
		{"invalid vm max scope", "VMS_MAX_SCOPE", "org"},
		{"invalid leaderboard cache ttl", "LEADERBOARD_CACHE_TTL", "bad-duration"},
		{"invalid app config cache ttl", "APP_CONFIG_CACHE_TTL", "bad-duration"},
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

func TestLoadConfigS3ValidationErrors(t *testing.T) {
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

func TestValidateConfigInvalidS3(t *testing.T) {
	cfg := Config{
		HTTPAddr:   ":8080",
		BcryptCost: bcrypt.DefaultCost,
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
			SubmissionWindow: time.Minute,
			SubmissionMax:    10,
		},
		Logging: LoggingConfig{
			Dir:          "logs",
			FilePrefix:   "app",
			MaxBodyBytes: 1024,
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

func TestLoadConfigProductionValidation(t *testing.T) {
	os.Clearenv()
	os.Setenv("APP_ENV", "production")
	defer os.Clearenv()

	_, err := Load()
	if err == nil {
		t.Error("expected error for default secrets in production, got nil")
	}

	os.Setenv("JWT_SECRET", "production-secret-123")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed with valid production config: %v", err)
	}

	if cfg.AppEnv != "production" {
		t.Errorf("expected AppEnv production, got %s", cfg.AppEnv)
	}
}

func TestValidateConfigInvalidLogging(t *testing.T) {
	cfg := Config{
		HTTPAddr:   ":8080",
		BcryptCost: bcrypt.DefaultCost,
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
			SubmissionWindow: time.Minute,
			SubmissionMax:    10,
		},
		Logging: LoggingConfig{
			Dir:          "",
			FilePrefix:   "",
			MaxBodyBytes: 0,
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

func TestValidateConfigEmptyValues(t *testing.T) {
	cfg := Config{
		HTTPAddr:   "",
		BcryptCost: bcrypt.DefaultCost,
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
			SubmissionWindow: time.Minute,
			SubmissionMax:    10,
		},
		Logging: LoggingConfig{
			Dir:          "logs",
			FilePrefix:   "app",
			MaxBodyBytes: 1024,
		},
	}

	err := validateConfig(cfg)
	if err == nil {
		t.Error("expected error for empty HTTPAddr")
	}
}

func TestValidateConfigInvalidDBConfig(t *testing.T) {
	cfg := Config{
		HTTPAddr:   ":8080",
		BcryptCost: bcrypt.DefaultCost,
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
			SubmissionWindow: time.Minute,
			SubmissionMax:    10,
		},
		Logging: LoggingConfig{
			Dir:          "logs",
			FilePrefix:   "app",
			MaxBodyBytes: 1024,
		},
	}

	err := validateConfig(cfg)
	if err == nil {
		t.Error("expected error for invalid DB config")
	}
}

func TestValidateConfigInvalidVMConfig(t *testing.T) {
	cfg := Config{
		AppEnv:     "local",
		HTTPAddr:   ":8080",
		BcryptCost: bcrypt.DefaultCost,
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
			SubmissionWindow: time.Minute,
			SubmissionMax:    10,
		},
		Cache: CacheConfig{
			TimelineTTL:    time.Minute,
			LeaderboardTTL: time.Minute,
			AppConfigTTL:   2 * time.Minute,
		},
		CORS: CORSConfig{
			AllowedOrigins: []string{"http://localhost:3000", "https://smctf.example.com"},
		},
		Logging: LoggingConfig{
			Dir:          "logs",
			FilePrefix:   "app",
			MaxBodyBytes: 1024,
		},
		VM: VMConfig{
			Enabled:             true,
			MaxScope:            "user",
			MaxPer:              0,
			OrchestratorBaseURL: "",
			OrchestratorTimeout: 0,
			CreateWindow:        0,
			CreateMax:           0,
		},
	}

	err := validateConfig(cfg)
	if err == nil {
		t.Fatal("expected vm validation error")
	}

	if !strings.Contains(err.Error(), "VMS_MAX_PER") {
		t.Fatalf("expected vm error, got %v", err)
	}
}

func TestValidateConfigAdditionalValidation(t *testing.T) {
	cfg := Config{
		AppEnv:     "local",
		HTTPAddr:   ":8080",
		BcryptCost: bcrypt.DefaultCost,
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
			SubmissionWindow: 0,
			SubmissionMax:    0,
		},
		Cache: CacheConfig{
			TimelineTTL:    time.Minute,
			LeaderboardTTL: time.Minute,
			AppConfigTTL:   2 * time.Minute,
		},
		Logging: LoggingConfig{
			Dir:          "logs",
			FilePrefix:   "app",
			MaxBodyBytes: 1024,
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
		Logging: LoggingConfig{
			Dir:          "logs",
			FilePrefix:   "app",
			MaxBodyBytes: 1024,
		},
		S3: S3Config{
			AccessKeyID:     "access-key",
			SecretAccessKey: "secret-key",
		},
		VM: VMConfig{},
		Bootstrap: BootstrapConfig{
			AdminEmail:    "admin@example.com",
			AdminPassword: "adminpass",
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

	if redacted.S3.AccessKeyID == cfg.S3.AccessKeyID {
		t.Fatalf("expected s3 access key redacted")
	}

	if redacted.S3.SecretAccessKey == cfg.S3.SecretAccessKey {
		t.Fatalf("expected s3 secret key redacted")
	}

	if redacted.Bootstrap.AdminEmail == cfg.Bootstrap.AdminEmail {
		t.Fatalf("expected bootstrap admin email redacted")
	}

	if redacted.Bootstrap.AdminPassword == cfg.Bootstrap.AdminPassword {
		t.Fatalf("expected bootstrap admin password redacted")
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
		AppEnv:          "local",
		HTTPAddr:        ":8080",
		ShutdownTimeout: 5 * time.Second,
		AutoMigrate:     true,
		BcryptCost:      10,
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
			SubmissionWindow: time.Minute,
			SubmissionMax:    10,
		},
		Cache: CacheConfig{
			TimelineTTL:    time.Minute,
			LeaderboardTTL: time.Minute,
			AppConfigTTL:   2 * time.Minute,
		},
		Logging: LoggingConfig{
			Dir:          "logs",
			FilePrefix:   "app",
			MaxBodyBytes: 1024,
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
		VM: VMConfig{
			Enabled:             true,
			MaxScope:            "user",
			MaxPer:              3,
			OrchestratorBaseURL: "http://localhost:8081",
			OrchestratorTimeout: 5 * time.Second,
		},
		Bootstrap: BootstrapConfig{
			AdminTeamEnabled: true,
			AdminUserEnabled: true,
			AdminEmail:       "admin@example.com",
			AdminPassword:    "adminpass",
		},
	}

	out := FormatForLog(cfg)
	if len(out) == 0 {
		t.Fatalf("expected output")
	}

	db := out["db"].(map[string]any)
	redis := out["redis"].(map[string]any)
	jwt := out["jwt"].(map[string]any)
	if db["password"].(string) == "dbpass" || redis["password"].(string) == "redispass" || jwt["secret"].(string) == "jwtsecret" {
		t.Fatalf("expected secrets redacted")
	}

	bootstrap := out["bootstrap"].(map[string]any)
	if bootstrap["admin_password"].(string) == "adminpass" {
		t.Fatalf("expected bootstrap admin password redacted")
	}

	if out["app_env"] != "local" || out["http_addr"] != ":8080" {
		t.Fatalf("expected top-level config fields")
	}

	cache := out["cache"].(map[string]any)
	if cache["timeline_ttl"] == nil || cache["leaderboard_ttl"] == nil {
		t.Fatalf("expected cache fields")
	}
}

func discordEnabledEnv() {
	os.Setenv("DISCORD_ENABLED", "true")
	os.Setenv("DISCORD_CLIENT_ID", "cid")
	os.Setenv("DISCORD_CLIENT_SECRET", "csecret")
	os.Setenv("DISCORD_REDIRECT_URI", "http://localhost:8080/api/discord/callback")
	os.Setenv("DISCORD_OAUTH_SCOPES", "identify")
	os.Setenv("DISCORD_BOT_BASE_URL", "http://localhost:8083")
	os.Setenv("DISCORD_BOT_SECRET", "bot-secret")
}

func TestLoadConfigDiscordEnabled(t *testing.T) {
	os.Clearenv()
	discordEnabledEnv()
	defer os.Clearenv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if !cfg.Discord.Enabled {
		t.Fatal("expected Discord enabled")
	}
	if cfg.Discord.ClientID != "cid" || cfg.Discord.BotSecret != "bot-secret" {
		t.Errorf("unexpected discord config: %+v", cfg.Discord)
	}
	if !cfg.Discord.AutoJoin {
		t.Error("expected AutoJoin true by default")
	}
	if cfg.Discord.StateTTL != 5*time.Minute {
		t.Errorf("expected StateTTL 5m, got %v", cfg.Discord.StateTTL)
	}

	red := Redact(cfg)
	if red.Discord.ClientSecret == "csecret" || red.Discord.BotSecret == "bot-secret" {
		t.Error("expected discord secrets to be redacted")
	}

	if _, ok := FormatForLog(cfg)["discord"]; !ok {
		t.Error("expected discord section in log map")
	}
}

func TestLoadConfigDiscordValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		key  string
		val  string
	}{
		{"missing client id", "DISCORD_CLIENT_ID", ""},
		{"missing client secret", "DISCORD_CLIENT_SECRET", ""},
		{"missing redirect uri", "DISCORD_REDIRECT_URI", ""},
		{"missing bot secret", "DISCORD_BOT_SECRET", ""},
		{"invalid state ttl", "DISCORD_STATE_TTL", "0s"},
		{"invalid bot timeout", "DISCORD_BOT_TIMEOUT", "0s"},
		{"invalid oauth timeout", "DISCORD_OAUTH_TIMEOUT", "0s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			discordEnabledEnv()
			os.Setenv(tt.key, tt.val)
			defer os.Clearenv()

			if _, err := Load(); err == nil {
				t.Fatalf("expected error for %s", tt.name)
			}
		})
	}
}

func TestValidateConfigInvalidDiscord(t *testing.T) {
	cfg := Config{
		HTTPAddr:   ":8080",
		BcryptCost: bcrypt.DefaultCost,
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
			SubmissionWindow: time.Minute,
			SubmissionMax:    10,
		},
		Logging: LoggingConfig{
			Dir:          "logs",
			FilePrefix:   "app",
			MaxBodyBytes: 1024,
		},
		Discord: DiscordConfig{
			Enabled:      true,
			ClientID:     "cid",
			ClientSecret: "csecret",
			RedirectURI:  "http://localhost/cb",
			Scopes:       "",
			BotBaseURL:   "",
			BotSecret:    "bot-secret",
			StateTTL:     time.Minute,
			BotTimeout:   time.Second,
			OAuthTimeout: time.Second,
		},
	}

	if err := validateConfig(cfg); err == nil {
		t.Fatal("expected discord validation errors")
	}
}
