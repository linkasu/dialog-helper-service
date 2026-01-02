package config

import (
	"strconv"
	"time"
)

type InferenceConfig struct {
	Port                 string
	StorageURL           string
	InternalToken        string
	FolderID             string
	ModelURI             string
	STTModel             string
	DefaultLanguage      string
	SystemPromptFallback string
	ConfigCacheTTL       time.Duration
	MaxAudioBytes        int64
	DefaultMaxCandidates int
	DefaultTemperature   float64
	DefaultMaxTokens     int
	PersistResults       bool
	STTProvider          string
	LLMProvider          string
	TimeWindowSec        int
}

func LoadInferenceConfig() (InferenceConfig, error) {
	cfg := InferenceConfig{
		Port:                 GetEnv("PORT", "8080"),
		StorageURL:           GetEnv("STORAGE_URL", ""),
		InternalToken:        GetEnv("INTERNAL_AUTH_TOKEN", ""),
		FolderID:             GetEnv("YC_FOLDER_ID", ""),
		ModelURI:             GetEnv("YC_GPT_MODEL_URI", ""),
		STTModel:             GetEnv("YC_STT_MODEL", "general"),
		DefaultLanguage:      GetEnv("DEFAULT_LANGUAGE", "ru-RU"),
		SystemPromptFallback: GetEnv("SYSTEM_PROMPT_FALLBACK", ""),
		ConfigCacheTTL:       ParseDurationEnv("CONFIG_CACHE_TTL", 1*time.Minute),
		MaxAudioBytes:        int64(ParseIntEnv("MAX_AUDIO_BYTES", 8*1024*1024)),
		DefaultMaxCandidates: ParseIntEnv("DEFAULT_MAX_CANDIDATES", 5),
		DefaultTemperature:   parseFloatEnv("DEFAULT_TEMPERATURE", 0.3),
		DefaultMaxTokens:     ParseIntEnv("DEFAULT_MAX_TOKENS", 300),
		PersistResults:       parseBoolEnv("PERSIST_RESULTS", true),
		STTProvider:          GetEnv("STT_PROVIDER", "yandex"),
		LLMProvider:          GetEnv("LLM_PROVIDER", "yandex"),
		TimeWindowSec:        ParseIntEnv("DEFAULT_TIME_WINDOW_SEC", 600),
	}

	if cfg.StorageURL == "" {
		return cfg, errMissing("STORAGE_URL")
	}
	if cfg.InternalToken == "" {
		return cfg, errMissing("INTERNAL_AUTH_TOKEN")
	}
	if cfg.FolderID == "" {
		return cfg, errMissing("YC_FOLDER_ID")
	}

	return cfg, nil
}

func parseBoolEnv(key string, def bool) bool {
	value := GetEnv(key, "")
	if value == "" {
		return def
	}
	switch value {
	case "true", "1", "yes":
		return true
	case "false", "0", "no":
		return false
	default:
		return def
	}
}

func parseFloatEnv(key string, def float64) float64 {
	value := GetEnv(key, "")
	if value == "" {
		return def
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return def
	}
	return parsed
}
