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
	router.Get("/register", handler.Make(handler.HandleRegisterIndex))
	router.Post("/register", handler.Make(handler.HandleRegisterCreate))

	// endpoints for game
	router.Post("/game/login", handler.Make(handler.HandleGameLogin))
	router.Post("/game/register", handler.Make(handler.HandleGameRegister))
	router.Post("/game/getfractions", handler.Make(handler.HandleGetGameFractions))
	router.Post("/game/getworded", handler.Make(handler.HandleGetGameWorded))
	router.Post("/game/getmcquestions", handler.Make(handler.HandleGetGameMCQuestions))
	router.Post("/game/updatestatistics", handler.Make(handler.HandleUpdateStatistics))

	// then everything below will be grouped, and have the user authenticated first
	// else be redirected to login.
	router.Group(func(auth chi.Router) {
		auth.Use(handler.WithAuth)
		auth.Get("/home", handler.Make(handler.HandleHomeIndex))
		auth.Post("/classroom", handler.Make(handler.HandleClassroomIndex))
		auth.Post("/createclassroom", handler.Make(handler.HandleClassroomCreate))
		auth.Get("/getclassrooms", handler.Make(handler.HandleGetClassrooms))
		auth.Get("/getclassrooms_menu", handler.Make(handler.HandleGetClassroomsMenu))
		auth.Post("/students", handler.Make(handler.HandleGetStudents))
		auth.Post("/unenrolledstudents", handler.Make(handler.HandleGetUnenrolledStudents))
		auth.Post("/addstudents", handler.Make(handler.HandleAddStudents))
		auth.Post("/logout", handler.Make(handler.HandleLogoutCreate))

		// minigame endpoints
		auth.Get("/minigame1", handler.Make(handler.HandleMinigame1Index))
		auth.Post("/getfractions", handler.Make(handler.HandleGetFractions))
		auth.Post("/addfractionquestions", handler.Make(handler.HandleAddFractions))
		auth.Post("/update/fractions", handler.Make(handler.HandleUpdateFractions))
		// auth.Post("/delete/fractions", handler.Make(handler.HandleDeleteFractions))
		auth.Get("/minigame2", handler.Make(handler.HandleMinigame2Index))
		auth.Get("/minigame3", handler.Make(handler.HandleMinigame3Index))
		auth.Post("/getwordedquestions", handler.Make(handler.HandleGetWorded))
		// auth.Post("/add/wordedquestions", handler.Make(handler.HandleAddWorded))
		auth.Post("/update/worded", handler.Make(handler.HandleUpdateWorded))
		// auth.Post("/delete/worded", handler.Make(handler.HandleDeleteWorded))
		auth.Get("/minigame4", handler.Make(handler.HandleMinigame4Index))
		auth.Get("/minigame5", handler.Make(handler.HandleMinigame5Index))
		auth.Post("/getmcquestions", handler.Make(handler.HandleGetMCQuestions))
		auth.Post("/addmcquestions", handler.Make(handler.HandleAddMCQuestions))
		auth.Post("/update/mcquestions", handler.Make(handler.HandleUpdateMCQuestions))
		// auth.Post("/delete/mcquestions", handler.Make(handler.HandleDeleteMCQuestions))
	})

	port := os.Getenv("HTTP_LISTEN_ADDRESS")
	slog.Info("application running", "port", port)
	log.Fatal(http.ListenAndServe(port, router))
}
