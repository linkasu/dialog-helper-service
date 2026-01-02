package iam

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"sync"
	"time"
)

const (
	defaultMetadataURL = "http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/token"
	metadataHeaderKey  = "Metadata-Flavor"
	metadataHeaderVal  = "Google"
)

type Token struct {
	Value     string
	ExpiresAt time.Time
}

type Source interface {
	Token(ctx context.Context) (Token, error)
}

type Provider struct {
	source Source
	skew   time.Duration
	mu     sync.Mutex
	cached Token
}

func NewProvider(source Source) *Provider {
	return &Provider{source: source, skew: time.Minute}
}

func NewDefaultProvider() *Provider {
	if env, ok := NewEnvSourceFromEnv(); ok {
		return NewProvider(env)
	}
	return NewProvider(NewMetadataSource(defaultMetadataURL))
}

func (p *Provider) Token(ctx context.Context) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now().UTC()
	if p.cached.Value != "" && now.Add(p.skew).Before(p.cached.ExpiresAt) {
		return p.cached.Value, nil
	}

	fresh, err := p.source.Token(ctx)
	if err != nil {
		return "", err
	}

	p.cached = fresh
	return fresh.Value, nil
}

type EnvSource struct {
	token string
}

func NewEnvSourceFromEnv() (*EnvSource, bool) {
	value := os.Getenv("YC_IAM_TOKEN")
	if value == "" {
		return nil, false
	}
	return &EnvSource{token: value}, true
}

func (e *EnvSource) Token(_ context.Context) (Token, error) {
	if e.token == "" {
		return Token{}, errors.New("empty YC_IAM_TOKEN")
	}
	return Token{Value: e.token, ExpiresAt: time.Now().UTC().Add(24 * time.Hour)}, nil
}

type MetadataSource struct {
	endpoint string
	client   *http.Client
}

func NewMetadataSource(endpoint string) *MetadataSource {
	return &MetadataSource{
		endpoint: endpoint,
		client: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

func (m *MetadataSource) Token(ctx context.Context) (Token, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.endpoint, nil)
	if err != nil {
		return Token{}, err
	}
	req.Header.Set(metadataHeaderKey, metadataHeaderVal)

	resp, err := m.client.Do(req)
	if err != nil {
		return Token{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Token{}, errors.New("metadata token request failed")
	}

	var payload struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
		TokenType   string `json:"token_type"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return Token{}, err
	}
	if payload.AccessToken == "" {
		return Token{}, errors.New("metadata token empty")
	}

	expires := time.Now().UTC().Add(time.Duration(payload.ExpiresIn) * time.Second)
	return Token{Value: payload.AccessToken, ExpiresAt: expires}, nil
}
