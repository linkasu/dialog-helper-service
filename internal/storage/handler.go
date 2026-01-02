package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/linkasu/dialog-helper-service/internal/auth"
	"github.com/linkasu/dialog-helper-service/internal/cloudlogging"
	"github.com/linkasu/dialog-helper-service/internal/hash"
	"github.com/linkasu/dialog-helper-service/internal/httpx"
	"github.com/linkasu/dialog-helper-service/internal/models"
)

const (
	AdminMasterKeyConfigKey      = "admin_master_key_hash"
	AdminOTPConfigKey            = "admin_otp"
	AdminSessionConfigKey        = "admin_session"
	SystemPromptConfigKey        = "system_prompt"
	SystemPromptVersionConfigKey = "system_prompt_version"
	defaultClientID              = "client"
	defaultKeyStatus             = auth.StatusActive
	defaultListLimit             = 100
	maxListLimit                 = 500
	maxAdminBodyBytes            = 1 << 20
	maxClientIDRunes             = 64
	maxPromptRunes               = 20000
	maxScopeRunes                = 32
	maxScopeCount                = 8
	minOTPSendInterval           = 30 * time.Second
	otpDigits                    = 6
)

type Handler struct {
	store           Store
	internalToken   string
	adminCache      adminKeyCache
	otpTTL          time.Duration
	sessionTTL      time.Duration
	telegramToken   string
	telegramAdminID int64
	httpClient      *http.Client
	logReader       cloudlogging.Reader
}

type HandlerConfig struct {
	InternalToken    string
	TelegramBotToken string
	TelegramAdminID  int64
	AdminOTPTTL      time.Duration
	AdminSessionTTL  time.Duration
	LogReader        cloudlogging.Reader
}

type adminKeyCache struct {
	hash      string
	expiresAt time.Time
}

func NewHandler(store Store, cfg HandlerConfig) *Handler {
	otpTTL := cfg.AdminOTPTTL
	if otpTTL <= 0 {
		otpTTL = 5 * time.Minute
	}
	sessionTTL := cfg.AdminSessionTTL
	if sessionTTL <= 0 {
		sessionTTL = 12 * time.Hour
	}

	return &Handler{
		store:           store,
		internalToken:   cfg.InternalToken,
		otpTTL:          otpTTL,
		sessionTTL:      sessionTTL,
		telegramToken:   strings.TrimSpace(cfg.TelegramBotToken),
		telegramAdminID: cfg.TelegramAdminID,
		httpClient:      &http.Client{Timeout: 5 * time.Second},
		logReader:       cfg.LogReader,
	}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/healthz", h.handleHealth)
	mux.HandleFunc("/internal/persist", h.withInternalAuth(h.handlePersist))
	mux.HandleFunc("/internal/cache/", h.withInternalAuth(h.handleCacheLookup))
	mux.HandleFunc("/internal/keys/verify", h.withInternalAuth(h.handleKeyVerify))
	mux.HandleFunc("/internal/config/", h.withInternalAuth(h.handleConfigGet))

	mux.HandleFunc("/v1/admin/otp/send", h.handleAdminOTPSend)
	mux.HandleFunc("/v1/admin/otp/verify", h.handleAdminOTPVerify)
	mux.HandleFunc("/v1/admin/keys", h.withAdminAuth(h.handleAdminKeys))
	mux.HandleFunc("/v1/admin/keys/", h.withAdminAuth(h.handleAdminKeyByID))
	mux.HandleFunc("/v1/admin/logs", h.withAdminAuth(h.handleAdminLogs))
	mux.HandleFunc("/v1/admin/stats", h.withAdminAuth(h.handleAdminStats))
	mux.HandleFunc("/v1/admin/system-prompt", h.withAdminAuth(h.handleAdminPrompt))
}

func (h *Handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, httpx.StatusResponse{Status: "ok"})
}

