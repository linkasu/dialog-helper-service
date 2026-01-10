package inference

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/linkasu/dialog-helper-service/internal/auth"
	"github.com/linkasu/dialog-helper-service/internal/config"
	"github.com/linkasu/dialog-helper-service/internal/hash"
	"github.com/linkasu/dialog-helper-service/internal/httpx"
	"github.com/linkasu/dialog-helper-service/internal/models"
	"github.com/linkasu/dialog-helper-service/internal/normalize"
	"github.com/linkasu/dialog-helper-service/internal/prompt"
	"github.com/linkasu/dialog-helper-service/internal/providers"
	"github.com/linkasu/dialog-helper-service/internal/storage"
	"github.com/linkasu/dialog-helper-service/internal/storageclient"
)

const (
	formPayloadField = "payload"
)

var (
	errPayloadTooLarge  = errors.New("payload too large")
	errInvalidMultipart = errors.New("invalid multipart payload")
	errAudioMissing     = errors.New("audio file is required")
)

type Handler struct {
	cfg          config.InferenceConfig
	storage      *storageclient.Client
	stt          providers.STTClient
	llm          providers.LLMClient
	promptLoader *PromptLoader
	keyCache     *auth.KeyCache
	rateLimiter  *auth.RateLimiter
}

func NewHandler(cfg config.InferenceConfig, storageClient *storageclient.Client, stt providers.STTClient, llm providers.LLMClient, promptLoader *PromptLoader) *Handler {
	return &Handler{
		cfg:          cfg,
		storage:      storageClient,
		stt:          stt,
		llm:          llm,
		promptLoader: promptLoader,
		keyCache:     auth.NewKeyCache(1 * time.Minute),
		rateLimiter:  auth.NewRateLimiter(),
	}
}

func (h *Handler) Register(mux *http.ServeMux) {
	h.RegisterRoutes(mux, true)
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux, includeHealth bool) {
	if includeHealth {
		mux.HandleFunc("/healthz", h.handleHealth)
	}
	mux.HandleFunc("/v1/infer", h.handleInfer)
	mux.HandleFunc("/client.md", h.handleClientDoc)
}

func (h *Handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, httpx.StatusResponse{Status: "ok"})
}

func (h *Handler) handleClientDoc(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	doc, err := loadClientDoc()
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "client doc not found")
		return
	}

	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(doc)
}

