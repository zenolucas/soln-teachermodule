package handler

import (
	"errors"
	"fmt"
	"net/http"

	// "os"
	"soln-teachermodule/database"
	"soln-teachermodule/types"
	"soln-teachermodule/view/classroom"
	"soln-teachermodule/view/home"
	"strconv"
	// "github.com/gorilla/sessions"
)

func HandleClassroomIndex(w http.ResponseWriter, r *http.Request) error {
	room := types.Classroom{
		ClassroomID: r.URL.Query().Get("classroom_id"),
	}

	if room.ClassroomID == "" {
		http.Error(w, "Missing classroom_id", http.StatusBadRequest)
		return errors.New("bad request")
	}

	// convert classroomID to int
	classroomID, _ := strconv.Atoi(room.ClassroomID)

	// save classroomID in session
	// store := sessions.NewCookieStore([]byte(os.Getenv("SESSION_SECRET")))
	session, _ := store.Get(r, sessionUserKey)
	session.Values["classroomID"] = classroomID
	err := session.Save(r, w)
	if err != nil {
		return err
	}

	// fmt.Print("classroomID is ", session.Values["classroomID"])
	w.Header().Set("Access-Control-Allow-Origin", "*")
	return render(w, r, classroom.Classroom(room.ClassroomID))
}

func HandleGetClassrooms(w http.ResponseWriter, r *http.Request) error {
	// store := sessions.NewCookieStore([]byte(os.Getenv("SESSION_SECRET")))
	session, _ := store.Get(r, sessionUserKey)
	teacherID := session.Values["teacherID"].(int)
	var classrooms []types.Classroom

	classrooms, err := database.GetClassrooms(teacherID)
	if err != nil {
		return err
	}

	for _, classroom := range classrooms {
		fmt.Fprintf(w, `
		<div class="glass card card-bordered bg-neutral w-96 shadow-xl h-80 flex justify-center ml-8 mt-8">
				<figure>
					<img src="/public/images/bg/soln-card-image.png" alt="image" />
				</figure>
				<div class="card-body">
					<h2 class="card-title">%s - %s</h2>
					<p>%s</p>
					<div class="card-actions justify-end">
						<a href="/classroom?classroom_id=%s" class="btn btn-secondary"> Open </a> 
					</div>
				</div>
			</div>
		`, classroom.ClassroomName, classroom.Section, classroom.Description, classroom.ClassroomID)
	}

	return nil
}

func HandleGetClassroomsMenu(w http.ResponseWriter, r *http.Request) error {
	// store := sessions.NewCookieStore([]byte(os.Getenv("SESSION_SECRET")))
	session, _ := store.Get(r, sessionUserKey)
	teacherID := session.Values["teacherID"].(int)
	var classrooms []types.Classroom

	classrooms, err := database.GetClassrooms(teacherID)
	if err != nil {
		return err
	}

	for _, classroom := range classrooms {
		fmt.Fprintf(w, `
		<a href="/classroom?classroom_id=%s" class="btn btn-wide btn-ghost w-full text-white text-left justify-start mt-2"> <i //
				class="fa-solid fa-users fa-2xl ml-6" style="color: #ffffff;"></i> %s - Section %s</div>
		`, classroom.ClassroomID, classroom.ClassroomName, classroom.Section)
	}

	return nil
}

