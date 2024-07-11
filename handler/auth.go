package handler

import (
	"log"
	"net/http"
	"soln-teachermodule/database"
	"soln-teachermodule/view/auth"
	"soln-teachermodule/view/home"
)

func HandleLoginIndex(w http.ResponseWriter, r *http.Request) error {

	return auth.Login().Render(r.Context(), w)
}

func HandleLoginCreate(w http.ResponseWriter, r *http.Request) error {
	// authenticate the user
	err := database.AuthenticateUser(w, r)
	if err != nil {
		log.Fatal(err)
	} else {
		return home.Index().Render(r.Context(), w)
	}

	return auth.Login().Render(r.Context(), w)
}
