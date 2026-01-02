package config

import (
	"strings"
	"time"
)

type StorageConfig struct {
	Port             string
	Backend          string
	YDBDSN           string
	InternalToken    string
	AdminMasterKey   string
	TelegramBotToken string
	TelegramAdminID  int64
	AdminOTPTTL      time.Duration
	AdminSessionTTL  time.Duration
	LogGroupID       string
	LogResourceTypes []string
	LogResourceIDs   []string
	LogStreamNames   []string
	LogLevels        []string
	LogFilter        string
	LogEndpoint      string
	LogMaxBytes      int64
}

func LoadStorageConfig() (StorageConfig, error) {
	cfg := StorageConfig{
		Port:             GetEnv("PORT", "8081"),
		Backend:          GetEnv("STORAGE_BACKEND", "ydb"),
		YDBDSN:           GetEnv("YDB_DSN", ""),
		InternalToken:    GetEnv("INTERNAL_AUTH_TOKEN", ""),
		AdminMasterKey:   GetEnv("ADMIN_MASTER_KEY", ""),
		TelegramBotToken: GetEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramAdminID:  ParseInt64Env("TELEGRAM_ADMIN_ID", 0),
		AdminOTPTTL:      ParseDurationEnv("ADMIN_OTP_TTL", 5*time.Minute),
		AdminSessionTTL:  ParseDurationEnv("ADMIN_SESSION_TTL", 12*time.Hour),
		LogGroupID:       GetEnv("YC_LOG_GROUP_ID", ""),
		LogResourceTypes: splitCSV(GetEnv("YC_LOG_RESOURCE_TYPES", "")),
		LogResourceIDs:   splitCSV(GetEnv("YC_LOG_RESOURCE_IDS", "")),
		LogStreamNames:   splitCSV(GetEnv("YC_LOG_STREAMS", "")),
		LogLevels:        splitCSV(GetEnv("YC_LOG_LEVELS", "")),
		LogFilter:        GetEnv("YC_LOG_FILTER", ""),
		LogEndpoint:      GetEnv("YC_LOG_ENDPOINT", ""),
		LogMaxBytes:      ParseInt64Env("YC_LOG_MAX_BYTES", 0),
	}

	if cfg.InternalToken == "" {
		return cfg, errMissing("INTERNAL_AUTH_TOKEN")
	}
	if cfg.Backend == "ydb" && cfg.YDBDSN == "" {
		return cfg, errMissing("YDB_DSN")
	}

	return cfg, nil
}

func splitCSV(value string) []string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	parts := strings.Split(trimmed, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		cleaned := strings.TrimSpace(part)
		if cleaned == "" {
			continue
		}
		out = append(out, cleaned)
	}
	return out
}
