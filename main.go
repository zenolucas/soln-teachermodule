package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal(err)
	}

	router := chi.NewMux()

	// router.Get("/", )

	port := os.Getenv("HTTP_LISTEN_ADDRESS")
	slog.Info("application running", "port", port)
	log.Fatal(http.ListenAndServe(port, router))
} 