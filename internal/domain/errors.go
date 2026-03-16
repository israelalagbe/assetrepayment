package domain

import "errors"

var (
	ErrDuplicatePayment   = errors.New("duplicate payment: transaction reference already processed")
	ErrCustomerNotFound   = errors.New("customer not found")
	ErrPaymentNotComplete = errors.New("payment ignored: status is not COMPLETE")
	ErrInvalidAmount      = errors.New("invalid transaction amount")
	ErrInvalidPayload     = errors.New("invalid payload: missing required fields")
)
