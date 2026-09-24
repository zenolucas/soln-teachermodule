package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"soln-teachermodule/database"
	"soln-teachermodule/types"
	"soln-teachermodule/view/classroom"
	"soln-teachermodule/view/ui"
	"strconv"
)

func HandleClassroomIndex(w http.ResponseWriter, r *http.Request) error {
	room := types.Classroom{
		ClassroomID: r.URL.Query().Get("classroom_id"),
	}

	if room.ClassroomID == "" {
		renderErrorPage(w, r, http.StatusBadRequest, "Missing classroom.")
		return errors.New("bad request")
	}

	// convert classroomID to int
	classroomID, _ := strconv.Atoi(room.ClassroomID)

	if err := assertOwnsClassroom(w, r, classroomID); err != nil {
		return err
	}

	// Old bookmarks/links use ?tab=minigames|students - keep them working by sending
	// them on to the new dedicated routes (see T1.1) instead of breaking them outright.
	if tab := r.URL.Query().Get("tab"); tab == "minigames" || tab == "students" {
		http.Redirect(w, r, "/classroom/"+tab+"?classroom_id="+room.ClassroomID, http.StatusFound)
		return nil
	}

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

	// So the page can show which classroom the teacher is looking at (see FE-12) -
	// ownership was already checked above, so this is just fetching the display text.
	fullClassroom, err := database.GetClassroom(r.Context(), classroomID)
	if err != nil {
		return err
	}

	page, err := pageFor(r, fullClassroom.ClassroomName+" · Sol'n Teacher Portal", "overview", room.ClassroomID)
	if err != nil {
		return err
	}

	return render(w, r, classroom.Overview(page, fullClassroom))
}

func HandleClassroomMinigames(w http.ResponseWriter, r *http.Request) error {
	classroomIDStr := r.URL.Query().Get("classroom_id")
	if classroomIDStr == "" {
		renderErrorPage(w, r, http.StatusBadRequest, "Missing classroom.")
		return errors.New("bad request")
	}
	classroomID, _ := strconv.Atoi(classroomIDStr)

	if err := assertOwnsClassroom(w, r, classroomID); err != nil {
		return err
	}

	fullClassroom, err := database.GetClassroom(r.Context(), classroomID)
	if err != nil {
		return err
	}

	page, err := pageFor(r, fullClassroom.ClassroomName+" · Sol'n Teacher Portal", "minigames", classroomIDStr)
	if err != nil {
		return err
	}

	return render(w, r, classroom.Minigames(page, fullClassroom))
}

func HandleClassroomStudents(w http.ResponseWriter, r *http.Request) error {
	classroomIDStr := r.URL.Query().Get("classroom_id")
	if classroomIDStr == "" {
		renderErrorPage(w, r, http.StatusBadRequest, "Missing classroom.")
		return errors.New("bad request")
	}
	classroomID, _ := strconv.Atoi(classroomIDStr)

	if err := assertOwnsClassroom(w, r, classroomID); err != nil {
		return err
	}

	fullClassroom, err := database.GetClassroom(r.Context(), classroomID)
	if err != nil {
		return err
	}

	page, err := pageFor(r, fullClassroom.ClassroomName+" · Sol'n Teacher Portal", "students", classroomIDStr)
	if err != nil {
		return err
	}

	return render(w, r, classroom.Students(page, fullClassroom))
}

