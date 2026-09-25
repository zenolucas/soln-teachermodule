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
	"soln-teachermodule/util"

	// "soln-teachermodule/types"
	"soln-teachermodule/view/layout"
	"soln-teachermodule/view/statistics"
	"soln-teachermodule/view/ui"

	// "github.com/gorilla/sessions"
)

// HandleStatisticsIndex is the old /statistics/fraction and /statistics/quiz route
// (02 §A1). It now just redirects to the merged page instead of rendering anything
// itself, translating classroomID → classroom_id (DEC-6: the old route keeps its own
// param name; the new one uses the new one).
func HandleStatisticsIndex(w http.ResponseWriter, r *http.Request) error {
	classroomIDStr := r.URL.Query().Get("classroomID")
	classroomID, _ := strconv.Atoi(classroomIDStr)

	if err := assertOwnsClassroom(w, r, classroomID); err != nil {
		return err
	}

	minigameID := r.URL.Query().Get("minigameID")
	http.Redirect(w, r, fmt.Sprintf("/classroom/statistics?classroom_id=%s&minigameID=%s", classroomIDStr, minigameID), http.StatusFound)
	return nil
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
		// T5.5: the first scene with any data, not always scene 1 - so a bare
		// /classroom/statistics?classroom_id= link opens something the teacher's
		// class has actually played, when they have.
		firstScene, err := database.FirstSceneWithData(r.Context(), classroomID)
		if err != nil {
			return err
		}
		minigameID = strconv.Itoa(firstScene)
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

	switch minigameKinds[minigameID] {
	case kindFractions:
		return renderFractionStatisticsPage(w, r, page, classroom, classroomID, classroomIDStr, minigameID, minigameIDInt, scene, false)
	case kindWorded:
		return renderFractionStatisticsPage(w, r, page, classroom, classroomID, classroomIDStr, minigameID, minigameIDInt, scene, true)
	case kindQuiz:
		return renderQuizStatisticsPage(w, r, page, classroom, classroomID, classroomIDStr, minigameID, minigameIDInt, scene)
	default:
		renderErrorPage(w, r, http.StatusBadRequest, "That minigame doesn't exist.")
		return errors.New("bad request")
	}
}

// renderQuizStatisticsPage builds and renders the merged quiz statistics page (T5.2,
// 01 §1g). The By-question section renders as CSS bars (T5.3, DEC-11) - no Chart.js
// anywhere on this page.
func renderQuizStatisticsPage(w http.ResponseWriter, r *http.Request, page layout.Page, classroom types.Classroom, classroomID int, classroomIDStr, minigameID string, minigameIDInt int, scene types.Scene) error {
	questionCount, err := database.CountQuizQuestions(r.Context(), minigameIDInt, classroomID)
	if err != nil {
		return err
	}

	scores, err := database.GetQuizStudentScores(r.Context(), classroomID, minigameIDInt)
	if err != nil {
		return err
	}
	sort.SliceStable(scores, func(i, j int) bool { return scores[i].Score < scores[j].Score })

	enrolled, err := database.GetEnrolledStudents(r.Context(), classroomID)
	if err != nil {
		return err
	}

	sum := summarizeQuiz(scores, questionCount, len(enrolled))

	var world types.World
	for _, w := range types.Worlds {
		for _, s := range w.Scenes {
			if s.MinigameID == minigameIDInt {
				world = w
			}
		}
	}

	h := ui.Header{
		Crumbs: []ui.Crumb{
			{Label: classroom.ClassroomName, Href: fmt.Sprintf("/classroom?classroom_id=%s", classroomIDStr)},
			{Label: "Statistics"},
		},
		Title:    scene.Name,
		Subtitle: fmt.Sprintf("World %d · %s · %d questions", world.Number, world.Topic, questionCount),
		Sprite:   scene.Image,
	}

	questions, err := database.GetQuizQuestions(r.Context(), minigameIDInt, classroomID)
	if err != nil {
		return err
	}
	dist, err := database.GetQuizChoiceDistribution(r.Context(), classroomID, minigameIDInt)
	if err != nil {
		return err
	}
	byQuestion := buildQuestionStats(questions, dist)

	return render(w, r, statistics.QuizPage(page, h, classroomIDStr, minigameID, sum, scores, questionCount, byQuestion))
}

