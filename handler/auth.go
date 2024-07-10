package handler

import (
	"net/http"
	"soln-teachermodule/view/auth"
)

func HandleLoginIndex(w http.ResponseWriter, r *http.Request) error {

	
	return auth.Login().Render(r.Context(), w) 
}