package config

import (
	"os"
	"strconv"
)

type Config struct {
	Server      ServerConfig
	Database    DatabaseConfig
	Redis       RedisConfig
	JWT         JWTConfig
	Snowflake   SnowflakeConfig
	Aliyun      AliyunConfig
	FreeSWITCH  FreeSWITCHConfig
	ServiceAuth ServiceAuthConfig
	Storage     StorageConfig
	NATS        NATSConfig
}

// NATSConfig points the lifecycle event publisher at a JetStream-enabled NATS
// cluster. Empty URL disables event publishing (best-effort, no-op).
type NATSConfig struct {
	URL    string
	Stream string
}

type FreeSWITCHConfig struct {
	Host     string
	Port     int
	Password string
	PoolSize int
}

type ServerConfig struct {
	Port string
}

type DatabaseConfig struct {
	DSN string
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type JWTConfig struct {
	Secret string
	Issuer string
}

// ServiceAuthConfig holds the HMAC secret used to authenticate internal
// service-to-service calls such as FreeSWITCH webhook callbacks into the
// API server. When empty, the /internal/v1 endpoints reject every request.
type ServiceAuthConfig struct {
	Secret string
}

// StorageConfig points the recording subsystem at an S3-compatible object
// store (typically MinIO in dev, AWS S3 in prod).
type StorageConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

type SnowflakeConfig struct {
	NodeID int64
}

type AliyunConfig struct {
	AccessKeyID     string
	AccessKeySecret string
	NLSAppKey       string
	NLSToken        string
	STTRegion       string
	TTSVoice        string
	TTSSampleRate   int
	DashScopeAPIKey string
	DashScopeModel  string
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port: envOr("SERVER_PORT", "8080"),
		},
		Database: DatabaseConfig{
			DSN: envOr("DATABASE_DSN", "root:root@tcp(127.0.0.1:3306)/ccc?parseTime=true&charset=utf8mb4&collation=utf8mb4_0900_ai_ci"),
		},
		Redis: RedisConfig{
			Addr:     envOr("REDIS_ADDR", "127.0.0.1:6379"),
			Password: envOr("REDIS_PASSWORD", ""),
			DB:       envOrInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret: envOr("JWT_SECRET", "change-me-in-production"),
			Issuer: envOr("JWT_ISSUER", "ccc-platform"),
		},
		Snowflake: SnowflakeConfig{
			NodeID: int64(envOrInt("SNOWFLAKE_NODE_ID", 1)),
		},
		FreeSWITCH: FreeSWITCHConfig{
			Host:     envOr("FREESWITCH_HOST", ""),
			Port:     envOrInt("FREESWITCH_PORT", 8021),
			Password: envOr("FREESWITCH_PASSWORD", "ClueCon"),
			PoolSize: envOrInt("FREESWITCH_POOL_SIZE", 5),
		},
		ServiceAuth: ServiceAuthConfig{
			Secret: envOr("SERVICE_AUTH_SECRET", ""),
		},
		Storage: StorageConfig{
			Endpoint:  envOr("STORAGE_ENDPOINT", ""),
			AccessKey: envOr("STORAGE_ACCESS_KEY", ""),
			SecretKey: envOr("STORAGE_SECRET_KEY", ""),
			Bucket:    envOr("STORAGE_BUCKET", "ccc-recordings"),
			UseSSL:    envOr("STORAGE_USE_SSL", "false") == "true",
		},
		NATS: NATSConfig{
			URL:    envOr("NATS_URL", ""),
			Stream: envOr("NATS_STREAM", "CCC_EVENTS"),
		},
		Aliyun: AliyunConfig{
			AccessKeyID:     firstEnv("ALIBABA_CLOUD_ACCESS_KEY_ID", "ALIBABA_ACCESS_KEY_ID", "ALIYUN_ACCESS_KEY_ID"),
			AccessKeySecret: firstEnv("ALIBABA_CLOUD_ACCESS_KEY_SECRET", "ALIBABA_ACCESS_KEY_SECRET", "ALIYUN_ACCESS_KEY_SECRET"),
			NLSAppKey:       firstEnv("NLS_APP_KEY", "NLS_PROJECT_APP_KEY", "ALIBABA_STT_APPKEY", "ALIYUN_NLS_APP_KEY"),
			NLSToken:        firstEnv("ALIBABA_STT_TOKEN", "ALIBABA_TTS_TOKEN"),
			STTRegion:       envOr("ALIBABA_STT_REGION", "cn-shanghai"),
			TTSVoice:        envOr("ALIBABA_TTS_VOICE", "zhixiaoxia"),
			TTSSampleRate:   envOrInt("ALIBABA_TTS_SAMPLE_RATE", 16000),
			DashScopeAPIKey: firstEnv("TONGYI_API_KEY", "DASHSCOPE_API_KEY"),
			DashScopeModel:  firstEnvOr("qwen-plus", "TONGYI_MODEL", "DASHSCOPE_MODEL"),
		},
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func firstEnv(keys ...string) string {
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return ""
}

func firstEnvOr(fallback string, keys ...string) string {
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return fallback
}

func envOrInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
