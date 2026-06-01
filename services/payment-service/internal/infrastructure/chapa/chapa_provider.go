// Package chapa implements the domain.PaymentProvider interface for Chapa,
// Ethiopia's leading payment gateway (https://developer.chapa.co).
//
// Key design decisions:
//   - Chapa has no native recurring subscription API; subscriptions are managed
//     as one-time payments per billing cycle by the Veritas scheduler.
//   - SyncPlan and DeactivatePlan are no-ops (Chapa has no plan registry).
//   - CancelSubscription and ReactivateSubscription are no-ops.
//   - RefundPayment returns ErrNotSupported in v1.
//   - Webhook signature is verified using HMAC-SHA256 of the raw payload against
//     either the "chapa-signature" or "x-chapa-signature" header.
package chapa

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/domain"
	"go.uber.org/zap"
)

const (
	chapaBaseURL     = "https://api.chapa.co/v1"
	chapaInitPath    = "/transaction/initialize"
	chapaVerifyPath  = "/transaction/verify/"
	httpTimeout      = 15 * time.Second
)

type chapaProvider struct {
	secretKey     string
	webhookSecret string
	returnURL     string
	callbackURL   string
	httpClient    *http.Client
}

// NewChapaProvider creates a Chapa implementation of domain.PaymentProvider.
//
//   - secretKey: Chapa secret key (CHASECK_TEST-... or CHASECK-...)
//   - webhookSecret: The secret hash configured in the Chapa dashboard for HMAC verification
//   - returnURL: Where the user is redirected after payment (frontend success page)
//   - callbackURL: Where Chapa sends the async charge.success webhook
func NewChapaProvider(secretKey, webhookSecret, returnURL, callbackURL string) domain.PaymentProvider {
	return &chapaProvider{
		secretKey:     secretKey,
		webhookSecret: webhookSecret,
		returnURL:     returnURL,
		callbackURL:   callbackURL,
		httpClient:    &http.Client{Timeout: httpTimeout},
	}
}

// ─── Initialize ───────────────────────────────────────────────────────────────

type initializeRequest struct {
	Amount        string                `json:"amount"`
	Currency      string                `json:"currency"`
	TxRef         string                `json:"tx_ref"`
	ReturnURL     string                `json:"return_url"`
	CallbackURL   string                `json:"callback_url"`
	Customization initializeCustomize   `json:"customization"`
	Meta          map[string]string     `json:"meta"`
}

type initializeCustomize struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type initializeResponse struct {
	Message string `json:"message"`
	Status  string `json:"status"`
	Data    struct {
		CheckoutURL string `json:"checkout_url"`
	} `json:"data"`
}

