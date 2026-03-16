package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/israelalagbe/assetrepayment/internal/domain"
)

type CustomerRepository interface {
	GetCustomerByID(tx *sql.Tx, customerID string) (*domain.Customer, error)
	UpdateBalance(tx *sql.Tx, customerID string, amountKobo int64) error
}

type customerRepository struct {
	db *sql.DB
}

func NewCustomerRepository(db *sql.DB) CustomerRepository {
	return &customerRepository{db: db}
}

func (r *customerRepository) GetCustomerByID(tx *sql.Tx, customerID string) (*domain.Customer, error) {
	row := tx.QueryRow(
		`SELECT id, outstanding_kobo, total_paid_kobo, created_at FROM customers WHERE id = ?`,
		customerID,
	)

	var c domain.Customer
	if err := row.Scan(&c.ID, &c.OutstandingKobo, &c.TotalPaidKobo, &c.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrCustomerNotFound
		}
		return nil, fmt.Errorf("get customer: %w", err)
	}

	return &c, nil
}

func (r *customerRepository) UpdateBalance(tx *sql.Tx, customerID string, amountKobo int64) error {
	_, err := tx.Exec(
		`UPDATE customers SET outstanding_kobo = outstanding_kobo - ?, total_paid_kobo = total_paid_kobo + ? WHERE id = ?`,
		amountKobo, amountKobo, customerID,
	)
	if err != nil {
		return fmt.Errorf("update balance: %w", err)
	}
	return nil
}
