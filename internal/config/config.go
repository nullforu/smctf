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
	AppEnv          string
	HTTPAddr        string
	ShutdownTimeout time.Duration
	AutoMigrate     bool
	BcryptCost      int
	CookieDomain    string

	DB        DBConfig
	Redis     RedisConfig
	JWT       JWTConfig
	Security  SecurityConfig
	Cache     CacheConfig
	CORS      CORSConfig
	Logging   LoggingConfig
	S3        S3Config
	VM        VMConfig
	Discord   DiscordConfig
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

type VMConfig struct {
	Enabled             bool
	MaxScope            string
	MaxPer              int
	OrchestratorBaseURL string
	OrchestratorSecret  string
	OrchestratorTimeout time.Duration
	CreateWindow        time.Duration
	CreateMax           int
}

type DiscordConfig struct {
	Enabled         bool
	ClientID        string
	ClientSecret    string
	RedirectURI     string
	Scopes          string
	StateTTL        time.Duration
	SuccessRedirect string
	InviteURL       string
	AutoJoin        bool

	BotBaseURL   string
	BotSecret    string
	BotTimeout   time.Duration
	OAuthTimeout time.Duration
}

type BootstrapConfig struct {
	AdminTeamEnabled bool
	AdminUserEnabled bool
	AdminEmail       string
	AdminPassword    string
	AdminUsername    string
}

