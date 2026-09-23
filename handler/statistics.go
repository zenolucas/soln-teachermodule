package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	// "os"
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

	switch minigameKinds[minigameID] {
	case kindFractions:
		return render(w, r, statistics.FractionStatistics(classroomIDStr, minigameID))
	case kindWorded:
		return render(w, r, statistics.WordedStatistics(classroomIDStr, minigameID))
	case kindQuiz:
		return render(w, r, statistics.QuizStatistics(classroomIDStr, minigameID))
	default:
		renderErrorPage(w, r, http.StatusBadRequest, "That minigame doesn't exist.")
		return errors.New("bad request")
	}
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

	// questionIDs to put into the url parameters on async functions
	var questions []types.FractionQuestion
	questions, err := database.GetFractionQuestions(r.Context(), minigameID, classroomID)
	if err != nil {
		return err
	}

	if len(questions) == 0 {
		renderNoQuestionStatistics(w)
		return nil
	}

	// Each question used to get its own inline <script> defining numbered
	// getClassStatisticsN/renderChartN globals - identical apart from the URL and
	// number. A shared renderer in public/js/index.js now does this for every
	// canvas[data-chart-url] an htmx swap adds to the page (see FE-34).
	for _, question := range questions {
		dataURL := fmt.Sprintf("/statistics/fraction/question/data?questionID=%d&classroomID=%d&minigameID=%d", question.QuestionID, classroomID, minigameID)
		fmt.Fprintf(w, `
			<div class="w-3/5 bg-base-100 py-10 px-8 rounded-xl mt-4 mb-4">
				<div class="text-2xl mt-2 mb-2">Question: %d/%d + %d/%d ?</div>
				<canvas data-chart-type="attempts" data-chart-url="%s" width="300" height="200"></canvas>
			</div>
		`, question.Fraction1_Numerator, question.Fraction1_Denominator, question.Fraction2_Numerator, question.Fraction2_Denominator, esc(dataURL))
	}

	return nil
}

// renderNoQuestionStatistics is shared by the fraction/worded/quiz question-chart
// fragments above - a minigame with no questions yet was rendering a blank statistics
// page with no charts and no explanation why (see FE-20).
func renderNoQuestionStatistics(w http.ResponseWriter) {
	fmt.Fprint(w, `<p class="text-white text-opacity-60 mt-4">This minigame doesn't have any questions yet.</p>`)
}

func HandleFractionResponseStatistics(w http.ResponseWriter, r *http.Request) error {
	classroomIDStr := r.URL.Query().Get("classroomID")
	minigameIDStr := r.URL.Query().Get("minigameID")
	questionIDStr := r.URL.Query().Get("questionID")

	classroomID, _ := strconv.Atoi(classroomIDStr)
	minigameID, _ := strconv.Atoi(minigameIDStr)
	questionID, _ := strconv.Atoi(questionIDStr)

	if err := assertOwnsClassroom(w, r, classroomID); err != nil {
		return err
	}

	statistics, err := database.GetFractionResponseStatistics(r.Context(), classroomID, minigameID, questionID)
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

func HandleWordedQuestionCharts(w http.ResponseWriter, r *http.Request) error {
	minigameIDStr := r.URL.Query().Get("minigameID")
	minigameID, _ := strconv.Atoi(minigameIDStr)

	// get minigameID
	classroomIDStr := r.URL.Query().Get("classroomID")
	classroomID, _ := strconv.Atoi(classroomIDStr)

	if err := assertOwnsClassroom(w, r, classroomID); err != nil {
		return err
	}

	// questionIDs to put into the url parameters on async functions
	var questions []types.FractionQuestion
	questions, err := database.GetWordedQuestions(r.Context(), minigameID, classroomID)
	if err != nil {
		return err
	}

	if len(questions) == 0 {
		renderNoQuestionStatistics(w)
		return nil
	}

	// See the matching comment in HandleFractionQuestionCharts above (FE-34) - this
	// loop used to be nearly identical inline-<script> boilerplate to that one.
	for _, question := range questions {
		dataURL := fmt.Sprintf("/statistics/worded/question/data?questionID=%d&classroomID=%d&minigameID=%d", question.QuestionID, classroomID, minigameID)
		fmt.Fprintf(w, `
			<div class="w-3/5 bg-base-100 py-10 px-8 rounded-xl mt-4 mb-4">
				<div class="text-2xl mt-2 mb-2">Question: %s</div>
				<canvas data-chart-type="attempts" data-chart-url="%s" width="300" height="200"></canvas>
			</div>
		`, esc(question.QuestionText), esc(dataURL))
	}

	return nil
}

func HandleWordedResponseStatistics(w http.ResponseWriter, r *http.Request) error {
	classroomIDStr := r.URL.Query().Get("classroomID")
	minigameIDStr := r.URL.Query().Get("minigameID")
	questionIDStr := r.URL.Query().Get("questionID")

	classroomID, _ := strconv.Atoi(classroomIDStr)
	minigameID, _ := strconv.Atoi(minigameIDStr)
	questionID, _ := strconv.Atoi(questionIDStr)

	if err := assertOwnsClassroom(w, r, classroomID); err != nil {
		return err
	}

	statistics, err := database.GetFractionResponseStatistics(r.Context(), classroomID, minigameID, questionID)
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

	return render(w, r, statistics.QuestionStatistics(minigameIDStr, classroomIDStr))
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
			<div class="w-3/5 bg-base-100 py-10 px-8 rounded-xl mt-4 mb-4">
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

	return render(w, r, statistics.StudentScores(student.Firstname, student.Lastname, studentIDStr, classroomIDStr))
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
