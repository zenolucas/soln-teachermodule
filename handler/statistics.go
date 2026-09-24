package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	// "os"
	"sort"
	"strconv"

	"soln-teachermodule/database"
	"soln-teachermodule/types"

	// "soln-teachermodule/types"
	"soln-teachermodule/view/statistics"

	// "github.com/gorilla/sessions"
)

func HandleStatisticsIndex(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	// get classroomID
	classroomIDStr := r.URL.Query().Get("classroomID")
	classroomID, _ := strconv.Atoi(classroomIDStr)

	if err := assertOwnsClassroom(w, r, classroomID); err != nil {
		return err
	}

	// get minigameID
	minigameID := r.URL.Query().Get("minigameID")

	fmt.Print("loading up statistics, we got minigameID: ", minigameID)

	return renderStatisticsPage(w, r, classroomID, minigameID)
}

// HandleClassroomStatistics is the T1.1-style /classroom/* route for the same page
// (see 02 §A1) - it dispatches exactly like HandleStatisticsIndex, just reached via
// classroom_id instead of classroomID, and with a default minigameID so a bare
// /classroom/statistics?classroom_id= link (e.g. from the future sidebar) has
// something to show instead of erroring on a missing minigame.
func HandleClassroomStatistics(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	classroomIDStr := r.URL.Query().Get("classroom_id")
	classroomID, _ := strconv.Atoi(classroomIDStr)

	if err := assertOwnsClassroom(w, r, classroomID); err != nil {
		return err
	}

	minigameID := r.URL.Query().Get("minigameID")
	if minigameID == "" {
		minigameID = "1"
	}

	return renderStatisticsPage(w, r, classroomID, minigameID)
}

// renderStatisticsPage is the shared body of HandleStatisticsIndex and
// HandleClassroomStatistics - both already called assertOwnsClassroom against
// classroomID before reaching here.
func renderStatisticsPage(w http.ResponseWriter, r *http.Request, classroomID int, minigameID string) error {
	// classroomName and sceneName are for the breadcrumb only (see FE-13) - dispatch
	// below is unchanged.
	classroom, err := database.GetClassroom(r.Context(), classroomID)
	if err != nil {
		return err
	}
	minigameIDInt, _ := strconv.Atoi(minigameID)
	scene, _ := types.SceneByID(minigameIDInt)

	var title string
	switch minigameKinds[minigameID] {
	case kindFractions:
		title = "Simple Fraction Statistics · Sol'n Teacher Portal"
	case kindWorded:
		title = "Worded Questions Statistics · Sol'n Teacher Portal"
	case kindQuiz:
		title = "Quiz Statistics · Sol'n Teacher Portal"
	default:
		renderErrorPage(w, r, http.StatusBadRequest, "That minigame doesn't exist.")
		return errors.New("bad request")
	}

	classroomIDStr := strconv.Itoa(classroomID)
	page, err := pageFor(r, title, "stats", classroomIDStr)
	if err != nil {
		return err
	}
	page.Charts = true

	switch minigameKinds[minigameID] {
	case kindFractions:
		return render(w, r, statistics.FractionStatistics(page, classroomIDStr, minigameID, classroom.ClassroomName, scene.Name))
	case kindWorded:
		return render(w, r, statistics.WordedStatistics(page, classroomIDStr, minigameID, classroom.ClassroomName, scene.Name))
	case kindQuiz:
		return render(w, r, statistics.QuizStatistics(page, classroomIDStr, minigameID, classroom.ClassroomName, scene.Name))
	default:
		renderErrorPage(w, r, http.StatusBadRequest, "That minigame doesn't exist.")
		return errors.New("bad request")
	}
}

// renderNoQuestionStatistics is shared by the fraction/worded/quiz question-chart
// fragments above - a minigame with no questions yet was rendering a blank statistics
// page with no charts and no explanation why (see FE-20).
func renderNoQuestionStatistics(w http.ResponseWriter) {
	fmt.Fprint(w, `<p class="text-white text-opacity-60 mt-4">This minigame doesn't have any questions yet.</p>`)
}

