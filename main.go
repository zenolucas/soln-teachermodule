package main

import (
	"context"
	"embed"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"soln-teachermodule/database"
	"soln-teachermodule/handler"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

//go:embed public
var FS embed.FS

func main() {
	if err := database.InitializeDatabase(); err != nil {
		log.Fatal(err)
	}

	// Must run after InitializeDatabase, which loads .env - SESSION_SECRET/
	// GAME_TOKEN_SECRET need to be visible to os.Getenv by this point.
	if err := handler.InitSessionStore(); err != nil {
		log.Fatal(err)
	}
	if err := handler.InitGameTokenStore(); err != nil {
		log.Fatal(err)
	}

	router := chi.NewMux()
	// Several handlers still panic on bad input (unchecked type assertions, index
	// access with no bounds check) rather than returning an error - without this,
	// each of those drops the connection instead of the client getting a 500.
	router.Use(middleware.Recoverer)

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
	router.Post("/game/postsavedata", handler.Make(handler.HandleUpdateSaveData))

	// then everything below will be grouped with the Auth middleware, and have the user authenticated first
	// else be redirected to login.
	router.Group(func(auth chi.Router) {
		auth.Use(handler.WithAuth)
		auth.Get("/home", handler.Make(handler.HandleHomeIndex))
		auth.Get("/classroom", handler.Make(handler.HandleClassroomIndex))
		auth.Get("/classroom/minigames", handler.Make(handler.HandleClassroomMinigames))
		auth.Get("/classroom/students", handler.Make(handler.HandleClassroomStudents))
		auth.Get("/classroom/statistics", handler.Make(handler.HandleClassroomStatistics))
		auth.Get("/about", handler.Make(handler.HandleAboutIndex))
		auth.Post("/createclassroom", handler.Make(handler.HandleClassroomCreate))
		auth.Get("/getclassrooms", handler.Make(handler.HandleGetClassrooms))
		auth.Get("/getclassrooms_menu", handler.Make(handler.HandleGetClassroomsMenu))
		auth.Post("/students", handler.Make(handler.HandleGetStudents))
		auth.Get("/students", handler.Make(handler.HandleGetStudents))
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
		auth.Get("/statistics/student/fraction", handler.Make(handler.HandleGetStudentFractionScore))
		auth.Get("/statistics/student/worded", handler.Make(handler.HandleGetStudentWordedScore))
		auth.Get("/statistics/student/quiz", handler.Make(handler.HandleGetStudentQuizScore))
		auth.Get("/statistics/fraction/question/chart", handler.Make(handler.HandleFractionQuestionCharts))
		auth.Get("/statistics/worded/question/chart", handler.Make(handler.HandleWordedQuestionCharts))
		auth.Get("/statistics/quiz/class", handler.Make(handler.HandleQuizClassStatistics))
		auth.Get("/statistics/quiz/question", handler.Make(handler.HandleQuizQuestionStatisticsIndex))
		auth.Get("/statistics/quiz/question/chart", handler.Make(handler.HandleQuizQuestionCharts))
		auth.Get("/statistics/quiz/question/data", handler.Make(handler.HandleQuizResponseStatistics))
	})

	port := os.Getenv("HTTP_LISTEN_ADDRESS")

	// Wrap the entire router with CORS
	wrapped := handler.WithCORS(router)

	// The zero-value server (a bare http.ListenAndServe call) has no timeouts at all,
	// so a single slow or half-open client could hold a connection indefinitely.
	// WriteTimeout is set well above a typical "5s" default deliberately: this server
	// also serves public/downloads/soln.zip (~57MB, the game client) straight off
	// disk, and a blanket 10s WriteTimeout would cut that download off for any
	// connection slower than ~45Mbps sustained - 5 minutes comfortably covers even a
	// slow mobile connection (~1.5Mbps) while still bounding a genuinely stuck one.
	srv := &http.Server{
		Addr:         port,
		Handler:      wrapped,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Minute,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		slog.Info("application running", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}
	slog.Info("server stopped gracefully")
}