// buildQuestionStats composes each quiz question's classroom-wide rollup (T5.3):
// accuracy (the correct choice's share of all responses), an optional misconception
// hint (T4.11's ClassHint), and every choice's tally. A choice's display bar scales to
// the question's most-picked choice, not to its total responses. Number is assigned
// by original question order, before the hardest-first sort, so it stays stable
// regardless of display order.
func buildQuestionStats(questions []types.MultipleChoiceQuestion, dist map[int][]types.ChoiceCount) []statistics.QuestionStat {
	rows := make([]statistics.QuestionStat, 0, len(questions))
	for i, q := range questions {
		counts := dist[q.QuestionID]

		hintChoices := make([]ChoiceStat, len(counts))
		viewChoices := make([]statistics.ChoiceStat, len(counts))
		total, maxCount, correctCount := 0, 0, 0
		for j, c := range counts {
			letter := string(rune('A' + j))
			hintChoices[j] = ChoiceStat{Letter: letter, Text: c.Text, Correct: c.IsCorrect, Count: c.Count}
			viewChoices[j] = statistics.ChoiceStat{Letter: letter, Text: c.Text, IsCorrect: c.IsCorrect, Count: c.Count}
			total += c.Count
			if c.Count > maxCount {
				maxCount = c.Count
			}
			if c.IsCorrect {
				correctCount = c.Count
			}
		}
		for j := range viewChoices {
			viewChoices[j].Pct = pctOfMax(viewChoices[j].Count, maxCount)
		}

		accuracy := -1
		if total > 0 {
			accuracy = correctCount * 100 / total
		}
		hint, _ := ClassHint(q.QuestionText, hintChoices)

		rows = append(rows, statistics.QuestionStat{
			Number:      i + 1,
			Text:        q.QuestionText,
			AccuracyPct: accuracy,
			Hint:        hint,
			Choices:     viewChoices,
		})
	}

	sort.SliceStable(rows, func(i, j int) bool {
		return pctAscNoDataLast(rows[i].AccuracyPct, rows[j].AccuracyPct)
	})
	return rows
}

func pctOfMax(count, max int) int {
	if max == 0 {
		return 0
	}
	return count * 100 / max
}

// renderFractionStatisticsPage builds and renders the merged fraction/worded
// statistics page (T5.4, 01 §2a). worded selects which of B3's summary queries to
// use and whether the By-question table shows question_text or the stacked operands -
// both kinds answer through the same fraction1/2 fields (X9, DEC-21's Scene.Op).
func renderFractionStatisticsPage(w http.ResponseWriter, r *http.Request, page layout.Page, classroom types.Classroom, classroomID int, classroomIDStr, minigameID string, minigameIDInt int, scene types.Scene, worded bool) error {
	var summaries []types.StudentFractionStatistics
	var err error
	if worded {
		summaries, err = database.GetWordedQuestionSummaries(r.Context(), classroomID, minigameIDInt)
	} else {
		summaries, err = database.GetFractionQuestionSummaries(r.Context(), classroomID, minigameIDInt)
	}
	if err != nil {
		return err
	}

	accuracy, err := database.GetStudentAccuracy(r.Context(), classroomID, minigameIDInt)
	if err != nil {
		return err
	}
	enrolled, err := database.GetEnrolledStudents(r.Context(), classroomID)
	if err != nil {
		return err
	}

	questions := buildFractionQuestionStats(summaries, scene.Op, worded)
	students := buildStudentAccuracyRows(accuracy)

	totalRight, totalWrong, belowCount := 0, 0, 0
	for _, s := range summaries {
		totalRight += s.RightAttemptsCount
		totalWrong += s.WrongAttemptsCount
		attempts := s.RightAttemptsCount + s.WrongAttemptsCount
		if attempts > 0 && s.RightAttemptsCount*100 < types.PassPct*attempts {
			belowCount++
		}
	}
	classAccuracy := -1
	if totalRight+totalWrong > 0 {
		classAccuracy = totalRight * 100 / (totalRight + totalWrong)
	}

	var world types.World
	for _, w := range types.Worlds {
		for _, s := range w.Scenes {
			if s.MinigameID == minigameIDInt {
				world = w
			}
		}
	}

	h := ui.Header{
		Crumbs: []ui.Crumb{
			{Label: classroom.ClassroomName, Href: fmt.Sprintf("/classroom?classroom_id=%s", classroomIDStr)},
			{Label: "Statistics"},
		},
		Title:    scene.Name,
		Subtitle: fmt.Sprintf("World %d · %s · %d questions", world.Number, world.Topic, len(summaries)),
		Sprite:   scene.Image,
	}

	sum := statistics.FractionSummary{
		AccuracyPct:   classAccuracy,
		TotalAttempts: totalRight + totalWrong,
		Played:        len(accuracy),
		Enrolled:      len(enrolled),
		BelowCount:    belowCount,
	}

	return render(w, r, statistics.FractionPage(page, h, classroomIDStr, minigameID, sum, questions, students))
}