// sortWeakestFirst orders a class-wide question summary by accuracy ascending (see
// FE-30, D6: the audit's chosen replacement for a one-chart-per-question stack was a
// single sortable table, weakest question first, so a teacher sees the questions worth
// re-teaching without scrolling). A question with zero attempts isn't "weak" - there's
// no signal yet either way - so those sort after every question with at least one real
// attempt, rather than tying with (or beating) a genuinely low-accuracy question.
func sortWeakestFirst(summaries []types.StudentFractionStatistics) {
	sort.SliceStable(summaries, func(i, j int) bool {
		iAttempts := summaries[i].RightAttemptsCount + summaries[i].WrongAttemptsCount
		jAttempts := summaries[j].RightAttemptsCount + summaries[j].WrongAttemptsCount
		if (iAttempts == 0) != (jAttempts == 0) {
			return jAttempts == 0
		}
		iPct := float64(summaries[i].RightAttemptsCount) / float64(max(iAttempts, 1))
		jPct := float64(summaries[j].RightAttemptsCount) / float64(max(jAttempts, 1))
		return iPct < jPct
	})
}

// renderQuestionSummaryTable renders the sortable class-wide question table shared by
// HandleFractionQuestionCharts and HandleWordedQuestionCharts below - the same
// question/correct/wrong/%-correct shape either way, just with a fraction expression or
// question text in the first column (see FE-30, D6). Sortable via a small client-side
// helper (public/js/index.js, solnMakeSortable) triggered by the data-sortable
// attribute - clicking a header re-sorts by that column without a server round trip.
func renderQuestionSummaryTable(w http.ResponseWriter, summaries []types.StudentFractionStatistics, questionLabel func(types.StudentFractionStatistics) string) {
	fmt.Fprint(w, `
		<div class="w-full max-w-3xl bg-base-100 py-10 px-8 rounded-xl mt-4 mb-4">
			<table class="table table-zebra text-lg" data-sortable>
				<thead>
					<tr>
						<th data-sort="text">Question</th>
						<th data-sort="number" class="text-center">Correct</th>
						<th data-sort="number" class="text-center">Wrong</th>
						<th data-sort="number" class="text-center">% Correct</th>
					</tr>
				</thead>
				<tbody>
	`)
	for _, summary := range summaries {
		fmt.Fprintf(w, `
					<tr>
						<td>%s</td>
						<td class="text-center">%d</td>
						<td class="text-center">%d</td>
						<td class="text-center">%s</td>
					</tr>
		`, esc(questionLabel(summary)), summary.RightAttemptsCount, summary.WrongAttemptsCount, pctCorrect(summary.RightAttemptsCount, summary.WrongAttemptsCount))
	}
	fmt.Fprint(w, `
				</tbody>
			</table>
		</div>
	`)
}

func HandleFractionQuestionCharts(w http.ResponseWriter, r *http.Request) error {
	// get minigameID
	minigameIDStr := r.URL.Query().Get("minigameID")
	minigameID, _ := strconv.Atoi(minigameIDStr)
	// get classroomID
	classroomIDStr := r.URL.Query().Get("classroomID")
	classroomID, _ := strconv.Atoi(classroomIDStr)

	if err := assertOwnsClassroom(w, r, classroomID); err != nil {
		return err
	}

	summaries, err := database.GetFractionQuestionSummaries(r.Context(), classroomID, minigameID)
	if err != nil {
		return err
	}

	if len(summaries) == 0 {
		renderNoQuestionStatistics(w)
		return nil
	}

	sortWeakestFirst(summaries)
	renderQuestionSummaryTable(w, summaries, func(s types.StudentFractionStatistics) string {
		return fmt.Sprintf("%d/%d + %d/%d ?", s.Fraction1_Numerator, s.Fraction1_Denominator, s.Fraction2_Numerator, s.Fraction2_Denominator)
	})
	return nil
}

