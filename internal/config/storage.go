package config

type StorageConfig struct {
	Port           string
	Backend        string
	YDBDSN         string
	InternalToken  string
	AdminMasterKey string
}

func LoadStorageConfig() (StorageConfig, error) {
	cfg := StorageConfig{
		Port:           GetEnv("PORT", "8081"),
		Backend:        GetEnv("STORAGE_BACKEND", "ydb"),
		YDBDSN:         GetEnv("YDB_DSN", ""),
		InternalToken:  GetEnv("INTERNAL_AUTH_TOKEN", ""),
		AdminMasterKey: GetEnv("ADMIN_MASTER_KEY", ""),
	}

	if cfg.InternalToken == "" {
		return cfg, errMissing("INTERNAL_AUTH_TOKEN")
	}
	if cfg.Backend == "ydb" && cfg.YDBDSN == "" {
		return cfg, errMissing("YDB_DSN")
	}

	return cfg, nil
}
