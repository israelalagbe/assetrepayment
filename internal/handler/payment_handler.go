package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/israelalagbe/assetrepayment/internal/domain"
	"github.com/israelalagbe/assetrepayment/internal/service"
)

type PaymentHandler struct {
	service service.PaymentService
}

func NewPaymentHandler(svc service.PaymentService) *PaymentHandler {
	return &PaymentHandler{service: svc}
}

func (h *PaymentHandler) HandlePayment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var payload domain.PaymentNotification
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "malformed JSON"})
		return
	}

	err := h.service.ProcessPayment(&payload)
	if err == nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	switch {
	case errors.Is(err, domain.ErrPaymentNotComplete):
		// Silently accept non-COMPLETE payments; they are not failures.
		writeJSON(w, http.StatusOK, map[string]string{"status": "ignored"})
	case errors.Is(err, domain.ErrInvalidPayload),
		errors.Is(err, domain.ErrInvalidAmount):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
	case errors.Is(err, domain.ErrCustomerNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}