// buildFractionQuestionStats composes each question's classroom-wide rollup and the
// class's computed answer (util.Combine). Number is assigned by question_id ascending
// before sortWeakestFirst reorders summaries for display, so "Q1" always means the
// same question regardless of how the table is currently sorted - matching the
// mockup's own example, where the displayed rows aren't in Q-number order at all.
func buildFractionQuestionStats(summaries []types.StudentFractionStatistics, op string, worded bool) []statistics.FractionQuestionStat {
	numberByID := map[int]int{}
	byID := append([]types.StudentFractionStatistics(nil), summaries...)
	sort.SliceStable(byID, func(i, j int) bool { return byID[i].QuestionID < byID[j].QuestionID })
	for i, s := range byID {
		numberByID[s.QuestionID] = i + 1
	}

	sortWeakestFirst(summaries)

	rows := make([]statistics.FractionQuestionStat, 0, len(summaries))
	for _, s := range summaries {
		rows = append(rows, fractionStatRow(s, op, worded, numberByID[s.QuestionID]))
	}
	return rows
}

// buildStudentFractionRows is buildFractionQuestionStats' single-student equivalent
// for the scene-detail panel (T5.8, 01 §1h): the same per-question row shape, numbered
// and ordered by question_id ascending - there's no "hardest first" framing for one
// student's own scores, so it skips sortWeakestFirst.
func buildStudentFractionRows(stats []types.StudentFractionStatistics, op string, worded bool) []statistics.FractionQuestionStat {
	sorted := append([]types.StudentFractionStatistics(nil), stats...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].QuestionID < sorted[j].QuestionID })

	rows := make([]statistics.FractionQuestionStat, len(sorted))
	for i, s := range sorted {
		rows[i] = fractionStatRow(s, op, worded, i+1)
	}
	return rows
}

// fractionStatRow builds one question's row (X7's aggregated right/wrong counts, plus
// the class's computed answer via util.Combine) - shared by the classroom-wide
// By-question table and a single student's scene-detail panel, which differ only in
// which rows they pass in and how they're numbered/ordered.
func fractionStatRow(s types.StudentFractionStatistics, op string, worded bool, number int) statistics.FractionQuestionStat {
	a := util.Frac{Num: s.Fraction1_Numerator, Den: s.Fraction1_Denominator}
	b := util.Frac{Num: s.Fraction2_Numerator, Den: s.Fraction2_Denominator}
	ansNum, ansDen := 0, 0
	if ans, ok := util.Combine(a, b, op); ok {
		ansNum, ansDen = ans.Num, ans.Den
	}

	attempts := s.RightAttemptsCount + s.WrongAttemptsCount
	accuracy := -1
	if attempts > 0 {
		accuracy = s.RightAttemptsCount * 100 / attempts
	}

	return statistics.FractionQuestionStat{
		Number:      number,
		IsWorded:    worded,
		Text:        s.QuestionText,
		Num1:        s.Fraction1_Numerator,
		Den1:        s.Fraction1_Denominator,
		Num2:        s.Fraction2_Numerator,
		Den2:        s.Fraction2_Denominator,
		Op:          op,
		AnsNum:      ansNum,
		AnsDen:      ansDen,
		Right:       s.RightAttemptsCount,
		Wrong:       s.WrongAttemptsCount,
		AccuracyPct: accuracy,
	}
}

