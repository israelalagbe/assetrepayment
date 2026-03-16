package service_test

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/israelalagbe/assetrepayment/internal/domain"
	"github.com/israelalagbe/assetrepayment/internal/service"
	_ "modernc.org/sqlite"
)

// --- mock repositories ---

type mockCustomerRepo struct {
	getCustomerFn   func(tx *sql.Tx, customerID string) (*domain.Customer, error)
	updateBalanceFn func(tx *sql.Tx, customerID string, amountKobo int64) error
}

func (m *mockCustomerRepo) GetCustomerByID(tx *sql.Tx, customerID string) (*domain.Customer, error) {
	return m.getCustomerFn(tx, customerID)
}

func (m *mockCustomerRepo) UpdateBalance(tx *sql.Tx, customerID string, amountKobo int64) error {
	return m.updateBalanceFn(tx, customerID, amountKobo)
}

type mockPaymentRepo struct {
	insertFn func(tx *sql.Tx, p *domain.Payment) error
	existsFn func(tx *sql.Tx, reference string) (bool, error)
}

func (m *mockPaymentRepo) InsertPayment(tx *sql.Tx, p *domain.Payment) error {
	return m.insertFn(tx, p)
}

func (m *mockPaymentRepo) ExistsPaymentByReference(tx *sql.Tx, reference string) (bool, error) {
	return m.existsFn(tx, reference)
}

// --- helpers ---

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:?_txlock=immediate")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func okCustomerRepo() *mockCustomerRepo {
	return &mockCustomerRepo{
		getCustomerFn: func(_ *sql.Tx, _ string) (*domain.Customer, error) {
			return &domain.Customer{ID: "GIGXXXXX", OutstandingKobo: 100000000}, nil
		},
		updateBalanceFn: func(_ *sql.Tx, _ string, _ int64) error { return nil },
	}
}

func okPaymentRepo() *mockPaymentRepo {
	return &mockPaymentRepo{
		existsFn: func(_ *sql.Tx, _ string) (bool, error) { return false, nil },
		insertFn: func(_ *sql.Tx, _ *domain.Payment) error { return nil },
	}
}

// --- tests ---

func TestProcessPayment(t *testing.T) {
	validPayload := &domain.PaymentNotification{
		CustomerID:           "GIGXXXXX",
		PaymentStatus:        "COMPLETE",
		TransactionAmount:    "10000",
		TransactionDate:      "2025-11-07 14:54:16",
		TransactionReference: "REF001",
	}

	tests := []struct {
		name         string
		payload      *domain.PaymentNotification
		customerRepo func() *mockCustomerRepo
		paymentRepo  func() *mockPaymentRepo
		wantErr      error
	}{
		{
			name:         "valid payment succeeds",
			payload:      validPayload,
			customerRepo: okCustomerRepo,
			paymentRepo:  okPaymentRepo,
			wantErr:      nil,
		},
		{
			name:         "non-COMPLETE status rejected",
			payload:      &domain.PaymentNotification{
				CustomerID: "GIGXXXXX", PaymentStatus: "PENDING",
				TransactionAmount: "10000", TransactionDate: "2025-11-07 14:54:16",
				TransactionReference: "REF002",
			},
			customerRepo: okCustomerRepo,
			paymentRepo:  okPaymentRepo,
			wantErr:      domain.ErrPaymentNotComplete,
		},
		{
			name:         "missing fields rejected",
			payload:      &domain.PaymentNotification{CustomerID: "GIGXXXXX"},
			customerRepo: okCustomerRepo,
			paymentRepo:  okPaymentRepo,
			wantErr:      domain.ErrInvalidPayload,
		},
		{
			name: "invalid amount rejected",
			payload: &domain.PaymentNotification{
				CustomerID: "GIGXXXXX", PaymentStatus: "COMPLETE",
				TransactionAmount: "abc", TransactionDate: "2025-11-07 14:54:16",
				TransactionReference: "REF003",
			},
			customerRepo: okCustomerRepo,
			paymentRepo:  okPaymentRepo,
			wantErr:      domain.ErrInvalidAmount,
		},
		{
			name: "zero amount rejected",
			payload: &domain.PaymentNotification{
				CustomerID: "GIGXXXXX", PaymentStatus: "COMPLETE",
				TransactionAmount: "0", TransactionDate: "2025-11-07 14:54:16",
				TransactionReference: "REF004",
			},
			customerRepo: okCustomerRepo,
			paymentRepo:  okPaymentRepo,
			wantErr:      domain.ErrInvalidAmount,
		},
		{
			name:         "already processed reference returns success",
			payload:      validPayload,
			customerRepo: okCustomerRepo,
			paymentRepo: func() *mockPaymentRepo {
				r := okPaymentRepo()
				r.existsFn = func(_ *sql.Tx, _ string) (bool, error) { return true, nil }
				return r
			},
			wantErr: nil,
		},
		{
			name:    "unknown customer rejected",
			payload: validPayload,
			customerRepo: func() *mockCustomerRepo {
				r := okCustomerRepo()
				r.getCustomerFn = func(_ *sql.Tx, _ string) (*domain.Customer, error) {
					return nil, domain.ErrCustomerNotFound
				}
				return r
			},
			paymentRepo: okPaymentRepo,
			wantErr:     domain.ErrCustomerNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db := newTestDB(t)
			svc := service.NewPaymentService(db, tc.customerRepo(), tc.paymentRepo())
			err := svc.ProcessPayment(tc.payload)

			if tc.wantErr == nil && err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if tc.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tc.wantErr)
				}
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected error %v, got %v", tc.wantErr, err)
				}
			}
		})
	}
}
