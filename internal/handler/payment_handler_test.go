package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/israelalagbe/assetrepayment/internal/domain"
	"github.com/israelalagbe/assetrepayment/internal/handler"
)

type mockPaymentService struct {
	processPaymentFn func(payload *domain.PaymentNotification) error
}

func (m *mockPaymentService) ProcessPayment(payload *domain.PaymentNotification) error {
	return m.processPaymentFn(payload)
}

func newRequest(t *testing.T, method, body string) *http.Request {
	t.Helper()
	return httptest.NewRequest(method, "/payments", bytes.NewBufferString(body))
}

func TestHandlePayment(t *testing.T) {
	validBody := `{"customer_id":"GIGXXXXX","payment_status":"COMPLETE","transaction_amount":"10000","transaction_date":"2025-11-07 14:54:16","transaction_reference":"REF001"}`

	tests := []struct {
		name       string
		method     string
		body       string
		serviceErr error
		wantStatus int
		wantKey    string
		wantVal    string
	}{
		{
			name:       "valid payment",
			method:     http.MethodPost,
			body:       validBody,
			serviceErr: nil,
			wantStatus: http.StatusOK,
			wantKey:    "status",
			wantVal:    "ok",
		},
		{
			name:       "wrong method",
			method:     http.MethodGet,
			body:       "",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "malformed json",
			method:     http.MethodPost,
			body:       `{bad json}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "duplicate payment is idempotent",
			method:     http.MethodPost,
			body:       validBody,
			serviceErr: nil,
			wantStatus: http.StatusOK,
			wantKey:    "status",
			wantVal:    "ok",
		},
		{
			name:       "non-complete status ignored",
			method:     http.MethodPost,
			body:       validBody,
			serviceErr: domain.ErrPaymentNotComplete,
			wantStatus: http.StatusOK,
			wantKey:    "status",
			wantVal:    "ignored",
		},
		{
			name:       "customer not found",
			method:     http.MethodPost,
			body:       validBody,
			serviceErr: domain.ErrCustomerNotFound,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid payload",
			method:     http.MethodPost,
			body:       validBody,
			serviceErr: domain.ErrInvalidPayload,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &mockPaymentService{
				processPaymentFn: func(_ *domain.PaymentNotification) error {
					return tc.serviceErr
				},
			}
			h := handler.NewPaymentHandler(svc)

			req := newRequest(t, tc.method, tc.body)
			rec := httptest.NewRecorder()
			h.HandlePayment(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("expected status %d, got %d", tc.wantStatus, rec.Code)
			}

			if tc.wantKey != "" {
				var resp map[string]string
				if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
					t.Fatalf("decode response: %v", err)
				}
				if resp[tc.wantKey] != tc.wantVal {
					t.Errorf("expected %s=%q, got %q", tc.wantKey, tc.wantVal, resp[tc.wantKey])
				}
			}
		})
	}
}
