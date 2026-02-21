package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

type Config struct {
	AppEnv             string
	HTTPAddr           string
	ShutdownTimeout    time.Duration
	AutoMigrate        bool
	PasswordBcryptCost int

	DB        DBConfig
	Redis     RedisConfig
	JWT       JWTConfig
	Security  SecurityConfig
	Cache     CacheConfig
	CORS      CORSConfig
	Logging   LoggingConfig
	S3        S3Config
	Stack     StackConfig
	Bootstrap BootstrapConfig
}

type DBConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
	PoolSize int
}

type JWTConfig struct {
	Secret     string
	Issuer     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

type SecurityConfig struct {
	FlagHMACSecret   string
	SubmissionWindow time.Duration
	SubmissionMax    int
}

type CacheConfig struct {
	TimelineTTL    time.Duration
	LeaderboardTTL time.Duration
	AppConfigTTL   time.Duration
}

type CORSConfig struct {
	AllowedOrigins []string
}

type LoggingConfig struct {
	Dir          string
	FilePrefix   string
	MaxBodyBytes int
}

type S3Config struct {
	Enabled         bool
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	Endpoint        string
	ForcePathStyle  bool
	PresignTTL      time.Duration
}

type StackConfig struct {
	Enabled            bool
	MaxPerUser         int
	ProvisionerBaseURL string
	ProvisionerAPIKey  string
	ProvisionerTimeout time.Duration
	CreateWindow       time.Duration
	CreateMax          int
}

type BootstrapConfig struct {
	AdminTeamEnabled bool
	AdminUserEnabled bool
	AdminEmail       string
	AdminPassword    string
	AdminUsername    string
}

const (
	defaultJWTSecret  = "change-me"
	defaultFlagSecret = "change-me-too"
)

func Load() (Config, error) {
	var errs []error

	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		errs = append(errs, fmt.Errorf("load .env: %w", err))
	}

	appEnv := getEnv("APP_ENV", "local")
	httpAddr := getEnv("HTTP_ADDR", ":8080")
	shutdownTimeout, err := getDuration("SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		errs = append(errs, err)
	}

	autoMigrate, err := getEnvBool("AUTO_MIGRATE", true)
	if err != nil {
		errs = append(errs, err)
	}

	bcryptCost, err := getEnvInt("BCRYPT_COST", 12)
	if err != nil {
		errs = append(errs, err)
	}

	dbPort, err := getEnvInt("DB_PORT", 5432)
	if err != nil {
		errs = append(errs, err)
	}

	dbMaxOpen, err := getEnvInt("DB_MAX_OPEN_CONNS", 25)
	if err != nil {
		errs = append(errs, err)
	}

	dbMaxIdle, err := getEnvInt("DB_MAX_IDLE_CONNS", 10)
	if err != nil {
		errs = append(errs, err)
	}

	dbConnMaxLifetime, err := getDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute)
	if err != nil {
		errs = append(errs, err)
	}

	redisDB, err := getEnvInt("REDIS_DB", 0)
	if err != nil {
		errs = append(errs, err)
	}

	redisPoolSize, err := getEnvInt("REDIS_POOL_SIZE", 20)
	if err != nil {
		errs = append(errs, err)
	}

	jwtAccessTTL, err := getDuration("JWT_ACCESS_TTL", 24*time.Hour)
	if err != nil {
		errs = append(errs, err)
	}

	jwtRefreshTTL, err := getDuration("JWT_REFRESH_TTL", 7*24*time.Hour)
	if err != nil {
		errs = append(errs, err)
	}

	submitWindow, err := getDuration("SUBMIT_WINDOW", 1*time.Minute)
	if err != nil {
		errs = append(errs, err)
	}

	submitMax, err := getEnvInt("SUBMIT_MAX", 10)
	if err != nil {
		errs = append(errs, err)
	}

	timelineCacheTTL, err := getDuration("TIMELINE_CACHE_TTL", 60*time.Second)
	if err != nil {
		errs = append(errs, err)
	}

	leaderboardCacheTTL, err := getDuration("LEADERBOARD_CACHE_TTL", 60*time.Second)
	if err != nil {
		errs = append(errs, err)
	}

	appConfigCacheTTL, err := getDuration("APP_CONFIG_CACHE_TTL", 2*time.Minute)
	if err != nil {
		errs = append(errs, err)
	}

	corsAllowedOrigins := parseCSV(getEnv("CORS_ALLOWED_ORIGINS", ""))

	logDir := getEnv("LOG_DIR", "logs")
	logPrefix := getEnv("LOG_FILE_PREFIX", "app")
	logMaxBodyBytes, err := getEnvInt("LOG_MAX_BODY_BYTES", 1024*1024)
	if err != nil {
		errs = append(errs, err)
	}

	s3Enabled, err := getEnvBool("S3_ENABLED", false)
	if err != nil {
		errs = append(errs, err)
	}

	s3PresignTTL, err := getDuration("S3_PRESIGN_TTL", 15*time.Minute)
	if err != nil {
		errs = append(errs, err)
	}

	s3ForcePathStyle, err := getEnvBool("S3_FORCE_PATH_STYLE", false)
	if err != nil {
		errs = append(errs, err)
	}

	stackEnabled, err := getEnvBool("STACKS_ENABLED", true)
	if err != nil {
		errs = append(errs, err)
	}

	stackMaxPerUser, err := getEnvInt("STACKS_MAX_PER_USER", 3)
	if err != nil {
		errs = append(errs, err)
	}

	stackTimeout, err := getDuration("STACKS_PROVISIONER_TIMEOUT", 5*time.Second)
	if err != nil {
		errs = append(errs, err)
	}

	stackCreateWindow, err := getDuration("STACKS_CREATE_WINDOW", time.Minute)
	if err != nil {
		errs = append(errs, err)
	}

	stackCreateMax, err := getEnvInt("STACKS_CREATE_MAX", 1)
	if err != nil {
		errs = append(errs, err)
	}

	bootstrapAdminTeamEnabled, err := getEnvBool("BOOTSTRAP_ADMIN_TEAM", true)
	if err != nil {
		errs = append(errs, err)
	}

	bootstrapAdminUserEnabled, err := getEnvBool("BOOTSTRAP_ADMIN_USER", true)
	if err != nil {
		errs = append(errs, err)
	}

	cfg := Config{
		AppEnv:             appEnv,
		HTTPAddr:           httpAddr,
		ShutdownTimeout:    shutdownTimeout,
		AutoMigrate:        autoMigrate,
		PasswordBcryptCost: bcryptCost,
		DB: DBConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            dbPort,
			User:            getEnv("DB_USER", "app_user"),
			Password:        getEnv("DB_PASSWORD", "app_password"),
			Name:            getEnv("DB_NAME", "app_db"),
			SSLMode:         getEnv("DB_SSLMODE", "disable"),
			MaxOpenConns:    dbMaxOpen,
			MaxIdleConns:    dbMaxIdle,
			ConnMaxLifetime: dbConnMaxLifetime,
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       redisDB,
			PoolSize: redisPoolSize,
		},
		JWT: JWTConfig{
			Secret:     getEnv("JWT_SECRET", defaultJWTSecret),
			Issuer:     getEnv("JWT_ISSUER", "smctf"),
			AccessTTL:  jwtAccessTTL,
			RefreshTTL: jwtRefreshTTL,
		},
		Security: SecurityConfig{
			FlagHMACSecret:   getEnv("FLAG_HMAC_SECRET", defaultFlagSecret),
			SubmissionWindow: submitWindow,
			SubmissionMax:    submitMax,
		},
		Cache: CacheConfig{
			TimelineTTL:    timelineCacheTTL,
			LeaderboardTTL: leaderboardCacheTTL,
			AppConfigTTL:   appConfigCacheTTL,
		},
		CORS: CORSConfig{
			AllowedOrigins: corsAllowedOrigins,
		},
		Logging: LoggingConfig{
			Dir:          logDir,
			FilePrefix:   logPrefix,
			MaxBodyBytes: logMaxBodyBytes,
		},
		S3: S3Config{
			Enabled:         s3Enabled,
			Region:          getEnv("S3_REGION", "us-east-1"),
			Bucket:          getEnv("S3_BUCKET", ""),
			AccessKeyID:     getEnv("S3_ACCESS_KEY_ID", ""),
			SecretAccessKey: getEnv("S3_SECRET_ACCESS_KEY", ""),
			Endpoint:        getEnv("S3_ENDPOINT", ""),
			ForcePathStyle:  s3ForcePathStyle,
			PresignTTL:      s3PresignTTL,
		},
		Stack: StackConfig{
			Enabled:            stackEnabled,
			MaxPerUser:         stackMaxPerUser,
			ProvisionerBaseURL: getEnv("STACKS_PROVISIONER_BASE_URL", "http://localhost:8081"),
			ProvisionerAPIKey:  getEnv("STACKS_PROVISIONER_API_KEY", ""),
			ProvisionerTimeout: stackTimeout,
			CreateWindow:       stackCreateWindow,
			CreateMax:          stackCreateMax,
		},
		Bootstrap: BootstrapConfig{
			AdminTeamEnabled: bootstrapAdminTeamEnabled,
			AdminUserEnabled: bootstrapAdminUserEnabled,
			AdminEmail:       getEnv("BOOTSTRAP_ADMIN_EMAIL", ""),
			AdminPassword:    getEnv("BOOTSTRAP_ADMIN_PASSWORD", ""),
			AdminUsername:    getEnv("BOOTSTRAP_ADMIN_USERNAME", "admin"),
		},
	}

	if err := validateConfig(cfg); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return Config{}, errors.Join(errs...)
	}

	return cfg, nil
}

