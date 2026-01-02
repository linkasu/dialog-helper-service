package cloudlogging

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	logging "github.com/yandex-cloud/go-genproto/yandex/cloud/logging/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const defaultEndpoint = "logging.api.cloud.yandex.net:443"

var errMissingLogGroup = errors.New("log group id is required")

type TokenProvider interface {
	Token(ctx context.Context) (string, error)
}

type Config struct {
	Endpoint         string
	LogGroupID       string
	ResourceTypes    []string
	ResourceIDs      []string
	StreamNames      []string
	Levels           []string
	Filter           string
	MaxResponseBytes int64
}

type ReadOptions struct {
	Since time.Time
	Until time.Time
	Limit int
}

type Entry struct {
	Timestamp    time.Time `json:"timestamp"`
	Level        string    `json:"level"`
	Message      string    `json:"message"`
	ResourceType string    `json:"resource_type,omitempty"`
	ResourceID   string    `json:"resource_id,omitempty"`
	StreamName   string    `json:"stream_name,omitempty"`
}

type Reader interface {
	Read(ctx context.Context, opts ReadOptions) ([]Entry, error)
}

type Client struct {
	cfg      Config
	levels   []logging.LogLevel_Level
	provider TokenProvider
	conn     *grpc.ClientConn
	svc      logging.LogReadingServiceClient
}

func NewClient(cfg Config, provider TokenProvider) (*Client, error) {
	if strings.TrimSpace(cfg.LogGroupID) == "" {
		return nil, errMissingLogGroup
	}
	if provider == nil {
		return nil, errors.New("token provider is required")
	}

	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		endpoint = defaultEndpoint
	}

	conn, err := grpc.NewClient(endpoint, grpc.WithTransportCredentials(credentials.NewTLS(nil)))
	if err != nil {
		return nil, fmt.Errorf("dial logging service: %w", err)
	}

	levels, err := parseLevels(cfg.Levels)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	return &Client{
		cfg:      cfg,
		levels:   levels,
		provider: provider,
		conn:     conn,
		svc:      logging.NewLogReadingServiceClient(conn),
	}, nil
}

func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *Client) Read(ctx context.Context, opts ReadOptions) ([]Entry, error) {
	if c == nil {
		return nil, errors.New("log reader is nil")
	}

	token, err := c.provider.Token(ctx)
	if err != nil {
		return nil, fmt.Errorf("iam token: %w", err)
	}
	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)

	limit := opts.Limit
	if limit <= 0 {
		limit = 100
	}

	until := opts.Until
	if until.IsZero() {
		until = time.Now().UTC()
	}

	criteria := &logging.Criteria{
		LogGroupId:    c.cfg.LogGroupID,
		ResourceTypes: c.cfg.ResourceTypes,
		ResourceIds:   c.cfg.ResourceIDs,
		StreamNames:   c.cfg.StreamNames,
		Levels:        c.levels,
		Filter:        c.cfg.Filter,
		PageSize:      int64(limit),
		Until:         timestamppb.New(until.UTC()),
	}
	if !opts.Since.IsZero() {
		criteria.Since = timestamppb.New(opts.Since.UTC())
	}
	if c.cfg.MaxResponseBytes > 0 {
		criteria.MaxResponseSize = c.cfg.MaxResponseBytes
	}

	resp, err := c.svc.Read(ctx, &logging.ReadRequest{
		Selector: &logging.ReadRequest_Criteria{Criteria: criteria},
	})
	if err != nil {
		return nil, fmt.Errorf("read logs: %w", err)
	}

	entries := make([]Entry, 0, len(resp.Entries))
	for _, item := range resp.Entries {
		entry := Entry{
			Level:   item.GetLevel().String(),
			Message: strings.TrimSpace(item.GetMessage()),
		}
		if ts := item.GetTimestamp(); ts != nil {
			entry.Timestamp = ts.AsTime()
		}
		if res := item.GetResource(); res != nil {
			entry.ResourceType = res.GetType()
			entry.ResourceID = res.GetId()
		}
		entry.StreamName = item.GetStreamName()
		entries = append(entries, entry)
	}

	return entries, nil
}

func parseLevels(values []string) ([]logging.LogLevel_Level, error) {
	if len(values) == 0 {
		return nil, nil
	}

	levels := make([]logging.LogLevel_Level, 0, len(values))
	for _, raw := range values {
		cleaned := strings.ToLower(strings.TrimSpace(raw))
		if cleaned == "" {
			continue
		}
		level, ok := levelFromString(cleaned)
		if !ok {
			return nil, fmt.Errorf("unsupported log level: %s", raw)
		}
		levels = append(levels, level)
	}

	return levels, nil
}

func levelFromString(value string) (logging.LogLevel_Level, bool) {
	switch strings.ToLower(value) {
	case "trace":
		return logging.LogLevel_TRACE, true
	case "debug":
		return logging.LogLevel_DEBUG, true
	case "info":
		return logging.LogLevel_INFO, true
	case "warn", "warning":
		return logging.LogLevel_WARN, true
	case "error":
		return logging.LogLevel_ERROR, true
	case "fatal":
		return logging.LogLevel_FATAL, true
	default:
		return logging.LogLevel_LEVEL_UNSPECIFIED, false
	}
}
