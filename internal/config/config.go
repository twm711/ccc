package config

import (
	"os"
	"strconv"
)

type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	JWT       JWTConfig
	Snowflake SnowflakeConfig
	Aliyun    AliyunConfig
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

type SnowflakeConfig struct {
	NodeID int64
}

type AliyunConfig struct {
	AccessKeyID     string
	AccessKeySecret string
	NLSAppKey       string
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
		Aliyun: AliyunConfig{
			AccessKeyID:     envOrChain("ALIBABA_CLOUD_ACCESS_KEY_ID", "ALIBABA_ACCESS_KEY_ID", "ALIYUN_ACCESS_KEY_ID"),
			AccessKeySecret: envOrChain("ALIBABA_CLOUD_ACCESS_KEY_SECRET", "ALIBABA_ACCESS_KEY_SECRET", "ALIYUN_ACCESS_KEY_SECRET"),
			NLSAppKey:       envOrChain("NLS_APP_KEY", "NLS_PROJECT_APP_KEY", "ALIBABA_STT_APPKEY", "ALIYUN_NLS_APP_KEY"),
			DashScopeAPIKey: envOrChain("DASHSCOPE_API_KEY", "TONGYI_API_KEY"),
			DashScopeModel:  envOrChainDefault("qwen-plus", "DASHSCOPE_MODEL", "TONGYI_MODEL"),
		},
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
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

func envOrChain(keys ...string) string {
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return ""
}

func envOrChainDefault(fallback string, keys ...string) string {
	if v := envOrChain(keys...); v != "" {
		return v
	}
	return fallback
}
