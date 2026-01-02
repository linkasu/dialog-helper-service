package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/linkasu/dialog-helper-service/internal/auth"
	"github.com/linkasu/dialog-helper-service/internal/models"
	"github.com/linkasu/dialog-helper-service/internal/ydb"

	sdk "github.com/ydb-platform/ydb-go-sdk/v3"
	"github.com/ydb-platform/ydb-go-sdk/v3/table"
	"github.com/ydb-platform/ydb-go-sdk/v3/table/result/named"
	"github.com/ydb-platform/ydb-go-sdk/v3/table/types"
)

type YDBStore struct {
	db *sdk.Driver
}

func NewYDBStore(ctx context.Context, dsn string, provider ydb.TokenProvider) (*YDBStore, error) {
	driver, err := ydb.Open(ctx, dsn, provider)
	if err != nil {
		return nil, err
	}
	return &YDBStore{db: driver}, nil
}

func (s *YDBStore) Close(ctx context.Context) error {
	if s.db == nil {
		return nil
	}
	return s.db.Close(ctx)
}

func (s *YDBStore) GetConfig(ctx context.Context, key string) (ConfigItem, bool, error) {
	var item ConfigItem
	var found bool

	tablePath := s.tablePath("config")
	query := "DECLARE $key AS Utf8;\n" +
		"SELECT config_key, config_value, updated_at " +
		"FROM `" + tablePath + "` WHERE config_key = $key;"

	err := s.db.Table().Do(ctx, func(ctx context.Context, session table.Session) error {
		params := table.NewQueryParameters(table.ValueParam("$key", types.TextValue(key)))
		_, res, err := session.Execute(ctx, table.OnlineReadOnlyTxControl(), query, params)
		if err != nil {
			return err
		}
		defer res.Close()
		if err := res.NextResultSetErr(ctx); err != nil {
			return err
		}
		if !res.NextRow() {
			found = false
			return res.Err()
		}
		found = true
		if err := res.ScanNamed(
			named.OptionalWithDefault("config_key", &item.Key),
			named.OptionalWithDefault("config_value", &item.Value),
			named.OptionalWithDefault("updated_at", &item.UpdatedAt),
		); err != nil {
			return err
		}
		return res.Err()
	}, table.WithIdempotent())

	if err != nil {
		return ConfigItem{}, false, err
	}

	return item, found, nil
}

func (s *YDBStore) SetConfig(ctx context.Context, key, value string) error {
	tablePath := s.tablePath("config")
	query := "DECLARE $key AS Utf8;\n" +
		"DECLARE $value AS Utf8;\n" +
		"DECLARE $updated_at AS Timestamp;\n" +
		"UPSERT INTO `" + tablePath + "` (config_key, config_value, updated_at) VALUES ($key, $value, $updated_at);"

	return s.db.Table().Do(ctx, func(ctx context.Context, session table.Session) error {
		params := table.NewQueryParameters(
			table.ValueParam("$key", types.TextValue(key)),
			table.ValueParam("$value", types.TextValue(value)),
			table.ValueParam("$updated_at", types.TimestampValueFromTime(time.Now().UTC())),
		)
		_, _, err := session.Execute(ctx, table.DefaultTxControl(), query, params)
		return err
	}, table.WithIdempotent())
}

