package httpclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tamirat-dejene/veritas/shared/pkg/logger"
)

func TestClient_Get(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.Header.Get("X-Request-ID") != "test-rid" {
			t.Errorf("expected X-Request-ID header to be test-rid")
		}
		if r.Header.Get("X-Enterprise-ID") != "test-eid" {
			t.Errorf("expected X-Enterprise-ID header to be test-eid")
		}
		if r.URL.Path != "/test" {
			t.Errorf("expected /test, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"foo": "bar"})
	}))
	defer server.Close()

	cfg := Config{
		BaseURL: server.URL,
		Timeout: 1 * time.Second,
	}
	client := New(cfg)

	ctx := context.Background()
	ctx = logger.SetRequestID(ctx, "test-rid")
	ctx = logger.SetEnterpriseID(ctx, "test-eid")

	resp, err := client.Get(ctx, "/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := resp.Error(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]string
	if err := resp.Decode(&data); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if data["foo"] != "bar" {
		t.Errorf("expected bar, got %s", data["foo"])
	}
}

func TestClient_Post(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json, got %s", r.Header.Get("Content-Type"))
		}

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["hello"] != "world" {
			t.Errorf("expected world, got %s", body["hello"])
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	client := New(Config{BaseURL: server.URL})

	resp, err := client.Post(context.Background(), "/post", map[string]string{"hello": "world"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}

	var data map[string]string
	resp.Decode(&data)
	if data["status"] != "ok" {
		t.Errorf("expected ok, got %s", data["status"])
	}
}
