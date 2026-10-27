package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/kaitoszk/stockade/internal/database"
	"github.com/kaitoszk/stockade/internal/handler"
	"github.com/kaitoszk/stockade/internal/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("server stopped", slog.Any("err", err))
		os.Exit(1)
	}
}

func run (logger *slog.Logger) error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("DATABASE_URL is not set")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("【DEBUG】run: DATABASE_URL set=%t, PORT=%s\n", dsn != "", port)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := database.Open(ctx, database.DefaultConfig(dsn))
	if err != nil {
		return err
	}
	defer db.Close()

	fmt.Printf("【DEBUG】run: db stats=%+v\n", db.Stats())

	productService := service.NewProductService(db)
	productHandler := handler.NewProductHandler(productService, logger)

	srv := &http.Server{
		Addr: ":" + port,
		Handler: handler.NewRouter(productHandler),

		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout: 10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout: 60 * time.Second,
	}
	logger.Info("server starting", slog.String("addr", srv.Addr))
	return srv.ListenAndServe()
	
}