func HandleGetClassrooms(w http.ResponseWriter, r *http.Request) error {
	teacherID, err := getTeacherID(r)
	if err != nil {
		return err
	}

	classrooms, err := database.GetClassrooms(r.Context(), teacherID)
	if err != nil {
		return err
	}

	// A brand-new teacher's very first view of the app was blank space with no hint
	// of what to do next - the "+" that opens the create-classroom modal isn't
	// obviously connected to this empty area (see FE-20).
	if len(classrooms) == 0 {
		// The label-for-checkbox toggle this once was has been dead since modals
		// became <dialog> elements in #43 - a for= attribute does nothing on a
		// <dialog>, and it named a ghost id besides. Opening it now goes through
		// the same data-modal-open JS every other modal uses (see X10).
		fmt.Fprint(w, `
		<div class="flex flex-col items-center justify-center w-full py-20 text-white">
			<p class="text-xl mb-4">No classrooms yet.</p>
			<button type="button" class="btn btn-primary text-white" data-modal-open="create-classroom-modal">Create classroom</button>
		</div>
		`)
		return nil
	}

	for _, classroom := range classrooms {
		fmt.Fprintf(w, `
		<div class="glass card card-bordered bg-neutral w-96 shadow-xl h-80 flex justify-center ml-8 mt-8">
				<figure>
					<img src="/public/images/bg/soln-card-image.png" alt="" />
				</figure>
				<div class="card-body">
					<h2 class="card-title">%s - %s</h2>
					<p>%s</p>
					<div class="card-actions justify-end">
						<a href="/classroom?classroom_id=%s" class="btn btn-secondary"> Open </a>
					</div>
				</div>
			</div>
		`, esc(classroom.ClassroomName), esc(classroom.Section), esc(classroom.Description), esc(classroom.ClassroomID))
	}

	return nil
}

func HandleGetClassroomsMenu(w http.ResponseWriter, r *http.Request) error {
	teacherID, err := getTeacherID(r)
	if err != nil {
		return err
	}

	classrooms, err := database.GetClassrooms(r.Context(), teacherID)
	if err != nil {
		return err
	}

	openID := r.URL.Query().Get("classroom_id")
	active := r.URL.Query().Get("active")

	return render(w, r, ui.ClassroomMenu(classrooms, openID, active))
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

	if err := assertOwnsClassroom(w, r, classroomID); err != nil {
		return err
	}

	// fetch users from database
	students, err := database.GetStudents(r.Context(), classroomID)
	if err != nil {
		http.Error(w, "Unable to get students", http.StatusInternalServerError)
		return err
	}

	if len(students) == 0 {
		fmt.Fprint(w, `<tr><td colspan="3" class="text-center text-white text-opacity-60">No students enrolled yet.</td></tr>`)
		return nil
	}

	for i, student := range students {
		fmt.Fprintf(w, `
			<tr id="student-%s">
				<th>%d</th>
				<td>%s %s</td>
				<td class="flex justify-end">
					<a href="/student/score?userID=%s&classroomID=%s" class="btn btn-primary text-white mr-2">
						view scores
					</a>
					<form
						hx-post="/delete/student"
						hx-target="#student-%s"
						hx-swap="outerHTML swap:1s"
						hx-confirm="Are you sure you want to remove this student?"
					>
						<input type="hidden" name="studentID" value="%s" />
						<input type="hidden" name="classroomID" value="%s" />
						<button type="submit" class="btn" aria-label="Remove student"><i class="fa-solid fa-trash" style="color: #f66151;"></i></button>
					</form>
				</td>
			</tr>
		`, esc(student.UserID), i+1, esc(student.Firstname), esc(student.Lastname), esc(student.UserID), esc(classroomIDStr), esc(student.UserID), esc(student.UserID), esc(classroomIDStr))
	}
	return nil
}