func (h *Handler) handlePersist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	var payload struct {
		DialogHash string                   `json:"dialog_hash"`
		UserHash   string                   `json:"user_hash"`
		Transcript string                   `json:"transcript"`
		Candidates []models.AnswerCandidate `json:"candidates"`
		MetaJSON   string                   `json:"meta_json,omitempty"`
	}

	if err := httpx.DecodeJSON(w, r, &payload, maxAdminBodyBytes); err != nil {
		status, code, message := httpx.JSONError(err)
		httpx.WriteError(w, status, code, message)
		return
	}

	if strings.TrimSpace(payload.DialogHash) == "" {
		httpx.WriteError(w, http.StatusBadRequest, "missing_dialog_hash", "dialog_hash is required")
		return
	}

	record := ResultRecord{
		DialogHash: strings.TrimSpace(payload.DialogHash),
		UserHash:   payload.UserHash,
		Transcript: payload.Transcript,
		Candidates: payload.Candidates,
		MetaJSON:   payload.MetaJSON,
		CreatedAt:  time.Now().UTC(),
	}

	if err := h.store.PutResult(r.Context(), record); err != nil {
		log.Printf("persist failed: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "persist_failed", "persist failed")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, httpx.StatusResponse{Status: "ok"})
}

func (h *Handler) handleCacheLookup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	prefix := "/internal/cache/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	dialogHash := strings.TrimPrefix(r.URL.Path, prefix)
	if dialogHash == "" {
		httpx.WriteError(w, http.StatusBadRequest, "missing_dialog_hash", "dialog_hash is required")
		return
	}

	record, ok, err := h.store.GetResult(r.Context(), dialogHash)
	if err != nil {
		log.Printf("cache lookup failed: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "cache_failed", "cache lookup failed")
		return
	}
	if !ok {
		httpx.WriteError(w, http.StatusNotFound, "cache_miss", "not found")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, record)
}

func (h *Handler) handleKeyVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	var payload struct {
		APIKey     string `json:"api_key"`
		APIKeyHash string `json:"api_key_hash"`
	}

	if err := httpx.DecodeJSON(w, r, &payload, maxAdminBodyBytes); err != nil {
		status, code, message := httpx.JSONError(err)
		httpx.WriteError(w, status, code, message)
		return
	}

	apiKeyHash := strings.TrimSpace(payload.APIKeyHash)
	apiKeyPlain := strings.TrimSpace(payload.APIKey)
	if apiKeyHash == "" && apiKeyPlain != "" {
		apiKeyHash = hash.HashAPIKey(apiKeyPlain)
	}
	if apiKeyHash == "" {
		httpx.WriteError(w, http.StatusBadRequest, "missing_api_key", "api_key or api_key_hash required")
		return
	}

	record, ok, err := h.store.GetAPIKey(r.Context(), apiKeyHash)
	if err != nil {
		log.Printf("api key lookup failed: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "key_lookup_failed", "api key lookup failed")
		return
	}
	if !ok || record.Status != auth.StatusActive {
		httpx.WriteError(w, http.StatusUnauthorized, "invalid_key", "invalid api key")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"client_id":             record.ClientID,
		"status":                record.Status,
		"scopes":                record.Scopes,
		"rate_limit_per_minute": record.RateLimitPerMinute,
	})
}

