package storage

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/linkasu/dialog-helper-service/internal/auth"
	"github.com/linkasu/dialog-helper-service/internal/hash"
	"github.com/linkasu/dialog-helper-service/internal/models"
)

const (
	AdminMasterKeyConfigKey      = "admin_master_key_hash"
	SystemPromptConfigKey        = "system_prompt"
	SystemPromptVersionConfigKey = "system_prompt_version"
	defaultClientID              = "client"
	defaultKeyStatus             = auth.StatusActive
)

type Handler struct {
	store         Store
	internalToken string
	adminCache    adminKeyCache
}

type adminKeyCache struct {
	hash      string
	expiresAt time.Time
}

func NewHandler(store Store, internalToken string) *Handler {
	return &Handler{store: store, internalToken: internalToken}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/healthz", h.handleHealth)
	mux.HandleFunc("/internal/persist", h.withInternalAuth(h.handlePersist))
	mux.HandleFunc("/internal/cache/", h.withInternalAuth(h.handleCacheLookup))
	mux.HandleFunc("/internal/keys/verify", h.withInternalAuth(h.handleKeyVerify))
	mux.HandleFunc("/internal/config/", h.withInternalAuth(h.handleConfigGet))

	mux.HandleFunc("/v1/admin/keys", h.withAdminAuth(h.handleAdminKeys))
	mux.HandleFunc("/v1/admin/system-prompt", h.withAdminAuth(h.handleAdminPrompt))
}

func (h *Handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) handlePersist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	var payload struct {
		DialogHash string                   `json:"dialog_hash"`
		UserHash   string                   `json:"user_hash"`
		Transcript string                   `json:"transcript"`
		Candidates []models.AnswerCandidate `json:"candidates"`
		MetaJSON   string                   `json:"meta_json,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json")
		return
	}

	if payload.DialogHash == "" {
		writeError(w, http.StatusBadRequest, "missing_dialog_hash", "dialog_hash is required")
		return
	}

	record := ResultRecord{
		DialogHash: payload.DialogHash,
		UserHash:   payload.UserHash,
		Transcript: payload.Transcript,
		Candidates: payload.Candidates,
		MetaJSON:   payload.MetaJSON,
		CreatedAt:  time.Now().UTC(),
	}

	if err := h.store.PutResult(r.Context(), record); err != nil {
		log.Printf("persist failed: %v", err)
		writeError(w, http.StatusInternalServerError, "persist_failed", "persist failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) handleCacheLookup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	prefix := "/internal/cache/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	dialogHash := strings.TrimPrefix(r.URL.Path, prefix)
	if dialogHash == "" {
		writeError(w, http.StatusBadRequest, "missing_dialog_hash", "dialog_hash is required")
		return
	}

	record, ok, err := h.store.GetResult(r.Context(), dialogHash)
	if err != nil {
		log.Printf("cache lookup failed: %v", err)
		writeError(w, http.StatusInternalServerError, "cache_failed", "cache lookup failed")
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "cache_miss", "not found")
		return
	}

	writeJSON(w, http.StatusOK, record)
}

func (h *Handler) handleKeyVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	var payload struct {
		APIKey     string `json:"api_key"`
		APIKeyHash string `json:"api_key_hash"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json")
		return
	}

	apiKeyHash := payload.APIKeyHash
	if apiKeyHash == "" && payload.APIKey != "" {
		apiKeyHash = hash.HashAPIKey(payload.APIKey)
	}
	if apiKeyHash == "" {
		writeError(w, http.StatusBadRequest, "missing_api_key", "api_key or api_key_hash required")
		return
	}

	record, ok, err := h.store.GetAPIKey(r.Context(), apiKeyHash)
	if err != nil {
		log.Printf("api key lookup failed: %v", err)
		writeError(w, http.StatusInternalServerError, "key_lookup_failed", "api key lookup failed")
		return
	}
	if !ok || record.Status != auth.StatusActive {
		writeError(w, http.StatusUnauthorized, "invalid_key", "invalid api key")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"client_id":             record.ClientID,
		"status":                record.Status,
		"scopes":                record.Scopes,
		"rate_limit_per_minute": record.RateLimitPerMinute,
	})
}