func HandleUnenrollStudent(w http.ResponseWriter, r *http.Request) error {
	err := r.ParseForm()
	if err != nil {
		fmt.Println("Error parsing form:", err)
		return err
	}
	studentID, err := formInt(r, "studentID")
	if err != nil {
		return err
	}
	classroomID, err := formInt(r, "classroomID")
	if err != nil {
		return err
	}

	if err := assertOwnsClassroom(w, r, classroomID); err != nil {
		return err
	}

	fmt.Print("we got studentID ")
	if err := database.UnenrollStudent(r.Context(), studentID, classroomID); err != nil {
		return err
	}
	fmt.Print("delete success!")

	// Returning empty content removes the row (because hx-swap="outerHTML")
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "")
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

	if err := assertOwnsClassroom(w, r, ClassroomID); err != nil {
		return err
	}

	students, err := database.GetUnenrolledStudents(r.Context(), ClassroomID)
	if err != nil {
		http.Error(w, "Unable to get students", http.StatusInternalServerError)
		return err
	}

	// Every student was already enrolled - the modal's checkbox table (and its "select
	// all" checkbox) had nothing to show and no explanation why (see FE-20).
	if len(students) == 0 {
		fmt.Fprint(w, `<tr><td colspan="2" class="text-center text-white text-opacity-60">All students are already enrolled in this classroom.</td></tr>`)
		return nil
	}

	// output student array here
	for _, student := range students {
		fmt.Fprintf(w, `
			<tr>
				<td> <input type="checkbox" name="userID" value="%s" class="checkbox-item"/></td>
				<td>%s %s</td>
			</tr>
		`, esc(student.UserID), esc(student.Firstname), esc(student.Lastname))
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

	if err := assertOwnsClassroom(w, r, classroomID); err != nil {
		return err
	}

	studentIDs := r.Form["userID"]

	database.AddStudents(r.Context(), studentIDs, classroomID)

	// Redirect back to the Students route, instead of landing on Overview with no sign
	// the students they just added were actually added (see FE-08).
	url := "/classroom/students?classroom_id=" + classroomIDStr
	hxRedirect(w, r, url)
	return nil
}

func HandleClassroomCreate(w http.ResponseWriter, r *http.Request) error {
	classname := strings.TrimSpace(r.FormValue("classname"))
	section := strings.TrimSpace(r.FormValue("section"))
	description := strings.TrimSpace(r.FormValue("description"))

	createParams := ui.CreateParams{
		Classname:   classname,
		Section:     section,
		Description: description,
	}

	// Client-side `required`/`maxlength` on the form already block most of this, but
	// those are trivially bypassed with a hand-crafted request - without a server-side
	// check here, a blank classname sailed straight into InsertClassroom, and an
	// over-length one hit classrooms.classroom_name VARCHAR(100) as an opaque MySQL
	// truncation error the teacher never saw either way (see FE-02).
	if classname == "" {
		return render(w, r, ui.CreateClassForm(createParams, ui.CreateErrors{
			ErrorMessage: "Class name is required.",
		}))
	}
	if len(classname) > 100 {
		return render(w, r, ui.CreateClassForm(createParams, ui.CreateErrors{
			ErrorMessage: "Class name must be 100 characters or fewer.",
		}))
	}
	if section == "" {
		return render(w, r, ui.CreateClassForm(createParams, ui.CreateErrors{
			ErrorMessage: "Section is required.",
		}))
	}
	if len(section) > 100 {
		return render(w, r, ui.CreateClassForm(createParams, ui.CreateErrors{
			ErrorMessage: "Section must be 100 characters or fewer.",
		}))
	}
	if len(description) > 200 {
		return render(w, r, ui.CreateClassForm(createParams, ui.CreateErrors{
			ErrorMessage: "Description must be 200 characters or fewer.",
		}))
	}

	teacherID, err := getTeacherID(r)
	if err != nil {
		return err
	}

	classroom := types.Classroom{
		ClassroomName: classname,
		Section:       section,
		Description:   description,
	}

	err = database.InsertClassroom(r.Context(), classroom, teacherID)
	if err != nil {
		// The raw DB error (e.g. a driver-level message naming columns/constraints)
		// isn't something a teacher can act on - log it for us and show a generic
		// message instead of leaking it into the form (see FE-02).
		slog.Error("failed to create classroom", "err", err, "teacher_id", teacherID)
		return render(w, r, ui.CreateClassForm(createParams, ui.CreateErrors{
			ErrorMessage: "Couldn't create the classroom. Please try again.",
		}))
	}
	hxRedirect(w, r, "/home")
	return nil
}