func HandleWordedQuestionCharts(w http.ResponseWriter, r *http.Request) error {
	minigameIDStr := r.URL.Query().Get("minigameID")
	minigameID, _ := strconv.Atoi(minigameIDStr)

	// get minigameID
	classroomIDStr := r.URL.Query().Get("classroomID")
	classroomID, _ := strconv.Atoi(classroomIDStr)

	if err := assertOwnsClassroom(w, r, classroomID); err != nil {
		return err
	}

	summaries, err := database.GetWordedQuestionSummaries(r.Context(), classroomID, minigameID)
	if err != nil {
		return err
	}

	if len(summaries) == 0 {
		renderNoQuestionStatistics(w)
		return nil
	}

	sortWeakestFirst(summaries)
	renderQuestionSummaryTable(w, summaries, func(s types.StudentFractionStatistics) string {
		return s.QuestionText
	})
	return nil
}

func HandleQuizClassStatistics(w http.ResponseWriter, r *http.Request) error {
	classroomIDStr := r.URL.Query().Get("classroomID")
	minigameIDStr := r.URL.Query().Get("minigameID")

	fmt.Printf("we got classroomid: %s, and minigameID %s", classroomIDStr, minigameIDStr)

	// convert string to int
	classroomID, _ := strconv.Atoi(classroomIDStr)
	minigameID, _ := strconv.Atoi(minigameIDStr)

	if err := assertOwnsClassroom(w, r, classroomID); err != nil {
		return err
	}

	// Fetch the statistics from the database
	statistics, err := database.GetQuizClassStatistics(r.Context(), classroomID, minigameID)
	if err != nil {
		http.Error(w, "Error retrieving class statistics", http.StatusInternalServerError)
		return err
	}

	// Set headers and send the response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(statistics)
	return nil
}

func HandleQuizQuestionStatisticsIndex(w http.ResponseWriter, r *http.Request) error {
	minigameIDStr := r.URL.Query().Get("minigameID")
	classroomIDStr := r.URL.Query().Get("classroomID")
	classroomID, _ := strconv.Atoi(classroomIDStr)

	if err := assertOwnsClassroom(w, r, classroomID); err != nil {
		return err
	}

	classroom, err := database.GetClassroom(r.Context(), classroomID)
	if err != nil {
		return err
	}
	minigameID, _ := strconv.Atoi(minigameIDStr)
	scene, _ := types.SceneByID(minigameID)

	page, err := pageFor(r, "Question Statistics · Sol'n Teacher Portal", "stats", classroomIDStr)
	if err != nil {
		return err
	}
	page.Charts = true

	return render(w, r, statistics.QuestionStatistics(page, minigameIDStr, classroomIDStr, classroom.ClassroomName, scene.Name))
}

func HandleQuizQuestionCharts(w http.ResponseWriter, r *http.Request) error {
	// get minigameID
	minigameIDStr := r.URL.Query().Get("minigameID")
	minigameID, _ := strconv.Atoi(minigameIDStr)

	// get classroomID
	classroomIDStr := r.URL.Query().Get("classroomID")
	classroomID, _ := strconv.Atoi(classroomIDStr)

	if err := assertOwnsClassroom(w, r, classroomID); err != nil {
		return err
	}

	// questionIDs to put into the url parameters on async functions
	var questions []types.MultipleChoiceQuestion
	questions, err := database.GetQuizQuestions(r.Context(), minigameID, classroomID)
	if err != nil {
		return err
	}

	if len(questions) == 0 {
		renderNoQuestionStatistics(w)
		return nil
	}

	// See the matching comment in HandleFractionQuestionCharts above (FE-34) - this
	// loop used to be nearly identical inline-<script> boilerplate to that one, plus
	// the per-choice colors from setColors embedded as literal JS text.
	for i, question := range questions {
		dataURL := fmt.Sprintf("/statistics/quiz/question/data?questionID=%d&classroomID=%d&minigameID=%d", question.QuestionID, classroomID, minigameID)
		colorsJSON, err := json.Marshal(setColors(question))
		if err != nil {
			return err
		}
		fmt.Fprintf(w, `
			<div class="w-full max-w-3xl bg-base-100 py-10 px-8 rounded-xl mt-4 mb-4">
				<div class="text-2xl mt-2 mb-2">Question %d: %s</div>
				<canvas data-chart-type="choices" data-chart-url="%s" data-colors='%s' width="300" height="200"></canvas>
			</div>
		`, i+1, esc(question.QuestionText), esc(dataURL), colorsJSON)
	}
	return nil
}