func (h *Handler) handleConfigGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	prefix := "/internal/config/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	key := strings.TrimPrefix(r.URL.Path, prefix)
	if key == "" {
		writeError(w, http.StatusBadRequest, "missing_key", "key is required")
		return
	}

	item, ok, err := h.store.GetConfig(r.Context(), key)
	if err != nil {
		log.Printf("get config failed: %v", err)
		writeError(w, http.StatusInternalServerError, "config_failed", "config lookup failed")
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "config not found")
		return
	}

	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) handleAdminKeys(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	var req models.APIKeyCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json")
		return
	}

	clientID := strings.TrimSpace(req.ClientID)
	if clientID == "" {
		clientID = defaultClientID
	}

	plain, hashed, err := auth.GenerateAPIKey()
	if err != nil {
		log.Printf("generate api key failed: %v", err)
		writeError(w, http.StatusInternalServerError, "key_generation_failed", "api key generation failed")
		return
	}

	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = defaultKeyStatus
	}
	record := APIKeyRecord{
		APIKeyHash:         hashed,
		ClientID:           clientID,
		Status:             status,
		Scopes:             req.Scopes,
		RateLimitPerMinute: req.RateLimitPerMinute,
		CreatedAt:          time.Now().UTC(),
	}

	if err := h.store.CreateAPIKey(r.Context(), record); err != nil {
		log.Printf("create api key failed: %v", err)
		writeError(w, http.StatusInternalServerError, "key_create_failed", "api key create failed")
		return
	}

	resp := models.APIKeyCreateResponse{
		APIKey:             plain,
		APIKeyHash:         hashed,
		ClientID:           clientID,
		Status:             status,
		Scopes:             req.Scopes,
		RateLimitPerMinute: req.RateLimitPerMinute,
		CreatedAt:          record.CreatedAt,
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleAdminPrompt(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		item, _, err := h.store.GetConfig(r.Context(), SystemPromptConfigKey)
		if err != nil {
			log.Printf("get system prompt failed: %v", err)
			writeError(w, http.StatusInternalServerError, "config_failed", "config lookup failed")
			return
		}
		versionItem, _, err := h.store.GetConfig(r.Context(), SystemPromptVersionConfigKey)
		if err != nil {
			log.Printf("get system prompt version failed: %v", err)
			writeError(w, http.StatusInternalServerError, "config_failed", "config lookup failed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{
			"prompt":  item.Value,
			"version": versionItem.Value,
		})
	case http.MethodPost:
		var payload struct {
			Prompt  string `json:"prompt"`
			Version string `json:"version"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", "invalid json")
			return
		}
		if strings.TrimSpace(payload.Prompt) == "" {
			writeError(w, http.StatusBadRequest, "missing_prompt", "prompt is required")
			return
		}
		if strings.TrimSpace(payload.Version) == "" {
			payload.Version = time.Now().UTC().Format("20060102150405")
		}
		if err := h.store.SetConfig(r.Context(), SystemPromptConfigKey, payload.Prompt); err != nil {
			log.Printf("set system prompt failed: %v", err)
			writeError(w, http.StatusInternalServerError, "config_failed", "config update failed")
			return
		}
		if err := h.store.SetConfig(r.Context(), SystemPromptVersionConfigKey, payload.Version); err != nil {
			log.Printf("set system prompt version failed: %v", err)
			writeError(w, http.StatusInternalServerError, "config_failed", "config update failed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

func (h *Handler) withInternalAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Internal-Token")
		if token == "" || token != h.internalToken {
			writeError(w, http.StatusUnauthorized, "unauthorized", "invalid internal token")
			return
		}
		next(w, r)
	}
}

func (h *Handler) withAdminAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("X-ADMIN-KEY")
		if strings.TrimSpace(key) == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized", "missing admin key")
			return
		}
		hashValue, err := h.getAdminKeyHash(r.Context())
		if err != nil {
			log.Printf("admin key lookup failed: %v", err)
			writeError(w, http.StatusInternalServerError, "admin_key_lookup_failed", "admin key lookup failed")
			return
		}
		if hashValue == "" || !auth.ConstantTimeEquals(hash.HashAPIKey(key), hashValue) {
			writeError(w, http.StatusUnauthorized, "unauthorized", "invalid admin key")
			return
		}
		next(w, r)
	}
}

func (h *Handler) getAdminKeyHash(ctx context.Context) (string, error) {
	if time.Now().UTC().Before(h.adminCache.expiresAt) && h.adminCache.hash != "" {
		return h.adminCache.hash, nil
	}

	item, ok, err := h.store.GetConfig(ctx, AdminMasterKeyConfigKey)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", nil
	}

	h.adminCache = adminKeyCache{
		hash:      item.Value,
		expiresAt: time.Now().UTC().Add(1 * time.Minute),
	}
	return item.Value, nil
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload != nil {
		_ = json.NewEncoder(w).Encode(payload)
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{
		"error":   code,
		"message": message,
	})
}
