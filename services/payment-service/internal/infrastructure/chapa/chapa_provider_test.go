package chapa

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/domain"
)

func TestChapaProvider_CreateCheckoutSession_CurrencyEnforcement(t *testing.T) {
	provider := NewChapaProvider("secret-key", "webhook-secret", "http://return", "http://callback")

	// 1. Non-ETB currency should fail
	reqUSD := domain.CheckoutRequest{
		EnterpriseID: uuid.New(),
		Plan: &domain.SubscriptionPlan{
			Currency: domain.CurrencyUSD,
			Price:    10.0,
		},
	}
	_, err := provider.CreateCheckoutSession(context.Background(), reqUSD)
	if err == nil {
		t.Error("expected error for non-ETB currency, got nil")
	}

	// 2. ETB currency should succeed if mock server responds correctly
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := initializeResponse{
			Message: "Hosted Link",
			Status:  "success",
		}
		resp.Data.CheckoutURL = "https://checkout.chapa.co/checkout/payment/123456"
		json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	// Temporarily override base URL / init path or just use custom http client to mock the URL.
	// We can cast provider to our concrete struct to inject mocked httpClient.
	cp := provider.(*chapaProvider)
	cp.httpClient = mockServer.Client()

	// Override base path inside client by altering the base URL.
	// Since chapaBaseURL is const, we can modify the host in the client's transport or use custom roundtripper.
	oldRoundTripper := cp.httpClient.Transport
	cp.httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		// Rewrite host to point to mockServer
		req.URL.Scheme = "http"
		req.URL.Host = mockServer.URL[7:] // remove http://
		return http.DefaultTransport.RoundTrip(req)
	})
	defer func() {
		cp.httpClient.Transport = oldRoundTripper
	}()

	reqETB := domain.CheckoutRequest{
		EnterpriseID: uuid.New(),
		Plan: &domain.SubscriptionPlan{
			ID:       uuid.New(),
			Name:     "Test Plan",
			Currency: domain.CurrencyETB,
			Price:    100.0,
		},
		TxRef: "test-tx-ref",
	}

	url, err := provider.CreateCheckoutSession(context.Background(), reqETB)
	if err != nil {
		t.Fatalf("unexpected error creating checkout session: %v", err)
	}

	expectedURL := "https://checkout.chapa.co/checkout/payment/123456"
	if url != expectedURL {
		t.Errorf("expected checkout URL %s, got %s", expectedURL, url)
	}
}

func TestChapaProvider_VerifyWebhookEvent(t *testing.T) {
	webhookSecret := "my-webhook-secret"
	provider := NewChapaProvider("secret-key", webhookSecret, "http://return", "http://callback")
	cp := provider.(*chapaProvider)

	// Mock verification endpoint
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := verifyResponse{
			Message: "Verification successful",
			Status:  "success",
			Data: map[string]any{
				"tx_ref": "tx-123",
				"meta": map[string]any{
					"enterprise_id": "00000000-0000-0000-0000-000000000001",
					"plan_id":       "00000000-0000-0000-0000-000000000002",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	cp.httpClient = mockServer.Client()
	cp.httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Scheme = "http"
		req.URL.Host = mockServer.URL[7:]
		return http.DefaultTransport.RoundTrip(req)
	})

	whPayload := chapaWebhookPayload{
		Event:         "charge.success",
		TxRef:         "tx-123",
		Status:        "success",
		Amount:        250.50,
		Currency:      "ETB",
		Email:         "test@example.com",
		PaymentMethod: "telebirr",
	}

	payloadBytes, _ := json.Marshal(whPayload)

	// Generate HMAC signature
	mac := hmac.New(sha256.New, []byte(webhookSecret))
	mac.Write(payloadBytes)
	sigHeader := hex.EncodeToString(mac.Sum(nil))

	// 1. Invalid signature should fail
	_, err := provider.VerifyWebhookEvent(payloadBytes, "invalid-sig")
	if err == nil {
		t.Error("expected error with invalid signature, got nil")
	}

	// 2. Valid signature and verification success
	pe, err := provider.VerifyWebhookEvent(payloadBytes, sigHeader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pe.EventType != "payment.success" {
		t.Errorf("expected EventType 'payment.success', got '%s'", pe.EventType)
	}
	if pe.TxRef != "tx-123" {
		t.Errorf("expected TxRef 'tx-123', got '%s'", pe.TxRef)
	}
	if pe.Amount != 250.50 {
		t.Errorf("expected Amount 250.50, got %f", pe.Amount)
	}
	if pe.Currency != domain.CurrencyETB {
		t.Errorf("expected Currency ETB, got %s", pe.Currency)
	}

	expectedEnterpriseID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	if pe.EnterpriseID != expectedEnterpriseID {
		t.Errorf("expected EnterpriseID %v, got %v", expectedEnterpriseID, pe.EnterpriseID)
	}
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
