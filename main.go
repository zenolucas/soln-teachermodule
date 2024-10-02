package main

import (
	"embed"
	"log"
	"log/slog"
	"net/http"
	"os"
	"soln-teachermodule/database"
	"soln-teachermodule/handler"

	"github.com/go-chi/chi/v5"
)

//go:embed public
var FS embed.FS

func main() {
	if err := database.InitializeDatabase(); err != nil {
		log.Fatal(err)
	}

	router := chi.NewMux()

	// handle static files
	router.Handle("/*", http.StripPrefix("/", http.FileServer(http.FS(FS))))
	router.Get("/", handler.Make(handler.HandleLoginIndex))
	router.Post("/login", handler.Make(handler.HandleLoginCreate))
	// router.Get("/register", handler.Make(handler.HandleRegisterIndex))
	// router.Post("/register", handler.Make(handler.HandleRegisterCreate))
	// router.Post("/logout", handler.Make(handler.HandleLogoutCreate))

	// router.Get("/students", handler.Make(handler.GetStudents))
	// router.Get("/getmcquestions", handler.Make(handler.HandleGetMCQuestions))
	// router.Post("/updatemcquestions", handler.Make(handler.HandleUpdateMCQuestions))

	router.Post("/soln/login", handler.Make(handler.HandleLoginGame))
	router.Post("/soln/getquestions", handler.Make(handler.HandleGetQuestions))
	// then everything below will be grouped, and have the user authenticated first
	// else be redirected to login.
	router.Group(func (auth chi.Router) {
		auth.Use(handler.WithAuth)
		auth.Get("/classroom", handler.Make(handler.HandleClassroomIndex))
		auth.Get("/level", handler.Make(handler.HandleLevelIndex))

		// auth.Post("/login", handler.Make(handler.HandleLoginCreate))
		auth.Get("/register", handler.Make(handler.HandleRegisterIndex))
		auth.Post("/register", handler.Make(handler.HandleRegisterCreate))
		auth.Post("/logout", handler.Make(handler.HandleLogoutCreate))

		auth.Get("/students", handler.Make(handler.GetStudents))
		auth.Get("/getmcquestions", handler.Make(handler.HandleGetMCQuestions))
		auth.Post("/updatemcquestions", handler.Make(handler.HandleUpdateMCQuestions))
	})

	// router.Get("/classroom", handler.Make(handler.HandleClassroomIndex))

	port := os.Getenv("HTTP_LISTEN_ADDRESS")
	slog.Info("application running", "port", port)
	log.Fatal(http.ListenAndServe(port, router))
}
