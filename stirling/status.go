package stirling

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Status is the health response of a self-hosted Stirling PDF server. The
// Processing API document does not describe this endpoint, but every
// self-hosted server exposes GET /api/v1/info/status.
type Status struct {
	Version string `json:"version"`
	Status  string `json:"status"`
}

// Status calls GET /api/v1/info/status. It is useful as a readiness check
// before uploading documents.
func (c *Client) Status(ctx context.Context) (*Status, error) {
	res, err := c.CallMethod(ctx, http.MethodGet, "/api/v1/info/status")
	if err != nil {
		return nil, err
	}
	body, err := res.Bytes()
	if err != nil {
		return nil, err
	}
	var status Status
	if err := json.Unmarshal(body, &status); err != nil {
		return nil, fmt.Errorf("stirling: status: decode JSON: %w", err)
	}
	return &status, nil
}
