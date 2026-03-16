package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/israelalagbe/assetrepayment/internal/db"
	"github.com/israelalagbe/assetrepayment/internal/handler"
	"github.com/israelalagbe/assetrepayment/internal/repository"
	"github.com/israelalagbe/assetrepayment/internal/service"
)

func main() {
	dbPath := getEnv("DB_PATH", "./data.db")
	port := getEnv("PORT", ":8080")

	database, err := db.Open(dbPath)
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
		Addr:         port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("server listening on %s", port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
