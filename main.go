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

	// handle static files in public folder
	router.Handle("/*", http.StripPrefix("/", http.FileServer(http.FS(FS))))
	router.Get("/", handler.Make(handler.HandleLandingIndex))
	router.Get("/login", handler.Make(handler.HandleLoginIndex))
	router.Post("/login", handler.Make(handler.HandleLoginCreate))
	router.Get("/register", handler.Make(handler.HandleRegisterIndex))
	router.Post("/register", handler.Make(handler.HandleRegisterCreate))

	// endpoints for game
	router.Post("/game/login", handler.Make(handler.HandleGameLogin))
	router.Post("/game/register", handler.Make(handler.HandleGameRegister))
	router.Post("/game/getfractions", handler.Make(handler.HandleGetGameFractions))
	router.Post("/game/getworded", handler.Make(handler.HandleGetGameWorded))
	router.Post("/game/getmcquestions", handler.Make(handler.HandleGetGameMCQuestions))
	router.Post("/game/add/statistics/quiz", handler.Make(handler.HandlePostQuizScore))
	router.Post("/game/add/statistics/quiz/response", handler.Make(handler.HandleQuizResponse))
	router.Post("/game/add/statistics/fraction", handler.Make(handler.HandleAddStatisticsFraction))
	router.Post("/game/getsavedata", handler.Make(handler.HandleGetSaveData))
	router.Post("/game/postsavedata", handler.Make(handler.HandlePostSaveData))

	// then everything below will be grouped with the Auth middleware, and have the user authenticated first
	// else be redirected to login.
	router.Group(func(auth chi.Router) {
		auth.Use(handler.WithAuth)
		auth.Get("/home", handler.Make(handler.HandleHomeIndex))
		auth.Get("/classroom", handler.Make(handler.HandleClassroomIndex))
		auth.Post("/createclassroom", handler.Make(handler.HandleClassroomCreate))
		auth.Get("/getclassrooms", handler.Make(handler.HandleGetClassrooms))
		auth.Get("/getclassrooms_menu", handler.Make(handler.HandleGetClassroomsMenu))
		auth.Post("/students", handler.Make(handler.HandleGetStudents))
		auth.Get("/student/score", handler.Make(handler.HandleStudentScoreIndex))
		auth.Post("/unenrolledstudents", handler.Make(handler.HandleGetUnenrolledStudents))
		auth.Post("/addstudents", handler.Make(handler.HandleAddStudents))
		auth.Post("/delete/student", handler.Make(handler.HandleUnenrollStudent))
		auth.Post("/logout", handler.Make(handler.HandleLogoutCreate))

		// minigame endpoints
		auth.Get("/minigame", handler.Make(handler.HandleMinigameIndex))
		auth.Post("/getfractions", handler.Make(handler.HandleGetFractions))
		auth.Post("/getwordedquestions", handler.Make(handler.HandleGetWorded))
		auth.Post("/getmcquestions", handler.Make(handler.HandleGetMCQuestions))
		auth.Post("/add/fractionquestions", handler.Make(handler.HandleAddFractions))
		auth.Post("/add/wordedquestions", handler.Make(handler.HandleAddWorded))
		auth.Post("/add/mcquestions", handler.Make(handler.HandleAddMCQuestions))
		auth.Post("/update/fractions", handler.Make(handler.HandleUpdateFractions))
		auth.Post("/update/worded", handler.Make(handler.HandleUpdateWorded))
		auth.Post("/update/mcquestions", handler.Make(handler.HandleUpdateMCQuestions))
		auth.Post("/delete/fractions", handler.Make(handler.HandleDeleteFractions)) // TO BE CHANGED FROM POST TO DELETE
		auth.Post("/delete/worded", handler.Make(handler.HandleDeleteWorded))
		auth.Post("/delete/mcquestions", handler.Make(handler.HandleDeleteMCQuestions))

		// statistics endpoints
		auth.Get("/statistics/fraction", handler.Make(handler.HandleStatisticsIndex))
		auth.Get("/statistics/quiz", handler.Make(handler.HandleStatisticsIndex))
		auth.Get("/statistics/quiz/score", handler.Make(handler.HandleGetQuizScores))
	})

	router.Get("/statistics/student/fraction", handler.Make(handler.HandleGetStudentFractionScore))
	router.Get("/statistics/student/worded", handler.Make(handler.HandleGetStudentWordedScore))
	router.Get("/statistics/student/quiz", handler.Make(handler.HandleGetStudentQuizScore))
	router.Get("/statistics/fraction/question/chart", handler.Make(handler.HandleFractionQuestionCharts))
	router.Get("/statistics/fraction/question/data", handler.Make(handler.HandleFractionResponseStatistics))
	router.Get("/statistics/worded/question/chart", handler.Make(handler.HandleWordedQuestionCharts))
	router.Get("/statistics/worded/question/data", handler.Make(handler.HandleWordedResponseStatistics))
	router.Get("/statistics/quiz/class", handler.Make(handler.HandleQuizClassStatistics))
	router.Get("/statistics/quiz/question", handler.Make(handler.HandleQuizQuestionStatisticsIndex))
	router.Get("/statistics/quiz/question/chart", handler.Make(handler.HandleQuizQuestionCharts))
	router.Get("/statistics/quiz/question/data", handler.Make(handler.HandleQuizResponseStatistics))

	port := os.Getenv("HTTP_LISTEN_ADDRESS")
	slog.Info("application running", "port", port)
	log.Fatal(http.ListenAndServe(port, router))
}