// setColors returns one Chart.js background color per choice, in the same order as
// question.Choices, with the correct choice highlighted teal and the rest red - same
// four-way if/else this replaced, just generalized to any choice count and returned
// as data instead of a literal JS snippet, since the chart itself moved out of a
// per-question inline <script> into a shared renderer (see FE-34). Serialized to JSON
// and passed via a data-colors attribute.
func setColors(question types.MultipleChoiceQuestion) []string {
	colors := make([]string, len(question.Choices))
	for i := range colors {
		colors[i] = "rgba(255, 99, 132, 0.5)"
	}
	for i, choice := range question.Choices {
		if choice.IsCorrect {
			colors[i] = "rgba(75, 192, 192, 0.5)"
			break
		}
	}
	return colors
}

func HandleQuizResponseStatistics(w http.ResponseWriter, r *http.Request) error {
	classroomIDStr := r.URL.Query().Get("classroomID")
	minigameIDStr := r.URL.Query().Get("minigameID")
	questionIDStr := r.URL.Query().Get("questionID")

	classroomID, _ := strconv.Atoi(classroomIDStr)
	minigameID, _ := strconv.Atoi(minigameIDStr)
	questionID, _ := strconv.Atoi(questionIDStr)

	if err := assertOwnsClassroom(w, r, classroomID); err != nil {
		return err
	}

	statistics, err := database.GetQuizResponseStatistics(r.Context(), classroomID, minigameID, questionID)
	if err != nil {
		http.Error(w, "Error retrieving class statistics", http.StatusInternalServerError)
		return err
	}

	// Set headers and send the response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(statistics)
	return nil
}

// game functions below

func HandlePostQuizScore(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return nil
	}

	// student_id comes from the token issued at /game/login, not the request body -
	// a body-supplied student_id would let any client post a score for any student.
	studentID, err := authenticateGameRequest(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return err
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return nil
	}
	defer r.Body.Close()

	// StudentID is deliberately not read from the body - see authenticateGameRequest
	// above.
	type Data struct {
		ClassroomID int
		MinigameID  int
		Score       int
	}

	var data Data
	err = json.Unmarshal(body, &data)
	if err != nil {
		http.Error(w, "Failed to parse JSON", http.StatusBadRequest)
		return err
	}

	fmt.Print("we recieved data: ", data)

	type QuizScoreResponse struct {
		Success bool `json:"success"`
	}

	// A score higher than the number of questions in the minigame can't be legitimate -
	// reject it instead of recording a number that will misrepresent this student's
	// results on every chart and leaderboard that reads it back.
	questionCount, err := database.CountQuizQuestions(r.Context(), data.MinigameID, data.ClassroomID)
	if err != nil {
		return err
	}
	if data.Score < 0 || data.Score > questionCount {
		http.Error(w, fmt.Sprintf("invalid score: %d (minigame has %d questions)", data.Score, questionCount), http.StatusBadRequest)
		return fmt.Errorf("invalid score %d for minigame %d with %d questions", data.Score, data.MinigameID, questionCount)
	}

	// record quiz statistics
	err = database.AddQuizStatistics(r.Context(), data.ClassroomID, data.MinigameID, studentID, data.Score)
	if err != nil {
		response := QuizScoreResponse{Success: false}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return err
	}

	response := QuizScoreResponse{Success: true}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
	return nil
}

func HandleQuizResponse(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return nil
	}

	// student_id comes from the token issued at /game/login, not the request body -
	// a body-supplied student_id would let any client record a response as any student.
	studentID, err := authenticateGameRequest(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return err
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return nil
	}
	defer r.Body.Close()

	// StudentID is deliberately not read from the body - see authenticateGameRequest
	// above.
	type Data struct {
		ClassroomID int
		MinigameID  int
		QuestionID  int
		ChoiceID    int
	}

	var data Data
	err = json.Unmarshal(body, &data)
	if err != nil {
		http.Error(w, "Failed to parse JSON", http.StatusBadRequest)
		return err
	}

	fmt.Print("we recieved data: ", data)

	type QuizResponseResult struct {
		Success bool `json:"success"`
	}

	// record quiz statistics
	err = database.AddQuizResponse(r.Context(), data.ClassroomID, data.MinigameID, data.QuestionID, studentID, data.ChoiceID)
	if err != nil {
		response := QuizResponseResult{Success: false}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return err
	}

	response := QuizResponseResult{Success: true}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
	return nil
}

