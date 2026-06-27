package scoring

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/ruaan-deysel/unraid-management-agent/daemon/domain"
	"github.com/ruaan-deysel/unraid-management-agent/daemon/logger"
)

// Client posts scores to the Langfuse Scores API.
type Client struct {
	base string
	auth string
	http *http.Client
}

// NewClient returns nil when Langfuse is disabled (callers must nil-check).
func NewClient(cfg domain.Config) *Client {
	if !cfg.LangfuseEnabled() {
		return nil
	}
	return &Client{
		base: cfg.LangfuseBaseURL,
		auth: "Basic " + base64.StdEncoding.EncodeToString([]byte(cfg.LangfusePublicKey+":"+cfg.LangfuseSecretKey)),
		http: &http.Client{Timeout: 5 * time.Second},
	}
}

// Post sends all scores for a trace to POST <base>/api/public/scores.
// Best-effort: errors are logged at debug level and never bubble up.
// A zero/empty trace ID (no recording tracer) is skipped silently.
func (c *Client) Post(ctx context.Context, traceID string, s Scores) {
	if c == nil || traceID == "" || traceID == "00000000000000000000000000000000" {
		return
	}
	for name, val := range map[string]bool{
		"no_hallucinated_tool": s.NoHallucinatedTool,
		"no_unconfirmed_write": s.NoUnconfirmedWrite,
		"read_only_respected":  s.ReadOnlyRespected,
	} {
		body, _ := json.Marshal(map[string]any{
			"traceId":  traceID,
			"name":     name,
			"dataType": "BOOLEAN",
			"value":    boolToFloat(val),
		})
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/api/public/scores", bytes.NewReader(body))
		if err != nil {
			logger.Debug("Langfuse score build failed: %v", err)
			continue
		}
		req.Header.Set("Authorization", c.auth)
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.http.Do(req)
		if err != nil {
			logger.Debug("Langfuse score post failed: %v", err)
			continue
		}
		if resp.StatusCode >= 300 {
			logger.Debug("Langfuse score post returned status %d", resp.StatusCode)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		if closeErr := resp.Body.Close(); closeErr != nil {
			logger.Debug("Langfuse score body close failed: %v", closeErr)
		}
	}
}

func boolToFloat(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