func (s *YDBStore) GetResult(ctx context.Context, dialogHash string) (ResultRecord, bool, error) {
	var record ResultRecord
	var candidatesJSON string
	var metaJSON string
	var found bool

	tablePath := s.tablePath("results")
	query := "DECLARE $dialog_hash AS Utf8;\n" +
		"SELECT dialog_hash, user_hash, stt_transcript, answer_candidates_json, meta_json, created_at " +
		"FROM `" + tablePath + "` WHERE dialog_hash = $dialog_hash;"

	err := s.db.Table().Do(ctx, func(ctx context.Context, session table.Session) error {
		params := table.NewQueryParameters(table.ValueParam("$dialog_hash", types.TextValue(dialogHash)))
		_, res, err := session.Execute(ctx, table.OnlineReadOnlyTxControl(), query, params)
		if err != nil {
			return err
		}
		defer res.Close()
		if err := res.NextResultSetErr(ctx); err != nil {
			return err
		}
		if !res.NextRow() {
			found = false
			return res.Err()
		}
		found = true
		if err := res.ScanNamed(
			named.OptionalWithDefault("dialog_hash", &record.DialogHash),
			named.OptionalWithDefault("user_hash", &record.UserHash),
			named.OptionalWithDefault("stt_transcript", &record.Transcript),
			named.OptionalWithDefault("answer_candidates_json", &candidatesJSON),
			named.OptionalWithDefault("meta_json", &metaJSON),
			named.OptionalWithDefault("created_at", &record.CreatedAt),
		); err != nil {
			return err
		}
		return res.Err()
	}, table.WithIdempotent())

	if err != nil {
		return ResultRecord{}, false, err
	}

	if !found {
		return ResultRecord{}, false, nil
	}

	if candidatesJSON != "" {
		var candidates []models.AnswerCandidate
		if err := json.Unmarshal([]byte(candidatesJSON), &candidates); err != nil {
			return ResultRecord{}, false, fmt.Errorf("decode candidates: %w", err)
		}
		record.Candidates = candidates
	}
	record.MetaJSON = metaJSON
	return record, true, nil
}

func (s *YDBStore) PutResult(ctx context.Context, record ResultRecord) error {
	candidatesJSON := ""
	if len(record.Candidates) > 0 {
		payload, err := json.Marshal(record.Candidates)
		if err != nil {
			return fmt.Errorf("encode candidates: %w", err)
		}
		candidatesJSON = string(payload)
	}

	tablePath := s.tablePath("results")
	query := "DECLARE $dialog_hash AS Utf8;\n" +
		"DECLARE $user_hash AS Utf8;\n" +
		"DECLARE $stt_transcript AS Utf8;\n" +
		"DECLARE $answer_candidates_json AS Utf8;\n" +
		"DECLARE $meta_json AS Utf8;\n" +
		"DECLARE $created_at AS Timestamp;\n" +
		"UPSERT INTO `" + tablePath + "` (dialog_hash, user_hash, stt_transcript, answer_candidates_json, meta_json, created_at) " +
		"VALUES ($dialog_hash, $user_hash, $stt_transcript, $answer_candidates_json, $meta_json, $created_at);"

	return s.db.Table().Do(ctx, func(ctx context.Context, session table.Session) error {
		createdAt := record.CreatedAt
		if createdAt.IsZero() {
			createdAt = time.Now().UTC()
		}
		params := table.NewQueryParameters(
			table.ValueParam("$dialog_hash", types.TextValue(record.DialogHash)),
			table.ValueParam("$user_hash", types.TextValue(record.UserHash)),
			table.ValueParam("$stt_transcript", types.TextValue(record.Transcript)),
			table.ValueParam("$answer_candidates_json", types.TextValue(candidatesJSON)),
			table.ValueParam("$meta_json", types.TextValue(record.MetaJSON)),
			table.ValueParam("$created_at", types.TimestampValueFromTime(createdAt)),
		)
		_, _, err := session.Execute(ctx, table.DefaultTxControl(), query, params)
		return err
	}, table.WithIdempotent())
}

