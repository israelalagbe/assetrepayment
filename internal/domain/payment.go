package domain

import "time"

type PaymentStatus string

const (
	PaymentStatusComplete PaymentStatus = "COMPLETE"
)

// PaymentNotification is the inbound payload from the payment provider.
type PaymentNotification struct {
	CustomerID           string `json:"customer_id"`
	PaymentStatus        string `json:"payment_status"`
	TransactionAmount    string `json:"transaction_amount"`
	TransactionDate      string `json:"transaction_date"`
	TransactionReference string `json:"transaction_reference"`
}

// Payment represents a stored payment record.
type Payment struct {
	ID                   int64
	CustomerID           string
	AmountKobo           int64 // stored in kobo to avoid float precision issues
	TransactionReference string
	TransactionDate      time.Time
	CreatedAt            time.Time
}
