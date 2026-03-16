package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/israelalagbe/assetrepayment/internal/domain"
)

type PaymentRepository interface {
	InsertPayment(tx *sql.Tx, p *domain.Payment) error
	ExistsPaymentByReference(tx *sql.Tx, reference string) (bool, error)
}

type paymentRepository struct {
	db *sql.DB
}

func NewPaymentRepository(db *sql.DB) PaymentRepository {
	return &paymentRepository{db: db}
}

func (r *paymentRepository) InsertPayment(tx *sql.Tx, p *domain.Payment) error {
	_, err := tx.Exec(
		`INSERT INTO payments (customer_id, amount_kobo, transaction_reference, transaction_date) VALUES (?, ?, ?, ?)`,
		p.CustomerID, p.AmountKobo, p.TransactionReference, p.TransactionDate.Format(time.DateTime),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return domain.ErrDuplicatePayment
		}
		return fmt.Errorf("insert payment: %w", err)
	}
	return nil
}

func (r *paymentRepository) ExistsPaymentByReference(tx *sql.Tx, reference string) (bool, error) {
	var count int
	err := tx.QueryRow(
		`SELECT COUNT(*) FROM payments WHERE transaction_reference = ?`, reference,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check reference: %w", err)
	}
	return count > 0, nil
}
