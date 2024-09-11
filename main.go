package main

import (
	"embed"
	"log"
	"log/slog"
	"net/http"
	"os"
	"soln-teachermodule/handler"
	"soln-teachermodule/database"

	"github.com/go-chi/chi/v5"
)


//go:embed public
var FS embed.FS

func main() {
	if err := database.InitializeDatabase(); err != nil {
		log.Fatal(err)
	}
 
	router := chi.NewMux()
	router.Use(handler.WithUser)

	// handle static files
	router.Handle("/*", http.StripPrefix("/", http.FileServer(http.FS(FS))))
	router.Get("/", handler.Make(handler.HandleLoginIndex))
	router.Post("/login", handler.Make(handler.HandleLoginCreate))
	router.Get("/register", handler.Make(handler.HandleRegisterIndex))
	router.Post("/register", handler.Make(handler.HandleRegisterCreate))
	router.Post("/soln/login", handler.Make(handler.HandleLoginGame))
	// to be changed into POST -> /classroom (show classrooms associated w/ teacher)
	router.Get("/classroom", handler.Make(handler.HandleClassroomCreate))

	port := os.Getenv("HTTP_LISTEN_ADDRESS")
	slog.Info("application running", "port", port)
	log.Fatal(http.ListenAndServe(port, router))
} 