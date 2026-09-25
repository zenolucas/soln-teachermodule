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
	page.Charts = true

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
// 01 §1g). The By-question section still loads today's Chart.js fragment
// (LegacyByQuestionFragment) until T5.3 replaces it, which is also why page.Charts
// stays true here.
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

	// T5.3: the By-question section no longer loads the legacy Chart.js fragment, so
	// this page needs neither chart.min.js nor its trailing <script> (see layout/app.templ).
	page.Charts = false

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

	// T5.4: no <canvas> on this page either - the Chart.js per-question charts are
	// replaced by the sortable table.
	page.Charts = false

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

		rows = append(rows, statistics.FractionQuestionStat{
			Number:      numberByID[s.QuestionID],
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
		})
	}
	return rows
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
// stats, and the 12-tile journey. The scene-detail panel starts on the student's
// current scene and swaps on tile click - see student.templ's StudentPage for why
// that panel still uses the old per-kind fragment handlers unrestyled (T5.8's job).
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

	q := parseStudentQuery(r.URL.Query())
	prevID, nextID := neighbours(sortedStudentIDs(insights, q.Sort), studentIDStr)

	title := fmt.Sprintf("%s %s Statistics · Sol'n Teacher Portal", ins.Firstname, ins.Lastname)
	page, err := pageFor(r, title, "students", classroomIDStr)
	if err != nil {
		return err
	}

	return render(w, r, statistics.StudentPage(page, classroomIDStr, classroom, ins, journey, finished, prevID, nextID, q.Sort))
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
	if err := assertEnrolled(w, r, studentID, classroomID); err != nil {
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
	if err := assertEnrolled(w, r, studentID, classroomID); err != nil {
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