func HandleAddStatisticsFraction(w http.ResponseWriter, r *http.Request) error {
	type StatisticsResponse struct {
		Success bool `json:"success"`
	}

	// student_id comes from the token issued at /game/login, not the request body -
	// a body-supplied student_id would let any client record attempts as any student.
	studentID, err := authenticateGameRequest(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return err
	}

	err = database.AddFractionStatistics(w, r, studentID)
	if err != nil {
		response := StatisticsResponse{Success: false}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return err
	}

	response := StatisticsResponse{Success: true}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
	return nil
}

func HandleGetQuizScores(w http.ResponseWriter, r *http.Request) error {
	var studentScores []types.StudentQuizScore
	minigameIDStr := r.URL.Query().Get("minigameID")
	minigameID, _ := strconv.Atoi(minigameIDStr)

	// get classroomID
	classroomIDStr := r.URL.Query().Get("classroomID")
	classroomID, _ := strconv.Atoi(classroomIDStr)

	if err := assertOwnsClassroom(w, r, classroomID); err != nil {
		return err
	}

	studentScores, err := database.GetStudentScores(r.Context(), classroomID, minigameID)
	if err != nil {
		return err
	}

	// questionCount lets each row show "score / N" instead of a bare number a teacher
	// has to already know the quiz length to make sense of (see FE-33).
	questionCount, err := database.CountQuizQuestions(r.Context(), minigameID, classroomID)
	if err != nil {
		return err
	}

	// This is a JOIN against multiple_choice_scores, not the enrolled-students list -
	// an enrolled student who simply hasn't played this quiz yet doesn't get a row at
	// all, so an empty result here means "no one has played yet," not "no one's
	// enrolled" (see FE-20).
	if len(studentScores) == 0 {
		fmt.Fprint(w, `<tr><td colspan="3" class="text-center text-white text-opacity-60">No scores recorded yet.</td></tr>`)
		return nil
	}

	for i, students := range studentScores {
		fmt.Fprintf(w, `
			<tr>
				<th>%d</th>
				<td>%s %s</td>
				<td class="text-right">%d / %d</td>
			</tr>
		`, i+1, esc(students.FirstName), esc(students.LastName), students.Score, questionCount)
	}

	return nil
}

func HandleStudentScoreIndex(w http.ResponseWriter, r *http.Request) error {
	studentIDStr := r.URL.Query().Get("userID")
	studentID, _ := strconv.Atoi(studentIDStr)
	classroomIDStr := r.URL.Query().Get("classroomID")
	classroomID, _ := strconv.Atoi(classroomIDStr)

	if err := assertOwnsClassroom(w, r, classroomID); err != nil {
		return err
	}

	// get student
	student, err := database.GetStudent(r.Context(), studentID)
	if err != nil {
		return err
	}
	classroom, err := database.GetClassroom(r.Context(), classroomID)
	if err != nil {
		return err
	}

	title := fmt.Sprintf("%s %s Statistics · Sol'n Teacher Portal", student.Firstname, student.Lastname)
	page, err := pageFor(r, title, "students", classroomIDStr)
	if err != nil {
		return err
	}

	return render(w, r, statistics.StudentScores(page, student.Firstname, student.Lastname, studentIDStr, classroomIDStr, classroom.ClassroomName))
}

