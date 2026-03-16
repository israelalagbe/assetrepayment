package service

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/israelalagbe/assetrepayment/internal/domain"
	"github.com/israelalagbe/assetrepayment/internal/repository"
)

const transactionDateLayout = "2006-01-02 15:04:05"

type PaymentService interface {
	ProcessPayment(payload *domain.PaymentNotification) error
}

type paymentService struct {
	db          *sql.DB
	customerRepo repository.CustomerRepository
	paymentRepo  repository.PaymentRepository
}

func NewPaymentService(
	db *sql.DB,
	customerRepo repository.CustomerRepository,
	paymentRepo repository.PaymentRepository,
) PaymentService {
	return &paymentService{
		db:           db,
		customerRepo: customerRepo,
		paymentRepo:  paymentRepo,
	}
}

func (s *paymentService) ProcessPayment(payload *domain.PaymentNotification) error {
	if err := validatePayload(payload); err != nil {
		return err
	}

	if domain.PaymentStatus(payload.PaymentStatus) != domain.PaymentStatusComplete {
		return domain.ErrPaymentNotComplete
	}

	amountKobo, err := parseAmountKobo(payload.TransactionAmount)
	if err != nil {
		return err
	}

	txDate, err := time.Parse(transactionDateLayout, payload.TransactionDate)
	if err != nil {
		return fmt.Errorf("invalid transaction_date format: %w", err)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	_, err = s.customerRepo.GetCustomerByID(tx, payload.CustomerID)
	if err != nil {
		return err
	}

	exists, err := s.paymentRepo.ExistsPaymentByReference(tx, payload.TransactionReference)
	if err != nil {
		return err
	}
	if exists {
		// Already processed — idempotent success.
		return nil
	}

	payment := &domain.Payment{
		CustomerID:           payload.CustomerID,
		AmountKobo:           amountKobo,
		TransactionReference: payload.TransactionReference,
		TransactionDate:      txDate,
	}

	if err = s.paymentRepo.InsertPayment(tx, payment); err != nil {
		if errors.Is(err, domain.ErrDuplicatePayment) {
			// Race condition: another request inserted the same reference concurrently.
			_ = tx.Rollback()
			err = nil
			return nil
		}
		return err
	}

	if err = s.customerRepo.UpdateBalance(tx, payload.CustomerID, amountKobo); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

func validatePayload(p *domain.PaymentNotification) error {
	if p.CustomerID == "" || p.PaymentStatus == "" || p.TransactionAmount == "" ||
		p.TransactionDate == "" || p.TransactionReference == "" {
		return domain.ErrInvalidPayload
	}
	return nil
}

func parseAmountKobo(raw string) (int64, error) {
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || v <= 0 {
		if err == nil {
			err = errors.New("amount must be greater than zero")
		}
		return 0, fmt.Errorf("%w: %w", domain.ErrInvalidAmount, err)
	}
	return v, nil
}
