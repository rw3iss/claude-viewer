package data

import (
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Usage is a snapshot of one account's rate-limit utilization.
type Usage struct {
	FiveHourPct      int       `json:"five_hour_pct"`
	FiveHourResetAt  time.Time `json:"five_hour_reset_at"`
	SevenDayPct      int       `json:"seven_day_pct"`
	SevenDayResetAt  time.Time `json:"seven_day_reset_at"`
	FetchedAt        time.Time `json:"fetched_at"`
}

// UsageStale reports whether the snapshot is older than ttl.
func (u Usage) Stale(ttl time.Duration) bool {
	return time.Since(u.FetchedAt) > ttl
}

const (
	usageEndpoint = "https://api.anthropic.com/api/oauth/usage"
	usageBeta     = "oauth-2025-04-20"
	usageUA       = "claude-code/2.1.34" // Anthropic API gates on UA
)

// UsageCacheTTL is the soft TTL for cached usage snapshots.
const UsageCacheTTL = 60 * time.Second

// ReadOAuthToken returns the bearer token from <claudeDir>/.credentials.json.
// Returns an error wrapping fs.ErrNotExist if the file is missing.
func ReadOAuthToken(claudeDir string) (string, error) {
	p := filepath.Join(claudeDir, ".credentials.json")
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()
	var creds struct {
		ClaudeAiOauth struct {
			AccessToken string `json:"accessToken"`
		} `json:"claudeAiOauth"`
	}
	if err := json.NewDecoder(f).Decode(&creds); err != nil {
		return "", fmt.Errorf("decode %s: %w", p, err)
	}
	if creds.ClaudeAiOauth.AccessToken == "" {
		return "", errors.New("no accessToken in " + p)
	}
	return creds.ClaudeAiOauth.AccessToken, nil
}

// FetchUsage hits the Anthropic OAuth usage endpoint and returns the parsed
// Usage. 10s timeout; non-200 returns an error with status + body.
func FetchUsage(token string) (*Usage, error) {
	req, err := http.NewRequest("GET", usageEndpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("anthropic-beta", usageBeta)
	req.Header.Set("User-Agent", usageUA)

	cli := &http.Client{Timeout: 10 * time.Second}
	res, err := cli.Do(req)
	if err != nil {
		return nil, fmt.Errorf("usage api: %w", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("usage api %d: %s", res.StatusCode, body)
	}
	var resp struct {
		FiveHour struct {
			Utilization float64 `json:"utilization"`
			ResetsAt    string  `json:"resets_at"`
		} `json:"five_hour"`
		SevenDay struct {
			Utilization float64 `json:"utilization"`
			ResetsAt    string  `json:"resets_at"`
		} `json:"seven_day"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode usage: %w", err)
	}
	fiveResetAt, _ := time.Parse(time.RFC3339Nano, resp.FiveHour.ResetsAt)
	if fiveResetAt.IsZero() {
		fiveResetAt, _ = time.Parse(time.RFC3339, resp.FiveHour.ResetsAt)
	}
	sevenResetAt, _ := time.Parse(time.RFC3339Nano, resp.SevenDay.ResetsAt)
	if sevenResetAt.IsZero() {
		sevenResetAt, _ = time.Parse(time.RFC3339, resp.SevenDay.ResetsAt)
	}
	return &Usage{
		FiveHourPct:     int(resp.FiveHour.Utilization),
		FiveHourResetAt: fiveResetAt,
		SevenDayPct:     int(resp.SevenDay.Utilization),
		SevenDayResetAt: sevenResetAt,
		FetchedAt:       time.Now(),
	}, nil
}

// usageCachePath returns the on-disk cache path for a given OAuth token.
// Per-token (not per-dir) so two configs sharing one account share a cache.
func usageCachePath(cacheRoot, token string) string {
	h := fnv.New64a()
	h.Write([]byte(token))
	return filepath.Join(cacheRoot, "usage-"+toHex(h.Sum64())+".json")
}

// ReadCachedUsage returns a previously-cached Usage for token, or fs.ErrNotExist.
func ReadCachedUsage(cacheRoot, token string) (*Usage, error) {
	p := usageCachePath(cacheRoot, token)
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var u Usage
	if err := json.Unmarshal(data, &u); err != nil {
		return nil, err
	}
	return &u, nil
}

// WriteCachedUsage atomically writes a Usage snapshot.
func WriteCachedUsage(cacheRoot, token string, u *Usage) error {
	p := usageCachePath(cacheRoot, token)
	tmp, err := os.CreateTemp(cacheRoot, "usage-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := json.NewEncoder(tmp).Encode(u); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), p)
}
