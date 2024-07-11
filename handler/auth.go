package handler

import (
	"net/http"
	"soln-teachermodule/database"
	"soln-teachermodule/view/auth"
)

func HandleLoginIndex(w http.ResponseWriter, r *http.Request) error {

	return auth.Login().Render(r.Context(), w)
}

func HandleLoginCreate(w http.ResponseWriter, r *http.Request) error {
	// authenticate the user
	database.AuthenticateUser(w, r)

	return auth.Login().Render(r.Context(), w)
}