// CreateCheckoutSession calls POST /transaction/initialize and returns the checkout URL.
// The plan must be denominated in ETB.
func (p *chapaProvider) CreateCheckoutSession(ctx context.Context, req domain.CheckoutRequest) (string, error) {
	if req.Plan.Currency != domain.CurrencyETB {
		return "", fmt.Errorf("chapa: %w: only ETB plans are supported, got %s", domain.ErrInvalidInput, req.Plan.Currency)
	}

	body := initializeRequest{
		Amount:      fmt.Sprintf("%.2f", req.Plan.Price),
		Currency:    string(req.Plan.Currency),
		TxRef:       req.TxRef,
		ReturnURL:   p.returnURL,
		CallbackURL: p.callbackURL,
		Customization: initializeCustomize{
			Title:       fmt.Sprintf("Veritas Subscription - %s", req.Plan.Name),
			Description: req.Plan.Description,
		},
		Meta: map[string]string{
			"enterprise_id": req.EnterpriseID.String(),
			"plan_id":       req.Plan.ID.String(),
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("chapa: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, chapaBaseURL+chapaInitPath, strings.NewReader(string(jsonBody)))
	if err != nil {
		return "", fmt.Errorf("chapa: build initialize request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.secretKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("chapa: initialize request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("chapa: read initialize response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("chapa: initialize returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var initResp initializeResponse
	if err := json.Unmarshal(respBody, &initResp); err != nil {
		return "", fmt.Errorf("chapa: decode initialize response: %w", err)
	}

	if initResp.Data.CheckoutURL == "" {
		return "", fmt.Errorf("chapa: empty checkout_url in response: %s", string(respBody))
	}

	return initResp.Data.CheckoutURL, nil
}

// ─── Webhook Verification ─────────────────────────────────────────────────────

type chapaWebhookPayload struct {
	Event         string  `json:"event"`
	TxRef         string  `json:"tx_ref"`
	Reference     string  `json:"reference"`
	Status        string  `json:"status"`
	Amount        float64 `json:"amount,string"`
	Currency      string  `json:"currency"`
	FirstName     string  `json:"first_name"`
	LastName      string  `json:"last_name"`
	Email         string  `json:"email"`
	Mobile        string  `json:"mobile"`
	PaymentMethod string  `json:"payment_method"`
	CreatedAt     string  `json:"created_at"`
}

// VerifyWebhookEvent validates the Chapa webhook signature and verifies the
// transaction via the Chapa API before returning a PaymentEvent.
//
// Chapa sends the HMAC-SHA256 signature in either "chapa-signature" or
// "x-chapa-signature" headers. We accept either.
func (p *chapaProvider) VerifyWebhookEvent(payload []byte, sigHeader string) (*domain.PaymentEvent, error) {
	// 1. Validate HMAC signature.
	if p.webhookSecret != "" {
		if err := p.validateSignature(payload, sigHeader); err != nil {
			return nil, err
		}
	}

	// 2. Decode payload.
	var wh chapaWebhookPayload
	if err := json.Unmarshal(payload, &wh); err != nil {
		return nil, fmt.Errorf("chapa: decode webhook payload: %w", err)
	}

	// 3. Only process charge.success events (others are ignored).
	eventType := mapChapaEvent(wh.Event, wh.Status)

	// 4. Verify transaction independently via the Chapa API.
	verifiedEvent, err := p.verifyTransaction(context.Background(), wh.TxRef)
	if err != nil {
		return nil, fmt.Errorf("chapa: verify transaction %s: %w", wh.TxRef, err)
	}

	// 5. Merge meta fields (enterprise_id, plan_id) from verification response.
	enterpriseID := uuid.Nil
	planID := uuid.Nil
	if meta, ok := verifiedEvent["meta"].(map[string]any); ok {
		if eid, _ := meta["enterprise_id"].(string); eid != "" {
			enterpriseID, _ = uuid.Parse(eid)
		}
		if pid, _ := meta["plan_id"].(string); pid != "" {
			planID, _ = uuid.Parse(pid)
		}
	}

	rawMap := make(map[string]any)
	_ = json.Unmarshal(payload, &rawMap)

	pe := &domain.PaymentEvent{
		// Use TxRef as EventID for Chapa idempotency (Chapa has no global event ID).
		EventID:      wh.TxRef,
		EventType:    eventType,
		TxRef:        wh.TxRef,
		EnterpriseID: enterpriseID,
		PlanID:       planID,
		Amount:       wh.Amount,
		Currency:     domain.Currency(wh.Currency),
		Raw:          rawMap,
	}

	return pe, nil
}

// validateSignature checks the chapa-signature or x-chapa-signature header.
func (p *chapaProvider) validateSignature(payload []byte, sigHeader string) error {
	if sigHeader == "" {
		return fmt.Errorf("chapa: missing webhook signature header")
	}

	mac := hmac.New(sha256.New, []byte(p.webhookSecret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(sigHeader)) {
		return fmt.Errorf("chapa: webhook signature mismatch")
	}
	return nil
}

// ─── Transaction Verification ─────────────────────────────────────────────────

type verifyResponse struct {
	Message string         `json:"message"`
	Status  string         `json:"status"`
	Data    map[string]any `json:"data"`
}

// verifyTransaction calls GET /transaction/verify/<tx_ref> and returns the data object.
func (p *chapaProvider) verifyTransaction(ctx context.Context, txRef string) (map[string]any, error) {
	url := chapaBaseURL + chapaVerifyPath + txRef
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build verify request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.secretKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("verify request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read verify response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("verify returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var vr verifyResponse
	if err := json.Unmarshal(body, &vr); err != nil {
		return nil, fmt.Errorf("decode verify response: %w", err)
	}

	if vr.Status != "success" {
		return nil, fmt.Errorf("chapa verify status: %s, message: %s", vr.Status, vr.Message)
	}

	return vr.Data, nil
}

// mapChapaEvent converts Chapa event/status strings to a Veritas-internal event type.
func mapChapaEvent(event, status string) string {
	switch {
	case event == "charge.success" || status == "success":
		return "payment.success"
	case event == "charge.failed" || event == "charge.failed/cancelled" || status == "failed" || status == "cancelled":
		return "payment.failed"
	default:
		return event
	}
}

// ─── No-op operations ─────────────────────────────────────────────────────────

// CancelSubscription is a no-op for Chapa; subscriptions are managed in Veritas DB only.
func (p *chapaProvider) CancelSubscription(_ context.Context, _ string, _ bool) error {
	zap.L().Warn("chapa: CancelSubscription called — Chapa has no native recurring billing, this is a no-op")
	return nil
}

// ReactivateSubscription is a no-op for Chapa.
func (p *chapaProvider) ReactivateSubscription(_ context.Context, _ string) error {
	zap.L().Warn("chapa: ReactivateSubscription called — no-op")
	return nil
}

// SyncPlan is a no-op for Chapa (no plan/product registry).
func (p *chapaProvider) SyncPlan(_ context.Context, _ *domain.SubscriptionPlan) (string, error) {
	return "", nil
}

// DeactivatePlan is a no-op for Chapa.
func (p *chapaProvider) DeactivatePlan(_ context.Context, _ string) error {
	return nil
}

// RefundPayment returns ErrNotSupported for Chapa in v1.
// Chapa refunds must be initiated manually via the Chapa dashboard.
func (p *chapaProvider) RefundPayment(_ context.Context, _ string, _ float64) error {
	return domain.ErrNotSupported
}
