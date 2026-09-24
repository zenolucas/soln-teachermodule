package handler

import (
	"net/http"
	"soln-teachermodule/view/about"
	"soln-teachermodule/view/home"
	"soln-teachermodule/view/landing"
)

func HandleLandingIndex(w http.ResponseWriter, r *http.Request) error {
	return render(w, r, landing.Index())
}

func HandleHomeIndex(w http.ResponseWriter, r *http.Request) error {
	page, err := pageFor(r, "Home · Sol'n Teacher Portal", "home", "")
	if err != nil {
		return err
	}
	return render(w, r, home.Index(page))
}

func HandleAboutIndex(w http.ResponseWriter, r *http.Request) error {
	page, err := pageFor(r, "About · Sol'n Teacher Portal", "about", "")
	if err != nil {
		return err
	}
	return render(w, r, about.Index(page))
}
