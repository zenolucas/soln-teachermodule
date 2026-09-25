package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"soln-teachermodule/database"
	"soln-teachermodule/types"
	"soln-teachermodule/view/classroom"
	"soln-teachermodule/view/home"
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

	insights, err := loadClassroomInsights(r.Context(), classroomID)
	if err != nil {
		return err
	}
	sum := summarize(fullClassroom, insights)

	finished, err := database.GetFinishedScenes(r.Context(), classroomID)
	if err != nil {
		return err
	}
	quiz, err := database.GetLatestQuizScores(r.Context(), classroomID)
	if err != nil {
		return err
	}
	counts, err := database.GetQuestionCounts(r.Context(), classroomID)
	if err != nil {
		return err
	}
	acc, err := database.GetSceneAccuracy(r.Context(), classroomID)
	if err != nil {
		return err
	}
	scenes := buildSceneSummaries(sum.StudentCount, finished, quiz, counts, acc)

	// Top 4 flagged, most flags first then by name - same ordering as Home's Needs
	// attention (T4.6), just sliced shorter and without the cross-classroom context
	// AttentionEntry adds, since this page is already scoped to one classroom.
	var struggling []types.StudentInsight
	for _, in := range insights {
		if len(in.Flags) > 0 {
			struggling = append(struggling, in)
		}
	}
	sort.SliceStable(struggling, func(i, j int) bool {
		if len(struggling[i].Flags) != len(struggling[j].Flags) {
			return len(struggling[i].Flags) > len(struggling[j].Flags)
		}
		if struggling[i].Lastname != struggling[j].Lastname {
			return struggling[i].Lastname < struggling[j].Lastname
		}
		return struggling[i].Firstname < struggling[j].Firstname
	})
	if len(struggling) > 4 {
		struggling = struggling[:4]
	}

	page, err := pageFor(r, fullClassroom.ClassroomName+" · Sol'n Teacher Portal", "overview", room.ClassroomID)
	if err != nil {
		return err
	}

	return render(w, r, classroom.Overview(page, fullClassroom, sum, scenes, struggling))
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

	students, err := database.GetEnrolledStudents(r.Context(), classroomID)
	if err != nil {
		return err
	}
	finished, err := database.GetFinishedScenes(r.Context(), classroomID)
	if err != nil {
		return err
	}
	quiz, err := database.GetLatestQuizScores(r.Context(), classroomID)
	if err != nil {
		return err
	}
	counts, err := database.GetQuestionCounts(r.Context(), classroomID)
	if err != nil {
		return err
	}
	acc, err := database.GetSceneAccuracy(r.Context(), classroomID)
	if err != nil {
		return err
	}
	scenes := buildSceneSummaries(len(students), finished, quiz, counts, acc)

	page, err := pageFor(r, fullClassroom.ClassroomName+" · Sol'n Teacher Portal", "minigames", classroomIDStr)
	if err != nil {
		return err
	}

	return render(w, r, classroom.Minigames(page, fullClassroom, scenes))
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

	// Seeds the toolbar's initial state (T4.10) - e.g. the T4.7 "View all"/"Struggling
	// students" links land here with ?filter=attention.
	q := parseStudentQuery(r.URL.Query())

	return render(w, r, classroom.Students(page, fullClassroom, q.Filter, q.Sort, q.Q))
}

// HandleGetClassrooms renders the Home page's class cards, plus (out of band, in the
// same response - see home.HomeExtras) the header's classroom/student counts and the
// Needs attention list: the top 5 flagged students across every classroom the teacher
// owns, most flags first, then by name (01 §1a).
func HandleGetClassrooms(w http.ResponseWriter, r *http.Request) error {
	teacherID, err := getTeacherID(r)
	if err != nil {
		return err
	}

	classrooms, err := database.GetClassrooms(r.Context(), teacherID)
	if err != nil {
		return err
	}

	summaries := make([]types.ClassroomSummary, 0, len(classrooms))
	var entries []home.AttentionEntry
	studentCount := 0
	for _, room := range classrooms {
		classroomID, err := strconv.Atoi(room.ClassroomID)
		if err != nil {
			return err
		}
		insights, err := loadClassroomInsights(r.Context(), classroomID)
		if err != nil {
			return err
		}
		summaries = append(summaries, summarize(room, insights))
		studentCount += len(insights)

		for _, in := range insights {
			if len(in.Flags) == 0 {
				continue
			}
			entries = append(entries, home.AttentionEntry{
				StudentInsight: in,
				ClassroomID:    room.ClassroomID,
				ClassroomName:  room.ClassroomName,
				FlagLabel:      FlagLabel(in.Flags[0].Kind),
				FlagDetail:     in.Flags[0].Detail,
			})
		}
	}

	sort.SliceStable(entries, func(i, j int) bool {
		if len(entries[i].Flags) != len(entries[j].Flags) {
			return len(entries[i].Flags) > len(entries[j].Flags)
		}
		if entries[i].Lastname != entries[j].Lastname {
			return entries[i].Lastname < entries[j].Lastname
		}
		return entries[i].Firstname < entries[j].Firstname
	})
	if len(entries) > 5 {
		entries = entries[:5]
	}

	seeAllHref := ""
	if len(classrooms) > 0 {
		seeAllHref = fmt.Sprintf("/classroom/students?classroom_id=%s", classrooms[0].ClassroomID)
	}

	// 30, not 6: collapseActivity (T6.2) can merge several of these rows into one
	// "played" event, so asking the query for exactly 6 could collapse down to fewer
	// than the card actually has room for. 30 is generous headroom for that without
	// pulling an unbounded amount of history.
	rawActivity, err := database.GetRecentActivity(r.Context(), teacherID, 30)
	if err != nil {
		return err
	}
	collapsed := collapseActivity(rawActivity)
	if len(collapsed) > 6 {
		collapsed = collapsed[:6]
	}
	now := time.Now()
	activity := make([]home.ActivityEntry, 0, len(collapsed))
	for _, a := range collapsed {
		scene, _ := types.SceneByID(a.MinigameID)
		text := fmt.Sprintf("played %s", sceneLabel(scene))
		if a.Kind == "scored" {
			text = fmt.Sprintf("scored %d/%d on %s", a.Score, a.Total, sceneLabel(scene))
		}
		activity = append(activity, home.ActivityEntry{
			Name:      fmt.Sprintf("%s %s", a.First, a.Last),
			Text:      text,
			ClassName: a.ClassName,
			RelTime:   ui.RelTime(now, a.At),
			Sprite:    scene.Image,
		})
	}

	if err := render(w, r, home.ClassCards(summaries)); err != nil {
		return err
	}
	return render(w, r, home.HomeExtras(len(classrooms), studentCount, entries, seeAllHref, activity))
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

	summaries, err := loadTeacherSummaries(r.Context(), teacherID)
	if err != nil {
		return err
	}
	flagged := make(map[string]int, len(summaries))
	for _, s := range summaries {
		flagged[s.ClassroomID] = s.FlaggedCount
	}

	openID := r.URL.Query().Get("classroom_id")
	active := r.URL.Query().Get("active")

	return render(w, r, ui.ClassroomMenu(classrooms, openID, active, flagged))
}

