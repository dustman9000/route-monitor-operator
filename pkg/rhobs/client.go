package rhobs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-logr/logr"
)

// ProbeRequest represents the payload for creating/updating a probe
type ProbeRequest struct {
	ClusterID           string `json:"cluster_id"`
	APIServerURL        string `json:"apiserver_url"`
	ManagementClusterID string `json:"management_cluster_id,omitempty"`
	Private             bool   `json:"private"`
}

// ProbeResponse represents the response from the RHOBS API
type ProbeResponse struct {
	ID        string `json:"id"`
	ClusterID string `json:"cluster_id"`
	Status    string `json:"status"`
}

// ProbesListResponse represents the response from GET probes endpoint
type ProbesListResponse struct {
	Probes []ProbeResponse `json:"probes"`
}

// Client handles communication with the RHOBS synthetics API
type Client struct {
	baseURL    string
	httpClient *http.Client
	tenant     string
	logger     logr.Logger
}

// NewClient creates a new RHOBS API client
func NewClient(baseURL, tenant string, logger logr.Logger) *Client {
	return &Client{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		tenant: tenant,
		logger: logger,
	}
}

// CreateProbe creates a new probe in RHOBS
func (c *Client) CreateProbe(ctx context.Context, req ProbeRequest) (*ProbeResponse, error) {
	url := fmt.Sprintf("%s/%s/metrics/probes", c.baseURL, c.tenant)

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal probe request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	c.logger.V(2).Info("Creating RHOBS probe", "url", url, "cluster_id", req.ClusterID)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var probeResp ProbeResponse
	if err := json.Unmarshal(body, &probeResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &probeResp, nil
}

// GetProbe retrieves a probe by cluster ID
func (c *Client) GetProbe(ctx context.Context, clusterID string) (*ProbeResponse, error) {
	url := fmt.Sprintf("%s/%s/metrics/probes", c.baseURL, c.tenant)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Add query parameter for cluster_id
	q := httpReq.URL.Query()
	q.Add("cluster_id", clusterID)
	httpReq.URL.RawQuery = q.Encode()

	c.logger.V(2).Info("Getting RHOBS probe", "url", httpReq.URL.String(), "cluster_id", clusterID)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // Probe doesn't exist
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var listResp ProbesListResponse
	if err := json.Unmarshal(body, &listResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Find the probe with matching cluster_id
	for _, probe := range listResp.Probes {
		if probe.ClusterID == clusterID {
			return &probe, nil
		}
	}

	return nil, nil // Probe not found
}

// DeleteProbe deletes a probe by cluster ID
func (c *Client) DeleteProbe(ctx context.Context, clusterID string) error {
	url := fmt.Sprintf("%s/%s/metrics/probes/%s", c.baseURL, c.tenant, clusterID)

	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	c.logger.V(2).Info("Deleting RHOBS probe", "url", url, "cluster_id", clusterID)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// Probe already doesn't exist, consider this success
		return nil
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// IsNon200Error checks if an error represents a non-200 HTTP status
func IsNon200Error(err error) bool {
	return err != nil && strings.Contains(err.Error(), "API request failed with status")
}
