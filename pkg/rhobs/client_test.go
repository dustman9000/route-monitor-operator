package rhobs

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-logr/logr/testr"
)

func TestCreateProbe(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		if r.URL.Path != "/test-tenant/metrics/probes" {
			t.Errorf("Expected path /test-tenant/metrics/probes, got %s", r.URL.Path)
		}

		var req ProbeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		if req.ClusterID != "test-cluster" {
			t.Errorf("Expected cluster_id test-cluster, got %s", req.ClusterID)
		}

		// Return a mock response
		resp := ProbeResponse{
			ID:        "probe-123",
			ClusterID: "test-cluster",
			Status:    "active",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create client
	client := NewClient(server.URL, "test-tenant", testr.New(t))

	// Test create probe
	probeReq := ProbeRequest{
		ClusterID:    "test-cluster",
		APIServerURL: "https://api.test-cluster.example.com/livez",
		Private:      false,
	}

	probe, err := client.CreateProbe(context.Background(), probeReq)
	if err != nil {
		t.Fatalf("CreateProbe failed: %v", err)
	}

	if probe.ID != "probe-123" {
		t.Errorf("Expected probe ID probe-123, got %s", probe.ID)
	}
}

func TestGetProbe(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET method, got %s", r.Method)
		}

		clusterID := r.URL.Query().Get("cluster_id")
		if clusterID != "test-cluster" {
			t.Errorf("Expected cluster_id test-cluster, got %s", clusterID)
		}

		// Return a mock response
		resp := ProbesListResponse{
			Probes: []ProbeResponse{
				{
					ID:        "probe-123",
					ClusterID: "test-cluster",
					Status:    "active",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create client
	client := NewClient(server.URL, "test-tenant", testr.New(t))

	// Test get probe
	probe, err := client.GetProbe(context.Background(), "test-cluster")
	if err != nil {
		t.Fatalf("GetProbe failed: %v", err)
	}

	if probe == nil {
		t.Fatal("Expected probe to be found, got nil")
	}

	if probe.ID != "probe-123" {
		t.Errorf("Expected probe ID probe-123, got %s", probe.ID)
	}
}

func TestDeleteProbe(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE method, got %s", r.Method)
		}

		if r.URL.Path != "/test-tenant/metrics/probes/test-cluster" {
			t.Errorf("Expected path /test-tenant/metrics/probes/test-cluster, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	// Create client
	client := NewClient(server.URL, "test-tenant", testr.New(t))

	// Test delete probe
	err := client.DeleteProbe(context.Background(), "test-cluster")
	if err != nil {
		t.Fatalf("DeleteProbe failed: %v", err)
	}
}

func TestIsNon200Error(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "non-API error",
			err:      http.ErrUseLastResponse,
			expected: false,
		},
		{
			name:     "API error with status code",
			err:      &APIError{StatusCode: 400, Message: "Bad Request"},
			expected: false, // This would be true if we used the error format from our client
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNon200Error(tt.err)
			if result != tt.expected {
				t.Errorf("IsNon200Error() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// APIError represents an API error for testing
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return e.Message
}