func (s *YDBStore) ListResults(ctx context.Context, opts ResultListOptions) ([]ResultRecord, error) {
	since := opts.Since
	if since.IsZero() {
		since = time.Unix(0, 0).UTC()
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = 100
	}

	tablePath := s.tablePath("results")
	query := "DECLARE $since AS Timestamp;\n" +
		"DECLARE $limit AS Uint64;\n" +
		"SELECT dialog_hash, user_hash, stt_transcript, answer_candidates_json, meta_json, created_at " +
		"FROM `" + tablePath + "` WHERE created_at >= $since ORDER BY created_at DESC LIMIT $limit;"

	records := make([]ResultRecord, 0, limit)
	err := s.db.Table().Do(ctx, func(ctx context.Context, session table.Session) error {
		params := table.NewQueryParameters(
			table.ValueParam("$since", types.TimestampValueFromTime(since)),
			table.ValueParam("$limit", types.Uint64Value(uint64(limit))),
		)
		_, res, err := session.Execute(ctx, table.OnlineReadOnlyTxControl(), query, params)
		if err != nil {
			return err
		}
		defer res.Close()
		if err := res.NextResultSetErr(ctx); err != nil {
			return err
		}
		for res.NextRow() {
			var record ResultRecord
			var candidatesJSON string
			var metaJSON string
			if err := res.ScanNamed(
				named.OptionalWithDefault("dialog_hash", &record.DialogHash),
				named.OptionalWithDefault("user_hash", &record.UserHash),
				named.OptionalWithDefault("stt_transcript", &record.Transcript),
				named.OptionalWithDefault("answer_candidates_json", &candidatesJSON),
				named.OptionalWithDefault("meta_json", &metaJSON),
				named.OptionalWithDefault("created_at", &record.CreatedAt),
			); err != nil {
				return err
			}
			if candidatesJSON != "" {
				var candidates []models.AnswerCandidate
				if err := json.Unmarshal([]byte(candidatesJSON), &candidates); err != nil {
					return fmt.Errorf("decode candidates: %w", err)
				}
				record.Candidates = candidates
			}
			record.MetaJSON = metaJSON
			records = append(records, record)
		}
		return res.Err()
	}, table.WithIdempotent())

	if err != nil {
		return nil, err
	}
	return records, nil
}

func (s *YDBStore) CountResults(ctx context.Context, since time.Time) (int, error) {
	if since.IsZero() {
		since = time.Unix(0, 0).UTC()
	}
	tablePath := s.tablePath("results")
	query := "DECLARE $since AS Timestamp;\n" +
		"SELECT COUNT(*) AS total FROM `" + tablePath + "` WHERE created_at >= $since;"

	var total uint64
	err := s.db.Table().Do(ctx, func(ctx context.Context, session table.Session) error {
		params := table.NewQueryParameters(
			table.ValueParam("$since", types.TimestampValueFromTime(since)),
		)
		_, res, err := session.Execute(ctx, table.OnlineReadOnlyTxControl(), query, params)
		if err != nil {
			return err
		}
		defer res.Close()
		if err := res.NextResultSetErr(ctx); err != nil {
			return err
		}
		if !res.NextRow() {
			return res.Err()
		}
		if err := res.ScanNamed(
			named.OptionalWithDefault("total", &total),
		); err != nil {
			return err
		}
		return res.Err()
	}, table.WithIdempotent())
	if err != nil {
		return 0, err
	}
	return int(total), nil
}

func (s *YDBStore) CountActiveUsers(ctx context.Context, since time.Time) (int, error) {
	if since.IsZero() {
		since = time.Unix(0, 0).UTC()
	}
	tablePath := s.tablePath("results")
	query := "DECLARE $since AS Timestamp;\n" +
		"SELECT COUNT(DISTINCT user_hash) AS total FROM `" + tablePath + "` WHERE created_at >= $since;"

	var total uint64
	err := s.db.Table().Do(ctx, func(ctx context.Context, session table.Session) error {
		params := table.NewQueryParameters(
			table.ValueParam("$since", types.TimestampValueFromTime(since)),
		)
		_, res, err := session.Execute(ctx, table.OnlineReadOnlyTxControl(), query, params)
		if err != nil {
			return err
		}
		defer res.Close()
		if err := res.NextResultSetErr(ctx); err != nil {
			return err
		}
		if !res.NextRow() {
			return res.Err()
		}
		if err := res.ScanNamed(
			named.OptionalWithDefault("total", &total),
		); err != nil {
			return err
		}
		return res.Err()
	}, table.WithIdempotent())
	if err != nil {
		return 0, err
	}
	return int(total), nil
}

