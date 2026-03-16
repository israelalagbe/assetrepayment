package handler

import (
	"encoding/json"
	"net/http"
)

func HandleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"service": "Asset Repayment API",
		"version": "1.0.0",
		"endpoints": map[string]string{
			"POST /payments": "Submit a payment notification",
		},
	})
}
