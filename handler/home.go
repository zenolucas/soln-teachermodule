package handler

import (
	"net/http"
	"soln-teachermodule/view/home"
	"soln-teachermodule/view/landing"
)

func HandleLandingIndex(w http.ResponseWriter, r *http.Request) error {
	return render(w, r, landing.Index())
}

func HandleHomeIndex(w http.ResponseWriter, r *http.Request) error {
	return render(w, r, home.Index())
}