func (h *Handler) handleInfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	requestID := newRequestID()
	w.Header().Set("X-Request-ID", requestID)

	apiKey := strings.TrimSpace(r.Header.Get("X-API-KEY"))
	if apiKey == "" {
		httpx.WriteError(w, http.StatusUnauthorized, "missing_api_key", "X-API-KEY header required")
		return
	}
	apiKeyHash := auth.HashAPIKey(apiKey)

	keyInfo, ok := h.keyCache.Get(apiKeyHash)
	if !ok {
		info, ok, err := h.storage.VerifyAPIKey(r.Context(), apiKeyHash)
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, storageclient.ErrUnauthorized) {
				status = http.StatusUnauthorized
			}
			httpx.WriteError(w, status, "auth_failed", "api key verification failed")
			return
		}
		if !ok || info.Status != auth.StatusActive {
			httpx.WriteError(w, http.StatusUnauthorized, "invalid_api_key", "invalid api key")
			return
		}
		keyInfo = info
		h.keyCache.Set(apiKeyHash, info)
	}

	if !h.rateLimiter.Allow(apiKeyHash, keyInfo.RateLimitPerMinute) {
		httpx.WriteError(w, http.StatusTooManyRequests, "rate_limited", "rate limit exceeded")
		return
	}

	payload, audioBytes, audioType, err := readMultipartPayload(w, r, h.cfg.MaxAudioBytes)
	if err != nil {
		status := http.StatusBadRequest
		code := "invalid_request"
		if errors.Is(err, errPayloadTooLarge) {
			status = http.StatusRequestEntityTooLarge
			code = "payload_too_large"
		}
		httpx.WriteError(w, status, code, err.Error())
		return
	}

	if err := validatePayload(h.cfg, payload); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	normalizedMessages, audioIndex, err := normalizeDialogMessages(payload.Messages, h.cfg.MaxMessages, h.cfg.MaxMessageChars)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	hasAudio := len(audioBytes) > 0
	if hasAudio && audioIndex == -1 {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "audio message is required")
		return
	}
	if !hasAudio && audioIndex != -1 {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "audio file is required")
		return
	}

	language := strings.TrimSpace(payload.Language)
	if language == "" {
		language = h.cfg.DefaultLanguage
	}

	userHash := hash.HashUser(payload.UserID, payload.UserProfile)
	promptInput := prompt.BuildUserMessage(payload.DisabledPersonBiography, normalizedMessages, "")
	promptHash := hash.HashPrompt(promptInput)
	audioHash := hash.SHA256HexString("no-audio")
	if len(audioBytes) > 0 {
		audioHash = hash.SHA256HexBytes(audioBytes)
	}

	dialogHash := hash.BuildDialogHash(hash.DialogHashInput{
		APIKeyHash: apiKeyHash,
		UserHash:   userHash,
		PromptHash: promptHash,
		AudioHash:  audioHash,
		Language:   language,
		Persona:    payload.Persona,
		DialogID:   payload.DialogID,
		StepID:     payload.StepID,
		Now:        time.Now().UTC(),
		WindowSec:  chooseTimeWindow(payload.TimeWindowSec, h.cfg.TimeWindowSec),
	})

	if cached, ok, err := h.storage.GetResult(r.Context(), dialogHash); err == nil && ok {
		resp := models.InferResponse{
			Response: candidatesToTexts(cached.Candidates),
		}
		if hasAudio {
			transcript := cached.Transcript
			resp.Transcript = &transcript
		}
		httpx.WriteJSON(w, http.StatusOK, resp)
		return
	}

	transcript := ""
	sttLatency := int64(0)
	dialogMessages := normalizedMessages
	if len(audioBytes) > 0 {
		start := time.Now()
		sttResult, err := h.stt.Transcribe(r.Context(), providers.STTRequest{
			Audio:              audioBytes,
			ContainerAudioType: audioType,
			Language:           language,
		})
		sttLatency = time.Since(start).Milliseconds()
		if err != nil {
			log.Printf("stt error: %v", err)
			httpx.WriteError(w, http.StatusBadGateway, "stt_failed", "speech-to-text failed")
			return
		}
		transcript = strings.TrimSpace(sttResult.Transcript)
		dialogMessages = applyTranscriptToMessages(normalizedMessages, audioIndex, transcript)
	}

	promptInfo, err := h.promptLoader.Get(r.Context())
	if err != nil {
		log.Printf("prompt load error: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "prompt_failed", "prompt load failed")
		return
	}

	systemPrompt := prompt.NormalizeSystemPrompt(promptInfo.Prompt)
	userPrompt := prompt.BuildUserMessage(payload.DisabledPersonBiography, dialogMessages, transcript)
	if userPrompt == "" {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "messages are required")
		return
	}

	maxCandidates := h.cfg.DefaultMaxCandidates
	if payload.Options != nil && payload.Options.MaxCandidates > 0 {
		maxCandidates = payload.Options.MaxCandidates
	}
	temperature := h.cfg.DefaultTemperature
	if payload.Options != nil && payload.Options.Temperature > 0 {
		temperature = payload.Options.Temperature
	}

	llmLatency := int64(0)
	start := time.Now()
	llmResult, err := h.llm.Complete(r.Context(), providers.LLMRequest{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		Temperature:  temperature,
		MaxTokens:    h.cfg.DefaultMaxTokens,
		ModelURI:     h.cfg.ModelURI,
	})
	llmLatency = time.Since(start).Milliseconds()
	if err != nil {
		log.Printf("llm error: %v", err)
		httpx.WriteError(w, http.StatusBadGateway, "llm_failed", "llm request failed")
		return
	}
	candidates := normalize.NormalizeCandidates(llmResult.Texts, maxCandidates, language)

	resp := models.InferResponse{
		Response: candidatesToTexts(candidates),
	}
	if hasAudio {
		resp.Transcript = &transcript
	}

	if h.cfg.PersistResults {
		metaPayload, _ := json.Marshal(map[string]interface{}{
			"stt_latency_ms":        sttLatency,
			"llm_latency_ms":        llmLatency,
			"system_prompt_version": promptInfo.Version,
		})
		record := storage.ResultRecord{
			DialogHash: dialogHash,
			UserHash:   userHash,
			Transcript: transcript,
			Candidates: candidates,
			MetaJSON:   string(metaPayload),
			CreatedAt:  time.Now().UTC(),
		}
		go h.persistAsync(context.Background(), record)
	}

	httpx.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) persistAsync(ctx context.Context, record storage.ResultRecord) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := h.storage.Persist(ctx, record); err != nil {
		log.Printf("persist error: %v", err)
	}
}

