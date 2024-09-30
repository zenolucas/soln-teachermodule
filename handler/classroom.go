package handler

import (
	"fmt"
	"net/http"
	"soln-teachermodule/database"
	"soln-teachermodule/view/classroom"
)

func HandleClassroomIndex(w http.ResponseWriter, r *http.Request) error {
	return render(w, r, classroom.Classroom())
}

func GetStudents(w http.ResponseWriter, r *http.Request) error {
	// fetch users from database
	students, err := database.GetStudents()
	if err != nil {
		http.Error(w, "Unable to get students", http.StatusInternalServerError)
		return err
	}

	for i, student := range students {
		fmt.Fprintf(w, `
			<tr>
				<th>%d</th>
				<td>%s</td>
				<td class="flex justify-end"><a href="" class="btn btn-secondary">edit</a></td>
			</tr>	
		`, i+1, student.Username)
	}
	return err
}

// func HandleClassroomCreate(w http.ResponseWriter, r *http.Request) {
// 	data := classroom.CreateParams{
// 		Classname: r.FormValue("classname"),
// 		Section:   r.FormValue("section"),
// 	}

// 	// maybe perform some data cleaning/inspection here?
// 	// if bad, return error, else insert classroom
// 	db.InsertClassroom(w, r)
// 	return nil
// }