// HandleGetStudents is the students list fragment (02 §C5): loads every enrolled
// student's insight, then filters/sorts/pages it (handler.applyStudentQuery) into
// the requested view. It's the target of the students-query form's hx-get (T4.10),
// GET or POST (DEC-7) - r.Form covers both a query string and a POST body.
func HandleGetStudents(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return err
	}
	classroomIDStr := r.FormValue("classroomID")

	classroomID, err := strconv.Atoi(classroomIDStr)
	if err != nil {
		return err
	}

	if err := assertOwnsClassroom(w, r, classroomID); err != nil {
		return err
	}

	insights, err := loadClassroomInsights(r.Context(), classroomID)
	if err != nil {
		return err
	}

	q := parseStudentQuery(r.Form)
	page, total, counts := applyStudentQuery(insights, q)

	pageCount := (total + types.StudentsPageSize - 1) / types.StudentsPageSize
	if pageCount < 1 {
		pageCount = 1
	}
	currentPage := q.Page
	if currentPage < 1 {
		currentPage = 1
	}
	if currentPage > pageCount {
		currentPage = pageCount
	}

	return render(w, r, classroom.StudentRows(classroomIDStr, q.Q, q.Filter, q.Sort, page, total, currentPage, pageCount, len(insights), counts))
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

	q := strings.TrimSpace(r.FormValue("q"))
	if len(q) > 50 {
		q = q[:50]
	}

	students, err := database.GetUnenrolledStudents(r.Context(), ClassroomID, q)
	if err != nil {
		return err
	}

	return render(w, r, classroom.UnenrolledRows(students))
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

// validateCreateClassroom is pure so it's cheap to table-test (T2.3): client-side
// `required`/`maxlength` on the form already block most of this, but those are
// trivially bypassed with a hand-crafted request - without a server-side check here, a
// blank classname sailed straight into InsertClassroom, and an over-length one hit
// classrooms.classroom_name VARCHAR(100) as an opaque MySQL truncation error the
// teacher never saw either way (see FE-02). Each field is checked independently, so
// e.g. a blank name and an over-length section both show their own error at once,
// rather than only ever reporting the first problem found.
func validateCreateClassroom(p ui.CreateParams) ui.CreateErrors {
	var errs ui.CreateErrors
	switch {
	case p.Classname == "":
		errs.Classname = "Class name is required."
	case len(p.Classname) > 100:
		errs.Classname = "Class name must be 100 characters or fewer."
	}
	switch {
	case p.Section == "":
		errs.Section = "Section is required."
	case len(p.Section) > 100:
		errs.Section = "Section must be 100 characters or fewer."
	}
	if len(p.Description) > 200 {
		errs.Description = "Description must be 200 characters or fewer."
	}
	return errs
}

func HandleClassroomCreate(w http.ResponseWriter, r *http.Request) error {
	createParams := ui.CreateParams{
		Classname:   strings.TrimSpace(r.FormValue("classname")),
		Section:     strings.TrimSpace(r.FormValue("section")),
		Description: strings.TrimSpace(r.FormValue("description")),
	}

	if errs := validateCreateClassroom(createParams); errs != (ui.CreateErrors{}) {
		return render(w, r, ui.CreateClassForm(createParams, errs))
	}

	teacherID, err := getTeacherID(r)
	if err != nil {
		return err
	}

	classroom := types.Classroom{
		ClassroomName: createParams.Classname,
		Section:       createParams.Section,
		Description:   createParams.Description,
	}

	if err := database.InsertClassroom(r.Context(), classroom, teacherID); err != nil {
		// The raw DB error (e.g. a driver-level message naming columns/constraints)
		// isn't something a teacher can act on - log it for us and show a generic
		// message instead of leaking it into the form (see FE-02).
		slog.Error("failed to create classroom", "err", err, "teacher_id", teacherID)
		return render(w, r, ui.CreateClassForm(createParams, ui.CreateErrors{
			General: "Couldn't create the classroom. Please try again.",
		}))
	}
	hxRedirect(w, r, "/home")
	return nil
}