func chooseTimeWindow(requested, fallback int) int {
	if requested > 0 {
		return requested
	}
	if fallback > 0 {
		return fallback
	}
	return 600
}

func readMultipartPayload(w http.ResponseWriter, r *http.Request, maxBytes int64) (models.InferPayload, []byte, string, error) {
	if r.Body == nil {
		return models.InferPayload{}, nil, "", errors.New("empty request body")
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	if err := r.ParseMultipartForm(maxBytes); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return models.InferPayload{}, nil, "", errPayloadTooLarge
		}
		return models.InferPayload{}, nil, "", errInvalidMultipart
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()

	payloadField := r.FormValue(formPayloadField)
	if payloadField == "" {
		payloadField = r.FormValue("metadata")
	}
	if payloadField == "" {
		payloadField = r.FormValue("json")
	}

	var payload models.InferPayload
	if payloadField != "" {
		if err := json.Unmarshal([]byte(payloadField), &payload); err != nil {
			return models.InferPayload{}, nil, "", errors.New("invalid payload json")
		}
	}

	file, header, err := findAudioFile(r.MultipartForm)
	if err != nil {
		if errors.Is(err, errAudioMissing) {
			return payload, nil, "", nil
		}
		return payload, nil, "", err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return models.InferPayload{}, nil, "", errors.New("failed to read audio")
	}

	contentType := header.Header.Get("Content-Type")
	containerType, err := resolveContainerAudioType(contentType, header.Filename)
	if err != nil {
		return models.InferPayload{}, nil, "", err
	}

	return payload, data, containerType, nil
}

func findAudioFile(form *multipart.Form) (multipart.File, *multipart.FileHeader, error) {
	if form == nil {
		return nil, nil, errInvalidMultipart
	}
	for _, field := range []string{"audio", "file"} {
		if headers := form.File[field]; len(headers) > 0 {
			file, err := headers[0].Open()
			if err != nil {
				return nil, nil, errors.New("failed to open audio")
			}
			return file, headers[0], nil
		}
	}
	for _, headers := range form.File {
		if len(headers) == 0 {
			continue
		}
		file, err := headers[0].Open()
		if err != nil {
			return nil, nil, errors.New("failed to open audio")
		}
		return file, headers[0], nil
	}
	return nil, nil, errAudioMissing
}

func resolveContainerAudioType(contentType, fileName string) (string, error) {
	lowerName := strings.ToLower(fileName)
	lowerType := strings.ToLower(strings.Split(contentType, ";")[0])

	switch {
	case strings.HasSuffix(lowerName, ".ogg"):
		return "OGG_OPUS", nil
	case strings.HasSuffix(lowerName, ".mp3"):
		return "MP3", nil
	case strings.HasSuffix(lowerName, ".wav"):
		return "WAV", nil
	case strings.HasSuffix(lowerName, ".webm"):
		return "WEBM_OPUS", nil
	case lowerType == "audio/ogg" || lowerType == "application/ogg":
		return "OGG_OPUS", nil
	case lowerType == "audio/mpeg" || lowerType == "audio/mp3" || lowerType == "audio/x-mp3" || lowerType == "audio/x-mpeg":
		return "MP3", nil
	case lowerType == "audio/wav" || lowerType == "audio/x-wav":
		return "WAV", nil
	case lowerType == "audio/webm" || lowerType == "video/webm":
		return "WEBM_OPUS", nil
	default:
		return "", errors.New("unsupported audio type")
	}
}

func loadClientDoc() ([]byte, error) {
	paths := make([]string, 0, 4)
	if envPath := strings.TrimSpace(os.Getenv("CLIENT_DOC_PATH")); envPath != "" {
		paths = append(paths, envPath)
	}
	if cwd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(cwd, "client.md"))
	}
	if exe, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(exe), "client.md"))
	}
	paths = append(paths, "/app/client.md")

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err == nil && len(data) > 0 {
			return data, nil
		}
	}
	return nil, errors.New("client doc not found")
}