func (s *YDBStore) GetAPIKey(ctx context.Context, apiKeyHash string) (APIKeyRecord, bool, error) {
	var record APIKeyRecord
	var scopesJSON string
	var revokedAt *time.Time
	var found bool

	tablePath := s.tablePath("api_keys")
	query := "DECLARE $api_key_hash AS Utf8;\n" +
		"SELECT api_key_hash, client_id, status, scopes_json, rate_limit_per_minute, created_at, revoked_at " +
		"FROM `" + tablePath + "` WHERE api_key_hash = $api_key_hash;"

	err := s.db.Table().Do(ctx, func(ctx context.Context, session table.Session) error {
		params := table.NewQueryParameters(table.ValueParam("$api_key_hash", types.TextValue(apiKeyHash)))
		_, res, err := session.Execute(ctx, table.OnlineReadOnlyTxControl(), query, params)
		if err != nil {
			return err
		}
		defer res.Close()
		if err := res.NextResultSetErr(ctx); err != nil {
			return err
		}
		if !res.NextRow() {
			found = false
			return res.Err()
		}
		found = true
		if err := res.ScanNamed(
			named.OptionalWithDefault("api_key_hash", &record.APIKeyHash),
			named.OptionalWithDefault("client_id", &record.ClientID),
			named.OptionalWithDefault("status", &record.Status),
			named.OptionalWithDefault("scopes_json", &scopesJSON),
			named.OptionalWithDefault("rate_limit_per_minute", &record.RateLimitPerMinute),
			named.OptionalWithDefault("created_at", &record.CreatedAt),
			named.Optional("revoked_at", &revokedAt),
		); err != nil {
			return err
		}
		return res.Err()
	}, table.WithIdempotent())

	if err != nil {
		return APIKeyRecord{}, false, err
	}
	if !found {
		return APIKeyRecord{}, false, nil
	}
	if scopesJSON != "" {
		var scopes []string
		if err := json.Unmarshal([]byte(scopesJSON), &scopes); err != nil {
			return APIKeyRecord{}, false, fmt.Errorf("decode scopes: %w", err)
		}
		record.Scopes = scopes
	}
	record.RevokedAt = revokedAt
	return record, true, nil
}

func (s *YDBStore) CreateAPIKey(ctx context.Context, record APIKeyRecord) error {
	scopesJSON := ""
	if len(record.Scopes) > 0 {
		payload, err := json.Marshal(record.Scopes)
		if err != nil {
			return fmt.Errorf("encode scopes: %w", err)
		}
		scopesJSON = string(payload)
	}

	tablePath := s.tablePath("api_keys")
	query := "DECLARE $api_key_hash AS Utf8;\n" +
		"DECLARE $client_id AS Utf8;\n" +
		"DECLARE $status AS Utf8;\n" +
		"DECLARE $scopes_json AS Utf8;\n" +
		"DECLARE $rate_limit_per_minute AS Int32;\n" +
		"DECLARE $created_at AS Timestamp;\n" +
		"DECLARE $revoked_at AS Timestamp?;\n" +
		"UPSERT INTO `" + tablePath + "` (api_key_hash, client_id, status, scopes_json, rate_limit_per_minute, created_at, revoked_at) " +
		"VALUES ($api_key_hash, $client_id, $status, $scopes_json, $rate_limit_per_minute, $created_at, $revoked_at);"

	return s.db.Table().Do(ctx, func(ctx context.Context, session table.Session) error {
		createdAt := record.CreatedAt
		if createdAt.IsZero() {
			createdAt = time.Now().UTC()
		}
		params := table.NewQueryParameters(
			table.ValueParam("$api_key_hash", types.TextValue(record.APIKeyHash)),
			table.ValueParam("$client_id", types.TextValue(record.ClientID)),
			table.ValueParam("$status", types.TextValue(record.Status)),
			table.ValueParam("$scopes_json", types.TextValue(scopesJSON)),
			table.ValueParam("$rate_limit_per_minute", types.Int32Value(int32(record.RateLimitPerMinute))),
			table.ValueParam("$created_at", types.TimestampValueFromTime(createdAt)),
			table.ValueParam("$revoked_at", nullableTimestamp(record.RevokedAt)),
		)
		_, _, err := session.Execute(ctx, table.DefaultTxControl(), query, params)
		return err
	}, table.WithIdempotent())
}

