package main

import (
	"log"
	"net/http"
	"time"

	"github.com/israelalagbe/assetrepayment/internal/config"
	"github.com/israelalagbe/assetrepayment/internal/db"
	"github.com/israelalagbe/assetrepayment/internal/handler"
	"github.com/israelalagbe/assetrepayment/internal/repository"
	"github.com/israelalagbe/assetrepayment/internal/service"
)

func main() {
	cfg := config.Load()

	database, err := db.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer database.Close()

	if err := db.Migrate(database, "./migrations"); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	customerRepo := repository.NewCustomerRepository(database)
	paymentRepo := repository.NewPaymentRepository(database)
	paymentSvc := service.NewPaymentService(database, customerRepo, paymentRepo)
	paymentHandler := handler.NewPaymentHandler(paymentSvc)

	mux := http.NewServeMux()
	mux.HandleFunc("/payments", paymentHandler.HandlePayment)

	srv := &http.Server{
		Addr:         cfg.Port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("server listening on %s", cfg.Port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