func getEnv(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}

	return v
}

func getEnvInt(key string, def int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}

	n, err := strconv.Atoi(v)
	if err != nil {
		return def, fmt.Errorf("%s must be an integer", key)
	}

	return n, nil
}

func getEnvBool(key string, def bool) (bool, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}

	b, err := strconv.ParseBool(v)
	if err != nil {
		return def, fmt.Errorf("%s must be a boolean", key)
	}

	return b, nil
}

func getDuration(key string, def time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}

	d, err := time.ParseDuration(v)
	if err != nil {
		return def, fmt.Errorf("%s must be a duration", key)
	}

	return d, nil
}

func validateConfig(cfg Config) error {
	var errs []error

	if cfg.HTTPAddr == "" {
		errs = append(errs, errors.New("HTTP_ADDR must not be empty"))
	}

	if cfg.PasswordBcryptCost < bcrypt.MinCost || cfg.PasswordBcryptCost > bcrypt.MaxCost {
		errs = append(errs, fmt.Errorf("BCRYPT_COST must be between %d and %d", bcrypt.MinCost, bcrypt.MaxCost))
	}

	if cfg.DB.Host == "" || cfg.DB.Name == "" || cfg.DB.User == "" {
		errs = append(errs, errors.New("DB_HOST, DB_NAME, and DB_USER must be set"))
	}
	if cfg.DB.Port <= 0 {
		errs = append(errs, errors.New("DB_PORT must be a positive integer"))
	}
	if cfg.DB.MaxOpenConns <= 0 || cfg.DB.MaxIdleConns <= 0 {
		errs = append(errs, errors.New("DB_MAX_OPEN_CONNS and DB_MAX_IDLE_CONNS must be positive"))
	}
	if cfg.DB.ConnMaxLifetime <= 0 {
		errs = append(errs, errors.New("DB_CONN_MAX_LIFETIME must be positive"))
	}

	// Redis validation
	if cfg.Redis.Addr == "" {
		errs = append(errs, errors.New("REDIS_ADDR must not be empty"))
	}
	if cfg.Redis.PoolSize <= 0 {
		errs = append(errs, errors.New("REDIS_POOL_SIZE must be positive"))
	}

	// JWT validation
	if cfg.JWT.Secret == "" {
		errs = append(errs, errors.New("JWT_SECRET must not be empty"))
	}
	if cfg.JWT.Issuer == "" {
		errs = append(errs, errors.New("JWT_ISSUER must not be empty"))
	}
	if cfg.JWT.AccessTTL <= 0 || cfg.JWT.RefreshTTL <= 0 {
		errs = append(errs, errors.New("JWT_ACCESS_TTL and JWT_REFRESH_TTL must be positive"))
	}

	// Security validation
	if cfg.Security.FlagHMACSecret == "" {
		errs = append(errs, errors.New("FLAG_HMAC_SECRET must not be empty"))
	}
	if cfg.Security.SubmissionWindow <= 0 || cfg.Security.SubmissionMax <= 0 {
		errs = append(errs, errors.New("SUBMIT_WINDOW and SUBMIT_MAX must be positive"))
	}

	// Production-specific validation
	if cfg.AppEnv == "production" {
		if cfg.JWT.Secret == defaultJWTSecret {
			errs = append(errs, errors.New("JWT_SECRET must be set in production"))
		}
		if cfg.Security.FlagHMACSecret == defaultFlagSecret {
			errs = append(errs, errors.New("FLAG_HMAC_SECRET must be set in production"))
		}
	}

	if cfg.Logging.Dir == "" {
		errs = append(errs, errors.New("LOG_DIR must not be empty"))
	}

	if cfg.Logging.FilePrefix == "" {
		errs = append(errs, errors.New("LOG_FILE_PREFIX must not be empty"))
	}

	if cfg.Logging.MaxBodyBytes <= 0 {
		errs = append(errs, errors.New("LOG_MAX_BODY_BYTES must be positive"))
	}

	if cfg.S3.Enabled {
		if cfg.S3.Region == "" {
			errs = append(errs, errors.New("S3_REGION must not be empty"))
		}
		if cfg.S3.Bucket == "" {
			errs = append(errs, errors.New("S3_BUCKET must not be empty"))
		}
		if (cfg.S3.AccessKeyID == "") != (cfg.S3.SecretAccessKey == "") {
			errs = append(errs, errors.New("S3_ACCESS_KEY_ID and S3_SECRET_ACCESS_KEY must both be set"))
		}
		if cfg.S3.PresignTTL <= 0 {
			errs = append(errs, errors.New("S3_PRESIGN_TTL must be positive"))
		}
	}

	if cfg.Stack.Enabled {
		if cfg.Stack.MaxPerUser <= 0 {
			errs = append(errs, errors.New("STACKS_MAX_PER_USER must be positive"))
		}
		if cfg.Stack.ProvisionerBaseURL == "" {
			errs = append(errs, errors.New("STACKS_PROVISIONER_BASE_URL must not be empty"))
		}
		if cfg.Stack.ProvisionerTimeout <= 0 {
			errs = append(errs, errors.New("STACKS_PROVISIONER_TIMEOUT must be positive"))
		}
		if cfg.Stack.ProvisionerAPIKey == "" {
			errs = append(errs, errors.New("STACKS_PROVISIONER_API_KEY must not be empty"))
		}
		if cfg.Stack.CreateWindow <= 0 {
			errs = append(errs, errors.New("STACKS_CREATE_WINDOW must be positive"))
		}
		if cfg.Stack.CreateMax <= 0 {
			errs = append(errs, errors.New("STACKS_CREATE_MAX must be positive"))
		}
	}

	if len(errs) == 0 {
		return nil
	}

	return errors.Join(errs...)
}