const defaultJWTSecret = "change-me"

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

	vmEnabled, err := getEnvBool("VMS_ENABLED", true)
	if err != nil {
		errs = append(errs, err)
	}

	vmMaxScope := strings.ToLower(strings.TrimSpace(getEnv("VMS_MAX_SCOPE", "team")))

	vmMaxPer, err := getEnvInt("VMS_MAX_PER", 3)
	if err != nil {
		errs = append(errs, err)
	}

	vmTimeout, err := getDuration("VMS_ORCHESTRATOR_TIMEOUT", 5*time.Second)
	if err != nil {
		errs = append(errs, err)
	}

	vmCreateWindow, err := getDuration("VMS_CREATE_WINDOW", time.Minute)
	if err != nil {
		errs = append(errs, err)
	}

	vmCreateMax, err := getEnvInt("VMS_CREATE_MAX", 1)
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

	discordEnabled, err := getEnvBool("DISCORD_ENABLED", false)
	if err != nil {
		errs = append(errs, err)
	}

	discordAutoJoin, err := getEnvBool("DISCORD_AUTO_JOIN", true)
	if err != nil {
		errs = append(errs, err)
	}

	discordStateTTL, err := getDuration("DISCORD_STATE_TTL", 5*time.Minute)
	if err != nil {
		errs = append(errs, err)
	}

	discordBotTimeout, err := getDuration("DISCORD_BOT_TIMEOUT", 5*time.Second)
	if err != nil {
		errs = append(errs, err)
	}

	discordOAuthTimeout, err := getDuration("DISCORD_OAUTH_TIMEOUT", 10*time.Second)
	if err != nil {
		errs = append(errs, err)
	}

	cfg := Config{
		AppEnv:          appEnv,
		HTTPAddr:        httpAddr,
		ShutdownTimeout: shutdownTimeout,
		AutoMigrate:     autoMigrate,
		BcryptCost:      bcryptCost,
		CookieDomain:    getEnv("COOKIE_DOMAIN", ""),
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
		VM: VMConfig{
			Enabled:             vmEnabled,
			MaxScope:            vmMaxScope,
			MaxPer:              vmMaxPer,
			OrchestratorBaseURL: getEnv("VMS_ORCHESTRATOR_BASE_URL", "http://localhost:8081"),
			OrchestratorSecret:  getEnv("VMS_ORCHESTRATOR_SECRET", ""),
			OrchestratorTimeout: vmTimeout,
			CreateWindow:        vmCreateWindow,
			CreateMax:           vmCreateMax,
		},
		Discord: DiscordConfig{
			Enabled:         discordEnabled,
			ClientID:        getEnv("DISCORD_CLIENT_ID", ""),
			ClientSecret:    getEnv("DISCORD_CLIENT_SECRET", ""),
			RedirectURI:     getEnv("DISCORD_REDIRECT_URI", ""),
			Scopes:          getEnv("DISCORD_OAUTH_SCOPES", "identify guilds.join"),
			StateTTL:        discordStateTTL,
			SuccessRedirect: getEnv("DISCORD_SUCCESS_REDIRECT", ""),
			InviteURL:       getEnv("DISCORD_INVITE_URL", ""),
			AutoJoin:        discordAutoJoin,
			BotBaseURL:      getEnv("DISCORD_BOT_BASE_URL", "http://localhost:8083"),
			BotSecret:       getEnv("DISCORD_BOT_SECRET", ""),
			BotTimeout:      discordBotTimeout,
			OAuthTimeout:    discordOAuthTimeout,
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

	if cfg.BcryptCost < bcrypt.MinCost || cfg.BcryptCost > bcrypt.MaxCost {
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
	if cfg.Security.SubmissionWindow <= 0 || cfg.Security.SubmissionMax <= 0 {
		errs = append(errs, errors.New("SUBMIT_WINDOW and SUBMIT_MAX must be positive"))
	}

	// Production-specific validation
	if cfg.AppEnv == "production" {
		if cfg.JWT.Secret == defaultJWTSecret {
			errs = append(errs, errors.New("JWT_SECRET must be set in production"))
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

	if cfg.VM.Enabled {
		if cfg.VM.MaxPer <= 0 {
			errs = append(errs, errors.New("VMS_MAX_PER must be positive"))
		}
		if cfg.VM.MaxScope != "user" && cfg.VM.MaxScope != "team" {
			errs = append(errs, errors.New("VMS_MAX_SCOPE must be user or team"))
		}
		if cfg.VM.OrchestratorBaseURL == "" {
			errs = append(errs, errors.New("VMS_ORCHESTRATOR_BASE_URL must not be empty"))
		}
		if cfg.VM.OrchestratorTimeout <= 0 {
			errs = append(errs, errors.New("VMS_ORCHESTRATOR_TIMEOUT must be positive"))
		}
		if cfg.VM.CreateWindow <= 0 {
			errs = append(errs, errors.New("VMS_CREATE_WINDOW must be positive"))
		}
		if cfg.VM.CreateMax <= 0 {
			errs = append(errs, errors.New("VMS_CREATE_MAX must be positive"))
		}
	}

	if cfg.Discord.Enabled {
		if cfg.Discord.ClientID == "" || cfg.Discord.ClientSecret == "" {
			errs = append(errs, errors.New("DISCORD_CLIENT_ID and DISCORD_CLIENT_SECRET must be set when DISCORD_ENABLED=true"))
		}
		if cfg.Discord.RedirectURI == "" {
			errs = append(errs, errors.New("DISCORD_REDIRECT_URI must not be empty when DISCORD_ENABLED=true"))
		}
		if cfg.Discord.Scopes == "" {
			errs = append(errs, errors.New("DISCORD_OAUTH_SCOPES must not be empty when DISCORD_ENABLED=true"))
		}
		if cfg.Discord.BotBaseURL == "" {
			errs = append(errs, errors.New("DISCORD_BOT_BASE_URL must not be empty when DISCORD_ENABLED=true"))
		}
		if cfg.Discord.BotSecret == "" {
			errs = append(errs, errors.New("DISCORD_BOT_SECRET must not be empty when DISCORD_ENABLED=true"))
		}
		if cfg.Discord.StateTTL <= 0 {
			errs = append(errs, errors.New("DISCORD_STATE_TTL must be positive"))
		}
		if cfg.Discord.BotTimeout <= 0 {
			errs = append(errs, errors.New("DISCORD_BOT_TIMEOUT must be positive"))
		}
		if cfg.Discord.OAuthTimeout <= 0 {
			errs = append(errs, errors.New("DISCORD_OAUTH_TIMEOUT must be positive"))
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
	cfg.S3.AccessKeyID = redact(cfg.S3.AccessKeyID)
	cfg.S3.SecretAccessKey = redact(cfg.S3.SecretAccessKey)
	cfg.Bootstrap.AdminEmail = redact(cfg.Bootstrap.AdminEmail)
	cfg.Bootstrap.AdminPassword = redact(cfg.Bootstrap.AdminPassword)
	cfg.VM.OrchestratorSecret = redact(cfg.VM.OrchestratorSecret)
	cfg.Discord.ClientSecret = redact(cfg.Discord.ClientSecret)
	cfg.Discord.BotSecret = redact(cfg.Discord.BotSecret)

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
		"app_env":          cfg.AppEnv,
		"http_addr":        cfg.HTTPAddr,
		"shutdown_timeout": seconds(cfg.ShutdownTimeout),
		"auto_migrate":     cfg.AutoMigrate,
		"bcrypt_cost":      cfg.BcryptCost,
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
		"vm": map[string]any{
			"enabled":               cfg.VM.Enabled,
			"max_scope":             cfg.VM.MaxScope,
			"max_per":               cfg.VM.MaxPer,
			"orchestrator_base_url": cfg.VM.OrchestratorBaseURL,
			"orchestrator_secret":   cfg.VM.OrchestratorSecret,
			"orchestrator_timeout":  seconds(cfg.VM.OrchestratorTimeout),
			"create_window":         seconds(cfg.VM.CreateWindow),
			"create_max":            cfg.VM.CreateMax,
		},
		"discord": map[string]any{
			"enabled":          cfg.Discord.Enabled,
			"client_id":        cfg.Discord.ClientID,
			"client_secret":    cfg.Discord.ClientSecret,
			"redirect_uri":     cfg.Discord.RedirectURI,
			"scopes":           cfg.Discord.Scopes,
			"state_ttl":        seconds(cfg.Discord.StateTTL),
			"success_redirect": cfg.Discord.SuccessRedirect,
			"invite_url":       cfg.Discord.InviteURL,
			"auto_join":        cfg.Discord.AutoJoin,
			"bot_base_url":     cfg.Discord.BotBaseURL,
			"bot_secret":       cfg.Discord.BotSecret,
			"bot_timeout":      seconds(cfg.Discord.BotTimeout),
			"oauth_timeout":    seconds(cfg.Discord.OAuthTimeout),
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
