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
			AccessKeyID:     envOr("ALIYUN_ACCESS_KEY_ID", ""),
			AccessKeySecret: envOr("ALIYUN_ACCESS_KEY_SECRET", ""),
			NLSAppKey:       envOr("ALIYUN_NLS_APP_KEY", ""),
			DashScopeAPIKey: envOr("DASHSCOPE_API_KEY", ""),
			DashScopeModel:  envOr("DASHSCOPE_MODEL", "qwen-plus"),
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