func Redact(cfg Config) Config {
	cfg.DB.Password = redact(cfg.DB.Password)
	cfg.Redis.Password = redact(cfg.Redis.Password)
	cfg.JWT.Secret = redact(cfg.JWT.Secret)
	cfg.Security.FlagHMACSecret = redact(cfg.Security.FlagHMACSecret)
	cfg.S3.AccessKeyID = redact(cfg.S3.AccessKeyID)
	cfg.S3.SecretAccessKey = redact(cfg.S3.SecretAccessKey)
	cfg.Stack.ProvisionerAPIKey = redact(cfg.Stack.ProvisionerAPIKey)
	cfg.Bootstrap.AdminEmail = redact(cfg.Bootstrap.AdminEmail)
	cfg.Bootstrap.AdminPassword = redact(cfg.Bootstrap.AdminPassword)

	return cfg
}

func redact(value string) string {
	if value == "" {
		return ""
	}

	const (
		visiblePrefix = 2
		visibleSuffix = 2
	)
	if len(value) <= visiblePrefix+visibleSuffix {
		return "***"
	}

	return value[:visiblePrefix] + "***" + value[len(value)-visibleSuffix:]
}

func parseCSV(value string) []string {
	if value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func FormatForLog(cfg Config) map[string]any {
	cfg = Redact(cfg)
	return map[string]any{
		"app_env":              cfg.AppEnv,
		"http_addr":            cfg.HTTPAddr,
		"shutdown_timeout":     seconds(cfg.ShutdownTimeout),
		"auto_migrate":         cfg.AutoMigrate,
		"password_bcrypt_cost": cfg.PasswordBcryptCost,
		"db": map[string]any{
			"host":              cfg.DB.Host,
			"port":              cfg.DB.Port,
			"user":              cfg.DB.User,
			"password":          cfg.DB.Password,
			"name":              cfg.DB.Name,
			"ssl_mode":          cfg.DB.SSLMode,
			"max_open_conns":    cfg.DB.MaxOpenConns,
			"max_idle_conns":    cfg.DB.MaxIdleConns,
			"conn_max_lifetime": seconds(cfg.DB.ConnMaxLifetime),
		},
		"redis": map[string]any{
			"addr":      cfg.Redis.Addr,
			"password":  cfg.Redis.Password,
			"db":        cfg.Redis.DB,
			"pool_size": cfg.Redis.PoolSize,
		},
		"jwt": map[string]any{
			"secret":      cfg.JWT.Secret,
			"issuer":      cfg.JWT.Issuer,
			"access_ttl":  seconds(cfg.JWT.AccessTTL),
			"refresh_ttl": seconds(cfg.JWT.RefreshTTL),
		},
		"security": map[string]any{
			"flag_hmac_secret":  cfg.Security.FlagHMACSecret,
			"submission_window": seconds(cfg.Security.SubmissionWindow),
			"submission_max":    cfg.Security.SubmissionMax,
		},
		"cache": map[string]any{
			"timeline_ttl":    seconds(cfg.Cache.TimelineTTL),
			"leaderboard_ttl": seconds(cfg.Cache.LeaderboardTTL),
			"app_config_ttl":  seconds(cfg.Cache.AppConfigTTL),
		},
		"cors": map[string]any{
			"allowed_origins": cfg.CORS.AllowedOrigins,
		},
		"logging": map[string]any{
			"dir":            cfg.Logging.Dir,
			"file_prefix":    cfg.Logging.FilePrefix,
			"max_body_bytes": cfg.Logging.MaxBodyBytes,
		},
		"s3": map[string]any{
			"enabled":           cfg.S3.Enabled,
			"region":            cfg.S3.Region,
			"bucket":            cfg.S3.Bucket,
			"access_key_id":     cfg.S3.AccessKeyID,
			"secret_access_key": cfg.S3.SecretAccessKey,
			"endpoint":          cfg.S3.Endpoint,
			"force_path_style":  cfg.S3.ForcePathStyle,
			"presign_ttl":       seconds(cfg.S3.PresignTTL),
		},
		"stack": map[string]any{
			"enabled":              cfg.Stack.Enabled,
			"max_per_user":         cfg.Stack.MaxPerUser,
			"provisioner_base_url": cfg.Stack.ProvisionerBaseURL,
			"provisioner_api_key":  cfg.Stack.ProvisionerAPIKey,
			"provisioner_timeout":  seconds(cfg.Stack.ProvisionerTimeout),
			"create_window":        seconds(cfg.Stack.CreateWindow),
			"create_max":           cfg.Stack.CreateMax,
		},
		"bootstrap": map[string]any{
			"admin_team_enabled": cfg.Bootstrap.AdminTeamEnabled,
			"admin_user_enabled": cfg.Bootstrap.AdminUserEnabled,
			"admin_username":     cfg.Bootstrap.AdminUsername,
			"admin_email":        cfg.Bootstrap.AdminEmail,
			"admin_password":     cfg.Bootstrap.AdminPassword,
		},
	}
}

func seconds(d time.Duration) int64 {
	return int64(d.Seconds())
}