func (s *YDBStore) RevokeAPIKey(ctx context.Context, apiKeyHash string, revokedAt time.Time) error {
	tablePath := s.tablePath("api_keys")
	query := "DECLARE $api_key_hash AS Utf8;\n" +
		"DECLARE $status AS Utf8;\n" +
		"DECLARE $revoked_at AS Timestamp;\n" +
		"UPSERT INTO `" + tablePath + "` (api_key_hash, status, revoked_at) VALUES ($api_key_hash, $status, $revoked_at);"

	return s.db.Table().Do(ctx, func(ctx context.Context, session table.Session) error {
		params := table.NewQueryParameters(
			table.ValueParam("$api_key_hash", types.TextValue(apiKeyHash)),
			table.ValueParam("$status", types.TextValue(auth.StatusRevoked)),
			table.ValueParam("$revoked_at", types.TimestampValueFromTime(revokedAt.UTC())),
		)
		_, _, err := session.Execute(ctx, table.DefaultTxControl(), query, params)
		return err
	}, table.WithIdempotent())
}

func (s *YDBStore) ListAPIKeys(ctx context.Context, opts APIKeyListOptions) ([]APIKeyRecord, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 100
	}

	tablePath := s.tablePath("api_keys")
	query := "DECLARE $limit AS Uint64;\n"
	if !opts.IncludeRevoked {
		query += "DECLARE $revoked AS Utf8;\n"
	}
	query += "SELECT api_key_hash, client_id, status, scopes_json, rate_limit_per_minute, created_at, revoked_at " +
		"FROM `" + tablePath + "` "
	if !opts.IncludeRevoked {
		query += "WHERE status != $revoked "
	}
	query += "ORDER BY created_at DESC LIMIT $limit;"

	records := make([]APIKeyRecord, 0, limit)
	err := s.db.Table().Do(ctx, func(ctx context.Context, session table.Session) error {
		var params *table.QueryParameters
		if opts.IncludeRevoked {
			params = table.NewQueryParameters(
				table.ValueParam("$limit", types.Uint64Value(uint64(limit))),
			)
		} else {
			params = table.NewQueryParameters(
				table.ValueParam("$limit", types.Uint64Value(uint64(limit))),
				table.ValueParam("$revoked", types.TextValue(auth.StatusRevoked)),
			)
		}

		_, res, err := session.Execute(ctx, table.OnlineReadOnlyTxControl(), query, params)
		if err != nil {
			return err
		}
		defer res.Close()
		if err := res.NextResultSetErr(ctx); err != nil {
			return err
		}
		for res.NextRow() {
			var record APIKeyRecord
			var scopesJSON string
			var revokedAt *time.Time
			if err := res.ScanNamed(
				named.OptionalWithDefault("api_key_hash", &record.APIKeyHash),
				named.OptionalWithDefault("client_id", &record.ClientID),
				named.OptionalWithDefault("status", &record.Status),
				named.OptionalWithDefault("scopes_json", &scopesJSON),
				named.OptionalWithDefault("rate_limit_per_minute", &record.RateLimitPerMinute),
				named.OptionalWithDefault("created_at", &record.CreatedAt),
				named.Optional("revoked_at", &revokedAt),
			); err != nil {
				return err
			}
			if scopesJSON != "" {
				var scopes []string
				if err := json.Unmarshal([]byte(scopesJSON), &scopes); err != nil {
					return fmt.Errorf("decode scopes: %w", err)
				}
				record.Scopes = scopes
			}
			record.RevokedAt = revokedAt
			records = append(records, record)
		}
		return res.Err()
	}, table.WithIdempotent())
	if err != nil {
		return nil, err
	}
	return records, nil
}

func nullableTimestamp(value *time.Time) types.Value {
	if value == nil {
		return types.NullValue(types.TypeTimestamp)
	}
	return types.TimestampValueFromTime(value.UTC())
}
func (s *YDBStore) tablePath(name string) string {
	if s.db == nil {
		return name
	}
	base := strings.TrimRight(s.db.Name(), "/")
	if base == "" {
		return name
	}
	return base + "/" + name
}
