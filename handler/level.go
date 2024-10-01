package handler

import (
	"net/http"
	"soln-teachermodule/view/level"
)

func HandleLevelIndex(w http.ResponseWriter, r *http.Request) error {
	return render(w, r, view.Level())
}
