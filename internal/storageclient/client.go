package storageclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/linkasu/dialog-helper-service/internal/models"
	"github.com/linkasu/dialog-helper-service/internal/storage"
)

var ErrUnauthorized = errors.New("unauthorized")

type Client struct {
	baseURL       string
	internalToken string
	tokenProvider TokenProvider
	httpClient    *http.Client
}

type TokenProvider interface {
	Token(ctx context.Context) (string, error)
}

func New(baseURL, internalToken string, provider TokenProvider) (*Client, error) {
	base := strings.TrimRight(baseURL, "/")
	if base == "" {
		return nil, errors.New("storage base url is required")
	}
	if internalToken == "" {
		return nil, errors.New("internal token is required")
	}

	return &Client{
		baseURL:       base,
		internalToken: internalToken,
		tokenProvider: provider,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

func (c *Client) GetConfig(ctx context.Context, key string) (storage.ConfigItem, bool, error) {
	url := fmt.Sprintf("%s/internal/config/%s", c.baseURL, key)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return storage.ConfigItem{}, false, err
	}
	if err := c.addInternalHeaders(ctx, req); err != nil {
		return storage.ConfigItem{}, false, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return storage.ConfigItem{}, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return storage.ConfigItem{}, false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return storage.ConfigItem{}, false, readHTTPError(resp)
	}

	var item storage.ConfigItem
	if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
		return storage.ConfigItem{}, false, err
	}

	return item, true, nil
}

func (c *Client) GetResult(ctx context.Context, dialogHash string) (storage.ResultRecord, bool, error) {
	url := fmt.Sprintf("%s/internal/cache/%s", c.baseURL, dialogHash)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return storage.ResultRecord{}, false, err
	}
	if err := c.addInternalHeaders(ctx, req); err != nil {
		return storage.ResultRecord{}, false, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return storage.ResultRecord{}, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return storage.ResultRecord{}, false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return storage.ResultRecord{}, false, readHTTPError(resp)
	}

	var record storage.ResultRecord
	if err := json.NewDecoder(resp.Body).Decode(&record); err != nil {
		return storage.ResultRecord{}, false, err
	}

	return record, true, nil
}

func (c *Client) Persist(ctx context.Context, record storage.ResultRecord) error {
	payload, err := json.Marshal(map[string]interface{}{
		"dialog_hash": record.DialogHash,
		"user_hash":   record.UserHash,
		"transcript":  record.Transcript,
		"candidates":  record.Candidates,
		"meta_json":   record.MetaJSON,
	})
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/internal/persist", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	if err := c.addInternalHeaders(ctx, req); err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return readHTTPError(resp)
	}

	return nil
}

func (c *Client) VerifyAPIKey(ctx context.Context, apiKeyHash string) (models.APIKeyInfo, bool, error) {
	payload, err := json.Marshal(map[string]string{"api_key_hash": apiKeyHash})
	if err != nil {
		return models.APIKeyInfo{}, false, err
	}

	url := fmt.Sprintf("%s/internal/keys/verify", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return models.APIKeyInfo{}, false, err
	}
	if err := c.addInternalHeaders(ctx, req); err != nil {
		return models.APIKeyInfo{}, false, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return models.APIKeyInfo{}, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return models.APIKeyInfo{}, false, ErrUnauthorized
	}
	if resp.StatusCode != http.StatusOK {
		return models.APIKeyInfo{}, false, readHTTPError(resp)
	}

	var info models.APIKeyInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return models.APIKeyInfo{}, false, err
	}

	return info, true, nil
}

func (c *Client) addInternalHeaders(ctx context.Context, req *http.Request) error {
	req.Header.Set("X-DH-Internal-Token", c.internalToken)
	req.Header.Set("X-Internal-Token", c.internalToken)
	if c.tokenProvider == nil {
		return nil
	}
	token, err := c.tokenProvider.Token(ctx)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return nil
}

func readHTTPError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	if len(body) == 0 {
		return fmt.Errorf("storage error: status %d", resp.StatusCode)
	}
	return fmt.Errorf("storage error: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
}