// different score formats
// 1. fraction
// 2. worded
// 3. quiz
func HandleGetStudentFractionScore(w http.ResponseWriter, r *http.Request) error {
	studentIDStr := r.URL.Query().Get("userID")
	studentID, _ := strconv.Atoi(studentIDStr)
	minigameIDStr := r.URL.Query().Get("minigameID")
	minigameID, _ := strconv.Atoi(minigameIDStr)
	classroomIDStr := r.URL.Query().Get("classroomID")
	classroomID, _ := strconv.Atoi(classroomIDStr)

	if err := assertOwnsClassroom(w, r, classroomID); err != nil {
		return err
	}

	var statistics []types.StudentFractionStatistics

	statistics, err := database.GetStudentFractionStatistics(r.Context(), studentID, minigameID, classroomID)
	if err != nil {
		return err
	}

	// This is a LEFT JOIN from fraction_questions, so a row exists per question even
	// when the student hasn't attempted it (with 0/0 counts) - an empty result here
	// means the minigame itself has no questions yet, not that the student skipped it
	// (see FE-20).
	if len(statistics) == 0 {
		fmt.Fprint(w, `<tr><td colspan="4" class="text-center text-white text-opacity-60">This minigame doesn't have any questions yet.</td></tr>`)
		return nil
	}

	for _, statistic := range statistics {
		fmt.Fprintf(w, `
			<tr>
				<td>%d/%d + %d/%d ?</td>
				<td class="text-center">%d</td>
				<td class="text-center">%d</td>
				<td class="text-center">%s</td>
			</tr>
		`, statistic.Fraction1_Numerator, statistic.Fraction1_Denominator, statistic.Fraction2_Numerator, statistic.Fraction2_Denominator,
			statistic.RightAttemptsCount, statistic.WrongAttemptsCount, pctCorrect(statistic.RightAttemptsCount, statistic.WrongAttemptsCount))
	}

	return nil
}

func HandleGetStudentWordedScore(w http.ResponseWriter, r *http.Request) error {
	studentIDStr := r.URL.Query().Get("userID")
	studentID, _ := strconv.Atoi(studentIDStr)
	minigameIDStr := r.URL.Query().Get("minigameID")
	minigameID, _ := strconv.Atoi(minigameIDStr)
	classroomIDStr := r.URL.Query().Get("classroomID")
	classroomID, _ := strconv.Atoi(classroomIDStr)

	if err := assertOwnsClassroom(w, r, classroomID); err != nil {
		return err
	}

	var statistics []types.StudentFractionStatistics

	statistics, err := database.GetStudentWordedStatistics(r.Context(), studentID, minigameID, classroomID)
	if err != nil {
		return err
	}

	// See the matching comment in HandleGetStudentFractionScore above.
	if len(statistics) == 0 {
		fmt.Fprint(w, `<tr><td colspan="4" class="text-center text-white text-opacity-60">This minigame doesn't have any questions yet.</td></tr>`)
		return nil
	}

	for _, statistic := range statistics {
		fmt.Fprintf(w, `
			<tr>
				<td>%s</td>
				<td class="text-center">%d</td>
				<td class="text-center">%d</td>
				<td class="text-center">%s</td>
			</tr>
		`, esc(statistic.QuestionText), statistic.RightAttemptsCount, statistic.WrongAttemptsCount, pctCorrect(statistic.RightAttemptsCount, statistic.WrongAttemptsCount))
	}

	return nil
}

func HandleGetStudentQuizScore(w http.ResponseWriter, r *http.Request) error {
	studentIDStr := r.URL.Query().Get("userID")
	studentID, _ := strconv.Atoi(studentIDStr)
	minigameIDStr := r.URL.Query().Get("minigameID")
	minigameID, _ := strconv.Atoi(minigameIDStr)
	classroomIDStr := r.URL.Query().Get("classroomID")
	classroomID, _ := strconv.Atoi(classroomIDStr)

	if err := assertOwnsClassroom(w, r, classroomID); err != nil {
		return err
	}

	var statistics []types.StudentQuizStatistics

	statistics, err := database.GetStudentQuizStatistics(r.Context(), studentID, minigameID, classroomID)
	if err != nil {
		return err
	}

	// See the matching comment in HandleGetStudentFractionScore above.
	if len(statistics) == 0 {
		fmt.Fprint(w, `<tr><td colspan="4" class="text-center text-white text-opacity-60">This minigame doesn't have any questions yet.</td></tr>`)
		return nil
	}

	for _, statistic := range statistics {
		fmt.Fprintf(w, `
			<tr>
				<td>%s</td>
				<td class="text-center">%s</td>
				<td class="text-center">%s</td>
				<td class="text-center">%d</td>
			</tr>	
		`, esc(statistic.QuestionText), esc(statistic.CorrectAnswer), esc(statistic.UserAnswer), statistic.Score)
	}

	return nil
}