// buildStudentAccuracyRows converts each student's raw right/wrong sum into a display
// row, sorted lowest-accuracy-first (-1 last, though GetStudentAccuracy's GROUP BY
// means a 0-attempt student never actually appears here).
func buildStudentAccuracyRows(rows []types.StudentAccuracy) []statistics.StudentAccuracyRow {
	view := make([]statistics.StudentAccuracyRow, len(rows))
	for i, s := range rows {
		total := s.Right + s.Wrong
		accuracy := -1
		if total > 0 {
			accuracy = s.Right * 100 / total
		}
		view[i] = statistics.StudentAccuracyRow{First: s.First, Last: s.Last, Right: s.Right, Wrong: s.Wrong, AccuracyPct: accuracy}
	}
	sort.SliceStable(view, func(i, j int) bool { return pctAscNoDataLast(view[i].AccuracyPct, view[j].AccuracyPct) })
	return view
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

// HandleQuizQuestionStatisticsIndex is the old /statistics/quiz/question route
// (02 §A1). It redirects to the merged page's #by-question section - both QuizPage
// (T5.3) and FractionPage (T5.4) render that id, so this works regardless of the
// scene's kind.
func HandleQuizQuestionStatisticsIndex(w http.ResponseWriter, r *http.Request) error {
	minigameIDStr := r.URL.Query().Get("minigameID")
	classroomIDStr := r.URL.Query().Get("classroomID")
	classroomID, _ := strconv.Atoi(classroomIDStr)

	if err := assertOwnsClassroom(w, r, classroomID); err != nil {
		return err
	}

	http.Redirect(w, r, fmt.Sprintf("/classroom/statistics?classroom_id=%s&minigameID=%s#by-question", classroomIDStr, minigameIDStr), http.StatusFound)
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

// assertEnrolled is assertOwnsClassroom's companion for the student statistics
// handlers (X6, T5.6): assertOwnsClassroom only checks the teacher owns
// classroomID, never that studentID is actually enrolled there, so a teacher could
// read any student's name and answers by pairing their own classroomID with a
// foreign userID. Called after assertOwnsClassroom, same as that function 404s
// rather than 500s or silently returning nothing.
func assertEnrolled(w http.ResponseWriter, r *http.Request, studentID, classroomID int) error {
	enrolled, err := database.IsEnrolled(r.Context(), studentID, classroomID)
	if err != nil {
		return err
	}
	if !enrolled {
		http.Error(w, "not found", http.StatusNotFound)
		return fmt.Errorf("student %d is not enrolled in classroom %d", studentID, classroomID)
	}
	return nil
}

// HandleStudentScoreIndex renders the student page (T5.7, 01 §1h): the header, three
// stats, the 12-tile journey, and the Minigame attempts card. The scene-detail panel
// starts on the student's current scene and swaps on tile click, rendering
// QuizDetail/FractionDetail (T5.8) via the /statistics/student/* handlers below.
func HandleStudentScoreIndex(w http.ResponseWriter, r *http.Request) error {
	studentIDStr := r.URL.Query().Get("userID")
	studentID, _ := strconv.Atoi(studentIDStr)
	classroomIDStr := r.URL.Query().Get("classroomID")
	classroomID, _ := strconv.Atoi(classroomIDStr)

	if err := assertOwnsClassroom(w, r, classroomID); err != nil {
		return err
	}
	if err := assertEnrolled(w, r, studentID, classroomID); err != nil {
		return err
	}

	classroom, err := database.GetClassroom(r.Context(), classroomID)
	if err != nil {
		return err
	}

	insights, err := loadClassroomInsights(r.Context(), classroomID)
	if err != nil {
		return err
	}
	var ins types.StudentInsight
	found := false
	for _, in := range insights {
		if in.UserID == studentIDStr {
			ins = in
			found = true
			break
		}
	}
	if !found {
		// assertEnrolled already confirmed the enrollment row exists, so this would
		// mean loadClassroomInsights and the enrollments table disagree - a real bug,
		// not a routine 404, but still safer to surface as "not found" than a panic
		// on a zero-value StudentInsight.
		http.Error(w, "not found", http.StatusNotFound)
		return fmt.Errorf("student %d enrolled but missing from classroom %d's insights", studentID, classroomID)
	}

	quiz, err := database.GetLatestQuizScores(r.Context(), classroomID)
	if err != nil {
		return err
	}
	frac, err := database.GetFractionAggregates(r.Context(), classroomID)
	if err != nil {
		return err
	}
	journey := buildJourney(ins, quiz, frac)
	finished := finishedCount(ins.Current, ins.Completed)
	attempts := buildMinigameAttempts(studentID, frac)

	q := parseStudentQuery(r.URL.Query())
	prevID, nextID := neighbours(sortedStudentIDs(insights, q.Sort), studentIDStr)

	title := fmt.Sprintf("%s %s Statistics · Sol'n Teacher Portal", ins.Firstname, ins.Lastname)
	page, err := pageFor(r, title, "students", classroomIDStr)
	if err != nil {
		return err
	}

	return render(w, r, statistics.StudentPage(page, classroomIDStr, classroom, ins, journey, finished, prevID, nextID, q.Sort, attempts))
}

// buildMinigameAttempts is the student page's Minigame attempts card (01 §1h): one row
// per fraction/worded scene in play order - quiz scenes are scored, not attempted, and
// already summarized by the Quiz average stat card - with 0/0/no-data for a scene the
// student hasn't touched yet.
func buildMinigameAttempts(studentID int, frac []types.FractionAggRow) []statistics.MinigameAttemptRow {
	byScene := map[int]types.FractionAggRow{}
	for _, f := range frac {
		if f.StudentID == studentID {
			byScene[f.MinigameID] = f
		}
	}

	rows := make([]statistics.MinigameAttemptRow, 0, len(sceneOrder))
	for _, id := range sceneOrder {
		scene, _ := types.SceneByID(id)
		if scene.Kind == types.KindQuiz {
			continue
		}
		agg := byScene[id]
		attempts := agg.Right + agg.Wrong
		accuracy := -1
		if attempts > 0 {
			accuracy = agg.Right * 100 / attempts
		}
		rows = append(rows, statistics.MinigameAttemptRow{Scene: scene, Right: agg.Right, Wrong: agg.Wrong, AccuracyPct: accuracy})
	}
	return rows
}

// sortedStudentIDs orders every insight's UserID the same way applyStudentQuery
// (T4.9) sorts for the Students page, but without paging - neighbours (DEC-19) needs
// the whole class's order, not one 10-row page of it.
func sortedStudentIDs(insights []types.StudentInsight, sortKey string) []string {
	sorted := append([]types.StudentInsight(nil), insights...)
	sort.SliceStable(sorted, func(i, j int) bool {
		a, b := sorted[i], sorted[j]
		switch sortKey {
		case "name":
			return studentSortName(a) < studentSortName(b)
		case "quiz":
			return pctAscNoDataLast(a.QuizAvgPct, b.QuizAvgPct)
		case "accuracy":
			return pctAscNoDataLast(a.AccuracyPct, b.AccuracyPct)
		default: // "attention"
			if len(a.Flags) != len(b.Flags) {
				return len(a.Flags) > len(b.Flags)
			}
			return studentSortName(a) < studentSortName(b)
		}
	})
	ids := make([]string, len(sorted))
	for i, s := range sorted {
		ids[i] = s.UserID
	}
	return ids
}

// different score formats
// 1. fraction
// 2. worded
// 3. quiz
// different score formats, rendered via view/statistics' QuizDetail/FractionDetail
// (T5.8) into the student page's #scene-detail panel:
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
	if err := assertEnrolled(w, r, studentID, classroomID); err != nil {
		return err
	}

	scene, _ := types.SceneByID(minigameID)

	// This is a LEFT JOIN from fraction_questions, so a row exists per question even
	// when the student hasn't attempted it (with 0/0 counts) - an empty result here
	// means the minigame itself has no questions yet, not that the student skipped it
	// (see FE-20). FractionDetail shows the same empty-state copy for that case.
	stats, err := database.GetStudentFractionStatistics(r.Context(), studentID, minigameID, classroomID)
	if err != nil {
		return err
	}

	rows := buildStudentFractionRows(stats, scene.Op, false)
	return render(w, r, statistics.FractionDetail(scene, rows))
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
	if err := assertEnrolled(w, r, studentID, classroomID); err != nil {
		return err
	}

	scene, _ := types.SceneByID(minigameID)

	// See the matching comment in HandleGetStudentFractionScore above.
	stats, err := database.GetStudentWordedStatistics(r.Context(), studentID, minigameID, classroomID)
	if err != nil {
		return err
	}

	rows := buildStudentFractionRows(stats, scene.Op, true)
	return render(w, r, statistics.FractionDetail(scene, rows))
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
	if err := assertEnrolled(w, r, studentID, classroomID); err != nil {
		return err
	}

	scene, _ := types.SceneByID(minigameID)

	stats, err := database.GetStudentQuizStatistics(r.Context(), studentID, minigameID, classroomID)
	if err != nil {
		return err
	}

	rows := make([]statistics.QuizDetailRow, len(stats))
	var wrong []struct{ Question, Chosen string }
	score := 0
	for i, s := range stats {
		right := s.Score == 1
		rows[i] = statistics.QuizDetailRow{Question: s.QuestionText, Correct: s.CorrectAnswer, Answer: s.UserAnswer, Right: right}
		if right {
			score++
		} else {
			wrong = append(wrong, struct{ Question, Chosen string }{s.QuestionText, s.UserAnswer})
		}
	}
	// StudentHint (02 §B6, T5.3) fires when >=2 of this student's wrong answers in the
	// quiz match the same misconception rule.
	hint, _ := StudentHint(wrong)

	return render(w, r, statistics.QuizDetail(scene, score, len(stats), hint, rows))
}