func (h *Handler) handleConfigGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	prefix := "/internal/config/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	key := strings.TrimPrefix(r.URL.Path, prefix)
	if key == "" {
		httpx.WriteError(w, http.StatusBadRequest, "missing_key", "key is required")
		return
	}

	item, ok, err := h.store.GetConfig(r.Context(), key)
	if err != nil {
		log.Printf("get config failed: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "config_failed", "config lookup failed")
		return
	}
	if !ok {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "config not found")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) handleAdminKeys(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		limit, err := parseLimitParam(r, defaultListLimit, maxListLimit)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_limit", err.Error())
			return
		}
		includeRevoked := parseBoolParam(r, "include_revoked")
		records, err := h.store.ListAPIKeys(r.Context(), APIKeyListOptions{
			Limit:          limit,
			IncludeRevoked: includeRevoked,
		})
		if err != nil {
			log.Printf("list api keys failed: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, "key_list_failed", "api key list failed")
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"items": records,
			"count": len(records),
		})
	case http.MethodPost:
		var req models.APIKeyCreateRequest
		if err := httpx.DecodeJSON(w, r, &req, maxAdminBodyBytes); err != nil {
			status, code, message := httpx.JSONError(err)
			httpx.WriteError(w, status, code, message)
			return
		}

		clientID, status, scopes, rateLimit, err := normalizeAPIKeyRequest(req)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}

		plain, hashed, err := auth.GenerateAPIKey()
		if err != nil {
			log.Printf("generate api key failed: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, "key_generation_failed", "api key generation failed")
			return
		}

		record := APIKeyRecord{
			APIKeyHash:         hashed,
			ClientID:           clientID,
			Status:             status,
			Scopes:             scopes,
			RateLimitPerMinute: rateLimit,
			CreatedAt:          time.Now().UTC(),
		}

		if err := h.store.CreateAPIKey(r.Context(), record); err != nil {
			log.Printf("create api key failed: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, "key_create_failed", "api key create failed")
			return
		}

		resp := models.APIKeyCreateResponse{
			APIKey:             plain,
			APIKeyHash:         hashed,
			ClientID:           clientID,
			Status:             status,
			Scopes:             scopes,
			RateLimitPerMinute: rateLimit,
			CreatedAt:          record.CreatedAt,
		}

		httpx.WriteJSON(w, http.StatusOK, resp)
	default:
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

func (h *Handler) handleAdminPrompt(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		item, ok, err := h.store.GetConfig(r.Context(), SystemPromptConfigKey)
		if err != nil {
			log.Printf("get system prompt failed: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, "config_failed", "config lookup failed")
			return
		}
		versionItem, versionOK, err := h.store.GetConfig(r.Context(), SystemPromptVersionConfigKey)
		if err != nil {
			log.Printf("get system prompt version failed: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, "config_failed", "config lookup failed")
			return
		}
		promptValue := ""
		versionValue := ""
		if ok {
			promptValue = item.Value
		}
		if versionOK {
			versionValue = versionItem.Value
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]string{
			"prompt":  promptValue,
			"version": versionValue,
		})
	case http.MethodPost:
		var payload struct {
			Prompt  string `json:"prompt"`
			Version string `json:"version"`
		}
		if err := httpx.DecodeJSON(w, r, &payload, maxAdminBodyBytes); err != nil {
			status, code, message := httpx.JSONError(err)
			httpx.WriteError(w, status, code, message)
			return
		}
		if strings.TrimSpace(payload.Prompt) == "" {
			httpx.WriteError(w, http.StatusBadRequest, "missing_prompt", "prompt is required")
			return
		}
		if utf8.RuneCountInString(payload.Prompt) > maxPromptRunes {
			httpx.WriteError(w, http.StatusBadRequest, "prompt_too_long", "prompt is too long")
			return
		}
		if strings.TrimSpace(payload.Version) == "" {
			payload.Version = time.Now().UTC().Format("20060102150405")
		}
		if err := h.store.SetConfig(r.Context(), SystemPromptConfigKey, payload.Prompt); err != nil {
			log.Printf("set system prompt failed: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, "config_failed", "config update failed")
			return
		}
		if err := h.store.SetConfig(r.Context(), SystemPromptVersionConfigKey, payload.Version); err != nil {
			log.Printf("set system prompt version failed: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, "config_failed", "config update failed")
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.StatusResponse{Status: "ok"})
	default:
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

func (h *Handler) handleAdminKeyByID(w http.ResponseWriter, r *http.Request) {
	prefix := "/v1/admin/keys/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	keyPart := strings.TrimPrefix(r.URL.Path, prefix)
	if keyPart == "" || strings.Contains(keyPart, "/") {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "not found")
		return
	}

	if keyPart == "revoke" && r.Method == http.MethodPost {
		var payload struct {
			APIKey     string `json:"api_key"`
			APIKeyHash string `json:"api_key_hash"`
		}
		if err := httpx.DecodeJSON(w, r, &payload, maxAdminBodyBytes); err != nil {
			status, code, message := httpx.JSONError(err)
			httpx.WriteError(w, status, code, message)
			return
		}
		apiKeyHash := strings.TrimSpace(payload.APIKeyHash)
		if apiKeyHash == "" && strings.TrimSpace(payload.APIKey) != "" {
			apiKeyHash = hash.HashAPIKey(payload.APIKey)
		}
		if apiKeyHash == "" {
			httpx.WriteError(w, http.StatusBadRequest, "missing_api_key", "api_key or api_key_hash required")
			return
		}
		if err := h.store.RevokeAPIKey(r.Context(), apiKeyHash, time.Now().UTC()); err != nil {
			log.Printf("revoke api key failed: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, "key_revoke_failed", "api key revoke failed")
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.StatusResponse{Status: "ok"})
		return
	}

	if r.Method != http.MethodDelete {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	apiKeyHash := strings.TrimSpace(keyPart)
	if strings.HasPrefix(apiKeyHash, "dhk_") {
		apiKeyHash = hash.HashAPIKey(apiKeyHash)
	}
	if apiKeyHash == "" {
		httpx.WriteError(w, http.StatusBadRequest, "missing_api_key", "api key is required")
		return
	}
	if err := h.store.RevokeAPIKey(r.Context(), apiKeyHash, time.Now().UTC()); err != nil {
		log.Printf("revoke api key failed: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "key_revoke_failed", "api key revoke failed")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.StatusResponse{Status: "ok"})
}

func (h *Handler) handleAdminLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	limit, err := parseLimitParam(r, defaultListLimit, maxListLimit)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_limit", err.Error())
		return
	}
	since, window, err := parseWindowParam(r, 24*time.Hour, 30*24*time.Hour)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_window", err.Error())
		return
	}

	if h.logReader == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "logging_not_configured", "runtime logging is not configured")
		return
	}

	entries, err := h.logReader.Read(r.Context(), cloudlogging.ReadOptions{
		Since: since,
		Until: time.Now().UTC(),
		Limit: limit,
	})
	if err != nil {
		log.Printf("list logs failed: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "log_list_failed", "log list failed")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"items":          entries,
		"count":          len(entries),
		"since":          since.Format(time.RFC3339),
		"window_seconds": int(window.Seconds()),
	})
}

func (h *Handler) handleAdminStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	since, window, err := parseWindowParam(r, 24*time.Hour, 30*24*time.Hour)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_window", err.Error())
		return
	}

	total, err := h.store.CountResults(r.Context(), since)
	if err != nil {
		log.Printf("count results failed: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "stats_failed", "stats query failed")
		return
	}
	activeUsers, err := h.store.CountActiveUsers(r.Context(), since)
	if err != nil {
		log.Printf("count active users failed: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "stats_failed", "stats query failed")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"since":          since.Format(time.RFC3339),
		"window_seconds": int(window.Seconds()),
		"total_requests": total,
		"active_users":   activeUsers,
	})
}

func (h *Handler) handleAdminOTPSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if h.telegramToken == "" || h.telegramAdminID == 0 {
		httpx.WriteError(w, http.StatusServiceUnavailable, "otp_not_configured", "otp delivery is not configured")
		return
	}

	now := time.Now().UTC()
	var existing otpConfig
	if ok, err := h.getConfigJSON(r.Context(), AdminOTPConfigKey, &existing); err != nil {
		log.Printf("otp lookup failed: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "otp_failed", "otp request failed")
		return
	} else if ok && !existing.SentAt.IsZero() && now.Sub(existing.SentAt) < minOTPSendInterval {
		httpx.WriteError(w, http.StatusTooManyRequests, "otp_rate_limited", "please wait before requesting a new code")
		return
	}

	code, err := generateOTP(otpDigits)
	if err != nil {
		log.Printf("otp generation failed: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "otp_failed", "otp request failed")
		return
	}
	otpPayload := otpConfig{
		Hash:      hash.SHA256HexString(code),
		ExpiresAt: now.Add(h.otpTTL),
		SentAt:    now,
	}
	if err := h.setConfigJSON(r.Context(), AdminOTPConfigKey, otpPayload); err != nil {
		log.Printf("otp store failed: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "otp_failed", "otp request failed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	message := fmt.Sprintf("LINKa Dialog Helper: login code %s. Valid for %d minutes.", code, int(h.otpTTL.Minutes()))
	if err := h.sendTelegramMessage(ctx, message); err != nil {
		log.Printf("otp send failed: %v", err)
		httpx.WriteError(w, http.StatusBadGateway, "otp_send_failed", "otp send failed")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, httpx.StatusResponse{Status: "ok"})
}

func (h *Handler) handleAdminOTPVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	var payload struct {
		Code string `json:"code"`
	}
	if err := httpx.DecodeJSON(w, r, &payload, maxAdminBodyBytes); err != nil {
		status, code, message := httpx.JSONError(err)
		httpx.WriteError(w, status, code, message)
		return
	}

	code := strings.TrimSpace(payload.Code)
	if !isValidOTP(code, otpDigits) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_code", "otp code is invalid")
		return
	}

	var stored otpConfig
	ok, err := h.getConfigJSON(r.Context(), AdminOTPConfigKey, &stored)
	if err != nil {
		log.Printf("otp lookup failed: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "otp_failed", "otp verification failed")
		return
	}
	if !ok || stored.Hash == "" {
		httpx.WriteError(w, http.StatusUnauthorized, "otp_missing", "otp code not found")
		return
	}
	if time.Now().UTC().After(stored.ExpiresAt) {
		httpx.WriteError(w, http.StatusUnauthorized, "otp_expired", "otp code expired")
		return
	}
	if !auth.ConstantTimeEquals(hash.SHA256HexString(code), stored.Hash) {
		httpx.WriteError(w, http.StatusUnauthorized, "otp_invalid", "otp code invalid")
		return
	}

	token, tokenHash, err := generateSessionToken()
	if err != nil {
		log.Printf("session token generation failed: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "session_failed", "session creation failed")
		return
	}

	session := sessionConfig{
		Hash:      tokenHash,
		ExpiresAt: time.Now().UTC().Add(h.sessionTTL),
	}
	if err := h.setConfigJSON(r.Context(), AdminSessionConfigKey, session); err != nil {
		log.Printf("session store failed: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "session_failed", "session creation failed")
		return
	}
	_ = h.setConfigJSON(r.Context(), AdminOTPConfigKey, otpConfig{Hash: "", ExpiresAt: time.Now().UTC(), SentAt: time.Now().UTC()})

	httpx.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"session_token": token,
		"expires_at":    session.ExpiresAt.Format(time.RFC3339),
	})
}

func (h *Handler) withInternalAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-DH-Internal-Token")
		if token == "" {
			token = r.Header.Get("X-Internal-Token")
		}
		if token == "" || token != h.internalToken {
			httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid internal token")
			return
		}
		next(w, r)
	}
}

func (h *Handler) withAdminAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionToken := strings.TrimSpace(r.Header.Get("X-ADMIN-SESSION"))
		if sessionToken != "" {
			ok, err := h.verifyAdminSession(r.Context(), sessionToken)
			if err != nil {
				log.Printf("admin session lookup failed: %v", err)
				httpx.WriteError(w, http.StatusInternalServerError, "admin_session_failed", "admin session lookup failed")
				return
			}
			if ok {
				next(w, r)
				return
			}
		}

		key := strings.TrimSpace(r.Header.Get("X-ADMIN-KEY"))
		if key == "" {
			httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing admin key or session")
			return
		}
		hashValue, err := h.getAdminKeyHash(r.Context())
		if err != nil {
			log.Printf("admin key lookup failed: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, "admin_key_lookup_failed", "admin key lookup failed")
			return
		}
		if hashValue == "" || !auth.ConstantTimeEquals(hash.HashAPIKey(key), hashValue) {
			httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid admin key")
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

func (h *Handler) verifyAdminSession(ctx context.Context, token string) (bool, error) {
	if strings.TrimSpace(token) == "" {
		return false, nil
	}
	var stored sessionConfig
	ok, err := h.getConfigJSON(ctx, AdminSessionConfigKey, &stored)
	if err != nil || !ok {
		return false, err
	}
	if stored.Hash == "" || time.Now().UTC().After(stored.ExpiresAt) {
		return false, nil
	}
	return auth.ConstantTimeEquals(hash.SHA256HexString(token), stored.Hash), nil
}

func (h *Handler) getConfigJSON(ctx context.Context, key string, out interface{}) (bool, error) {
	item, ok, err := h.store.GetConfig(ctx, key)
	if err != nil || !ok {
		return ok, err
	}
	if strings.TrimSpace(item.Value) == "" {
		return false, nil
	}
	if err := json.Unmarshal([]byte(item.Value), out); err != nil {
		return false, err
	}
	return true, nil
}

func (h *Handler) setConfigJSON(ctx context.Context, key string, value interface{}) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return h.store.SetConfig(ctx, key, string(payload))
}

func (h *Handler) sendTelegramMessage(ctx context.Context, message string) error {
	if h.telegramToken == "" || h.telegramAdminID == 0 {
		return errors.New("telegram is not configured")
	}

	form := url.Values{}
	form.Set("chat_id", strconv.FormatInt(h.telegramAdminID, 10))
	form.Set("text", message)
	form.Set("disable_web_page_preview", "true")

	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", h.telegramToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 == 2 {
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("telegram send failed: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
}

func parseLimitParam(r *http.Request, def, max int) (int, error) {
	raw := strings.TrimSpace(r.URL.Query().Get("limit"))
	if raw == "" {
		return def, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, errors.New("limit must be a positive integer")
	}
	if value > max {
		return max, nil
	}
	return value, nil
}

func parseBoolParam(r *http.Request, key string) bool {
	raw := strings.TrimSpace(strings.ToLower(r.URL.Query().Get(key)))
	return raw == "1" || raw == "true" || raw == "yes"
}

func parseWindowParam(r *http.Request, def time.Duration, max time.Duration) (time.Time, time.Duration, error) {
	now := time.Now().UTC()
	if raw := strings.TrimSpace(r.URL.Query().Get("since")); raw != "" {
		since, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return time.Time{}, 0, errors.New("since must be RFC3339")
		}
		if since.After(now) {
			return time.Time{}, 0, errors.New("since must be in the past")
		}
		window := now.Sub(since)
		if max > 0 && window > max {
			window = max
			since = now.Add(-window)
		}
		return since, window, nil
	}

	window := def
	if raw := strings.TrimSpace(r.URL.Query().Get("window")); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil || parsed <= 0 {
			return time.Time{}, 0, errors.New("window must be a valid duration")
		}
		window = parsed
	}
	if max > 0 && window > max {
		window = max
	}
	return now.Add(-window), window, nil
}

func normalizeAPIKeyRequest(req models.APIKeyCreateRequest) (string, string, []string, int, error) {
	clientID := strings.TrimSpace(req.ClientID)
	if clientID == "" {
		clientID = defaultClientID
	}
	if utf8.RuneCountInString(clientID) > maxClientIDRunes {
		return "", "", nil, 0, errors.New("client_id is too long")
	}

	status := strings.ToLower(strings.TrimSpace(req.Status))
	if status == "" {
		status = defaultKeyStatus
	}
	if _, ok := validKeyStatuses[status]; !ok {
		return "", "", nil, 0, errors.New("unsupported status")
	}

	scopes, err := normalizeScopes(req.Scopes)
	if err != nil {
		return "", "", nil, 0, err
	}

	rateLimit := req.RateLimitPerMinute
	if rateLimit < 0 {
		return "", "", nil, 0, errors.New("rate_limit_per_minute must be >= 0")
	}

	return clientID, status, scopes, rateLimit, nil
}

func normalizeScopes(scopes []string) ([]string, error) {
	if len(scopes) == 0 {
		return nil, nil
	}
	if len(scopes) > maxScopeCount {
		return nil, errors.New("too many scopes")
	}
	seen := make(map[string]struct{})
	out := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		cleaned := strings.ToLower(strings.TrimSpace(scope))
		if cleaned == "" {
			continue
		}
		if utf8.RuneCountInString(cleaned) > maxScopeRunes {
			return nil, errors.New("scope is too long")
		}
		if _, ok := validScopes[cleaned]; !ok {
			return nil, fmt.Errorf("unsupported scope: %s", cleaned)
		}
		if _, ok := seen[cleaned]; ok {
			continue
		}
		seen[cleaned] = struct{}{}
		out = append(out, cleaned)
	}
	return out, nil
}

func generateOTP(digits int) (string, error) {
	if digits <= 0 {
		return "", errors.New("invalid otp length")
	}
	maxValue := pow10(digits)
	n, err := rand.Int(rand.Reader, big.NewInt(maxValue))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%0*d", digits, n.Int64()), nil
}

func generateSessionToken() (string, string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	token := hex.EncodeToString(buf)
	return token, hash.SHA256HexString(token), nil
}

func isValidOTP(value string, digits int) bool {
	if len(value) != digits {
		return false
	}
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func pow10(exp int) int64 {
	value := int64(1)
	for i := 0; i < exp; i++ {
		value *= 10
	}
	return value
}

type otpConfig struct {
	Hash      string    `json:"hash"`
	ExpiresAt time.Time `json:"expires_at"`
	SentAt    time.Time `json:"sent_at"`
}

type sessionConfig struct {
	Hash      string    `json:"hash"`
	ExpiresAt time.Time `json:"expires_at"`
}

var validKeyStatuses = map[string]struct{}{
	auth.StatusActive:    {},
	auth.StatusSuspended: {},
	auth.StatusRevoked:   {},
}

var validScopes = map[string]struct{}{
	"infer": {},
}