func newRequestID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format("20060102150405")))
	}
	return hex.EncodeToString(buf)
}

func validatePayload(cfg config.InferenceConfig, payload models.InferPayload) error {
	if cfg.MaxBiographyChars > 0 && utf8.RuneCountInString(payload.DisabledPersonBiography) > cfg.MaxBiographyChars {
		return errors.New("disabled_person_biography is too long")
	}
	if cfg.MaxPersonaChars > 0 && utf8.RuneCountInString(payload.Persona) > cfg.MaxPersonaChars {
		return errors.New("persona is too long")
	}
	if cfg.MaxUserIDChars > 0 && utf8.RuneCountInString(payload.UserID) > cfg.MaxUserIDChars {
		return errors.New("user_id is too long")
	}
	if cfg.MaxUserProfileChars > 0 && utf8.RuneCountInString(payload.UserProfile) > cfg.MaxUserProfileChars {
		return errors.New("user_profile is too long")
	}
	if cfg.MaxDialogIDChars > 0 && utf8.RuneCountInString(payload.DialogID) > cfg.MaxDialogIDChars {
		return errors.New("dialog_id is too long")
	}
	if cfg.MaxStepIDChars > 0 && utf8.RuneCountInString(payload.StepID) > cfg.MaxStepIDChars {
		return errors.New("step_id is too long")
	}
	if cfg.MaxLanguageChars > 0 && utf8.RuneCountInString(payload.Language) > cfg.MaxLanguageChars {
		return errors.New("language is too long")
	}
	if payload.TimeWindowSec < 0 {
		return errors.New("time_window_sec must be positive")
	}
	if payload.Options != nil {
		if payload.Options.MaxCandidates < 0 {
			return errors.New("max_candidates must be positive")
		}
		if payload.Options.MaxCandidates > 0 && cfg.MaxCandidates > 0 && payload.Options.MaxCandidates > cfg.MaxCandidates {
			return errors.New("max_candidates exceeds limit")
		}
		if payload.Options.Temperature < 0 || payload.Options.Temperature > cfg.MaxTemperature {
			return errors.New("temperature is out of range")
		}
	}
	return nil
}

func normalizeDialogMessages(messages []models.DialogMessage, maxMessages, maxContentRunes int) ([]models.DialogMessage, int, error) {
	if len(messages) == 0 {
		return nil, -1, errors.New("messages are required")
	}
	if maxMessages > 0 && len(messages) > maxMessages {
		return nil, -1, errors.New("too many messages")
	}

	normalized := make([]models.DialogMessage, 0, len(messages))
	audioIndex := -1
	for _, msg := range messages {
		role := strings.ToLower(strings.TrimSpace(msg.Role))
		if role == "" {
			return nil, -1, errors.New("message role is required")
		}
		if role != "speaker" && role != "disabled_person" {
			return nil, -1, errors.New("unsupported message role")
		}
		content := strings.TrimSpace(msg.Content)
		if msg.Audio {
			if role != "speaker" {
				return nil, -1, errors.New("audio is only supported for speaker role")
			}
			if audioIndex != -1 {
				return nil, -1, errors.New("only one audio message is supported")
			}
			audioIndex = len(normalized)
			content = ""
		} else if content == "" {
			return nil, -1, errors.New("message content is required")
		} else if maxContentRunes > 0 && utf8.RuneCountInString(content) > maxContentRunes {
			return nil, -1, errors.New("message content is too long")
		}
		normalized = append(normalized, models.DialogMessage{
			Role:    role,
			Content: content,
			Audio:   msg.Audio,
		})
	}

	return normalized, audioIndex, nil
}

func applyTranscriptToMessages(messages []models.DialogMessage, index int, transcript string) []models.DialogMessage {
	if index < 0 || index >= len(messages) {
		return messages
	}
	updated := make([]models.DialogMessage, len(messages))
	copy(updated, messages)
	updated[index].Content = strings.TrimSpace(transcript)
	updated[index].Audio = false
	return updated
}

func candidatesToTexts(candidates []models.AnswerCandidate) []string {
	if len(candidates) == 0 {
		return nil
	}
	texts := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate.Text) == "" {
			continue
		}
		texts = append(texts, candidate.Text)
	}
	return texts
}
