package handler

import (
	"net/http"
	"soln-teachermodule/view/classroom"
)

func HandleClassroomCreate(w http.ResponseWriter, r *http.Request) error {
	return render(w, r, classroom.Classroom())
}