func HandleGetStudents(w http.ResponseWriter, r *http.Request) error {
	err := r.ParseForm()
	if err != nil {
		fmt.Println("Error parsing form:", err)
		return err
	}
	classroomIDStr := r.FormValue("classroomID")
	fmt.Println("Parsed classroomIDStr:", classroomIDStr)

	// convert to int
	classroomID, err := strconv.Atoi(classroomIDStr)
	if err != nil {
		return err
	}
	// fetch users from database
	students, err := database.GetStudents(classroomID)
	if err != nil {
		http.Error(w, "Unable to get students", http.StatusInternalServerError)
		return err
	}

	for i, student := range students {
		fmt.Fprintf(w, `
			<tr>
				<th>%d</th>
				<td>%s %s</td>
				<td class="flex justify-end">
					<a target="_blank" href="/student/score?userID=%s" class="btn btn-primary text-white mr-2">
						view scores
					</a>
					<form hx-post="/delete/student">
						<input type="hidden" name="studentID" value="%s" />
						<input type="hidden" name="classroomID" value="%s" />
						<button type="submit" class="btn"><i class="fa-solid fa-trash" style="color: #f66151;"></i></button>
					</form>
				</td>
			</tr>	
		`, i+1, student.Firstname, student.Lastname, student.UserID, student.UserID, classroomIDStr)
	}
	return nil
}

func HandleUnenrollStudent(w http.ResponseWriter, r *http.Request) error {
	err := r.ParseForm()
	if err != nil {
		fmt.Println("Error parsing form:", err)
		return err
	}
	studentIDStr := r.FormValue("studentID")
	classroomIDStr := r.FormValue("classroomID")
	studentID, _ := strconv.Atoi(studentIDStr)
	fmt.Print("we got studentID ")
	if err := database.UnenrollStudent(studentID); err != nil {
		return err
	}
	fmt.Print("delete success!")

	url := "/classroom?classroom_id="
	url += classroomIDStr
	hxRedirect(w, r, url)
	return nil
}

func HandleGetUnenrolledStudents(w http.ResponseWriter, r *http.Request) error {
	room := types.Classroom{
		ClassroomID: r.FormValue("classroomID"),
	}

	ClassroomID, err := strconv.Atoi(room.ClassroomID)
	if err != nil {
		return err
	}

	students, err := database.GetUnenrolledStudents(ClassroomID)
	if err != nil {
		http.Error(w, "Unable to get students", http.StatusInternalServerError)
		return err
	}

	// output student array here
	for _, student := range students {
		fmt.Fprintf(w, `
			<tr>
				<td> <input type="checkbox" name="userID" value="%s" class="checkbox-item"/></td>
				<td>%s %s</td>
			</tr>	
		`, student.UserID, student.Firstname, student.Lastname)
	}

	fmt.Fprintf(w, `
	  <script>
	    // script for the select all checkbox
		// Get references to the "Select All" checkbox and the individual checkboxes
		const selectAllCheckbox = document.getElementById('select-all');
		const checkboxes = document.querySelectorAll('.checkbox-item');

		// Function to handle the "Select All" checkbox toggle
		selectAllCheckbox.addEventListener('change', () => {
		checkboxes.forEach(checkbox => {
			checkbox.checked = selectAllCheckbox.checked;
		});
		});

		// Function to update "Select All" based on individual checkbox state
		checkboxes.forEach(checkbox => {
		checkbox.addEventListener('change', () => {
			selectAllCheckbox.checked = [...checkboxes].every(cb => cb.checked);
		});
		});
	</script>
	
	`)

	return nil
}

func HandleAddStudents(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return err
	}

	classroomIDStr := r.FormValue("classroomID")
	classroomID, _ := strconv.Atoi(classroomIDStr)

	studentIDs := r.Form["userID"]

	database.AddStudents(studentIDs, classroomID)

	url := "/classroom?classroom_id="
	url += classroomIDStr
	hxRedirect(w, r, url)
	return nil
}

func HandleClassroomCreate(w http.ResponseWriter, r *http.Request) error {
	createParams := home.CreateParams{
		Classname: r.FormValue("classname"),
		Section:   r.FormValue("section"),
	}

	// TODO: error handling / data cleaning

	err := database.InsertClassroom(w, r)
	if err != nil {
		// if an error occurs
		return render(w, r, home.CreateClassForm(createParams, home.CreateErrors{
			ErrorMessage: err.Error(),
		}))
	}
	hxRedirect(w, r, "/home")
	return nil
}
