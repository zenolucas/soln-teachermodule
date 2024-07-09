package main

import (
	"fmt"
	"embed"
	"log"
	"log/slog"
	"net/http"
	"os"
	"database/sql"
	"soln-teachermodule/handler"

	"github.com/go-sql-driver/mysql"
	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
)

var db *sql.DB

//go:embed public
var FS embed.FS

func main() {
	if err := initializeDatabase(); err != nil {
		log.Fatal(err)
	}
	

	router := chi.NewMux()
	router.Use(handler.WithUser)

	// handle static files
	router.Handle("/*", http.StripPrefix("/", http.FileServer(http.FS(FS))))
	router.Get("/", handler.Make(handler.HandleLoginIndex))
	// router.Get("/login", handler.Make(handler.HandleLoginIndex))

	port := os.Getenv("HTTP_LISTEN_ADDRESS")
	slog.Info("application running", "port", port)
	log.Fatal(http.ListenAndServe(port, router))
} 

func initializeDatabase() error {
	if err := godotenv.Load(); err != nil {
		log.Fatal(err)
		return err
	}

	// Capture connection properties.
	cfg := mysql.Config{
		User:                 os.Getenv("DBUSER"),
		Passwd:               os.Getenv("DBPASS"),
		Net:                  "tcp",
		Addr:                 "127.0.0.1:3306",
		DBName:               "soln_db",
	}

	// Get a database handle.
	var err error
	db, err = sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		log.Fatal(err)
	}

	pingErr := db.Ping()
	if pingErr != nil {
		log.Fatal(pingErr)
	}
	fmt.Println("Database connection established.")

	return err
}