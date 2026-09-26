package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"soln-teachermodule/database"
	"soln-teachermodule/types"
	"soln-teachermodule/util"
	"soln-teachermodule/view/minigame"
	"strconv"
)

func HandleMinigameIndex(w http.ResponseWriter, r *http.Request) error {
	minigameIDStr := r.URL.Query().Get("minigameID")
	classroomIDStr := r.URL.Query().Get("classroomID")

	classroomID, err := strconv.Atoi(classroomIDStr)
	if err != nil {
		renderErrorPage(w, r, http.StatusBadRequest, "That classroom link looks invalid.")
		return err
	}
	if err := assertOwnsClassroom(w, r, classroomID); err != nil {
		return err
	}

	// classroomName and sceneName are for the breadcrumb only (see FE-13) - the page
	// itself already dispatches by minigameKinds[minigameIDStr] below, same as before.
	classroom, err := database.GetClassroom(r.Context(), classroomID)
	if err != nil {
		return err
	}
	minigameID, _ := strconv.Atoi(minigameIDStr)
	scene, _ := types.SceneByID(minigameID)

	var title string
	switch minigameKinds[minigameIDStr] {
	case kindFractions:
		title = "Simple Fraction Questions · Sol'n Teacher Portal"
	case kindWorded:
		title = "Worded Fraction Questions · Sol'n Teacher Portal"
	case kindQuiz:
		title = "Quiz Questions · Sol'n Teacher Portal"
	default:
		renderErrorPage(w, r, http.StatusBadRequest, "That minigame doesn't exist.")
		return errors.New("bad request")
	}

	page, err := pageFor(r, title, "minigames", classroomIDStr)
	if err != nil {
		return err
	}

	switch minigameKinds[minigameIDStr] {
	case kindFractions:
		return render(w, r, minigame.Fractions(page, classroomIDStr, classroom.ClassroomName, scene, worldForScene(minigameID)))
	case kindWorded:
		return render(w, r, minigame.Worded(page, classroomIDStr, classroom.ClassroomName, scene, worldForScene(minigameID)))
	case kindQuiz:
		return render(w, r, minigame.Quiz(page, classroomIDStr, classroom.ClassroomName, scene, worldForScene(minigameID)))
	default:
		renderErrorPage(w, r, http.StatusBadRequest, "That minigame doesn't exist.")
		return errors.New("bad request")
	}
}

// worldForScene finds the World containing minigameID, for the editor header's
// "World N · Topic" badge (T3.4) - types.Scene alone doesn't know which World it
// belongs to, and adding that back-reference isn't worth it for one caller.
func worldForScene(minigameID int) types.World {
	for _, world := range types.Worlds {
		for _, scene := range world.Scenes {
			if scene.MinigameID == minigameID {
				return world
			}
		}
	}
	return types.World{}
}

func HandleGetFractions(w http.ResponseWriter, r *http.Request) error {
	minigameID, err := formInt(r, "minigameID")
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

	rows, err := fractionRows(r.Context(), minigameID, classroomID, false)
	if err != nil {
		return err
	}
	return render(w, r, minigame.FractionRows(rows, 0, false))
}

func HandleGetWorded(w http.ResponseWriter, r *http.Request) error {
	minigameID, err := formInt(r, "minigameID")
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

	rows, err := fractionRows(r.Context(), minigameID, classroomID, true)
	if err != nil {
		return err
	}
	return render(w, r, minigame.FractionRows(rows, 0, true))
}

// fractionRows builds the editor list's rows (T3.4/T3.5): each question's
// stacked-fraction display (or question_text for a worded row), its computed answer,
// and its class accuracy. The list/summary queries are two separate ones per kind
// (02 §B3), joined here by question_id - GetWordedQuestions/GetWordedQuestionSummaries
// read the same fraction_questions table as the fraction pair, just selecting
// question_text as well.
func fractionRows(ctx context.Context, minigameID, classroomID int, worded bool) ([]minigame.FractionRow, error) {
	scene, ok := types.SceneByID(minigameID)
	if !ok {
		return nil, fmt.Errorf("no such minigame: %d", minigameID)
	}

	getQuestions, getSummaries := database.GetFractionQuestions, database.GetFractionQuestionSummaries
	if worded {
		getQuestions, getSummaries = database.GetWordedQuestions, database.GetWordedQuestionSummaries
	}

	questions, err := getQuestions(ctx, minigameID, classroomID)
	if err != nil {
		return nil, err
	}

	summaries, err := getSummaries(ctx, classroomID, minigameID)
	if err != nil {
		return nil, err
	}
	accuracy := make(map[int]types.StudentFractionStatistics, len(summaries))
	for _, s := range summaries {
		accuracy[s.QuestionID] = s
	}

	rows := make([]minigame.FractionRow, 0, len(questions))
	for i, q := range questions {
		row := minigame.FractionRow{
			MinigameID:   minigameID,
			ClassroomID:  classroomID,
			QuestionID:   q.QuestionID,
			Number:       i + 1,
			QuestionText: q.QuestionText,
			Num1:         q.Fraction1_Numerator,
			Den1:         q.Fraction1_Denominator,
			Num2:         q.Fraction2_Numerator,
			Den2:         q.Fraction2_Denominator,
			Op:           scene.Op,
		}
		if answer, ok := util.Combine(util.Frac{Num: q.Fraction1_Numerator, Den: q.Fraction1_Denominator}, util.Frac{Num: q.Fraction2_Numerator, Den: q.Fraction2_Denominator}, scene.Op); ok {
			row.Answer = answer.String()
		}
		if s, ok := accuracy[q.QuestionID]; ok {
			row.AccuracyTotal = s.RightAttemptsCount + s.WrongAttemptsCount
			if row.AccuracyTotal > 0 {
				row.AccuracyPct = s.RightAttemptsCount * 100 / row.AccuracyTotal
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// respondFractionRowsSaved re-renders the whole #question-rows list after a
// successful add/update/delete and sets the HX-Trigger toast event (DEC-4) - sel
// highlights a specific row (the one just edited), or 0 for none (a fresh add or a
// delete has nothing in particular to highlight).
func respondFractionRowsSaved(w http.ResponseWriter, r *http.Request, minigameID, classroomID, sel int, worded bool, event, message string) error {
	rows, err := fractionRows(r.Context(), minigameID, classroomID, worded)
	if err != nil {
		return err
	}
	triggerEvent(w, event, message)
	return render(w, r, minigame.FractionRows(rows, sel, worded))
}

// respondFractionDrawerErrors re-renders the drawer in place with inline field errors
// (DEC-4), as a 422 with HX-Retarget/HX-Reswap so htmx's default (which only swaps 2xx
// responses) doesn't drop it - see the matching htmx:beforeSwap listener in
// public/js/index.js.
func respondFractionDrawerErrors(w http.ResponseWriter, r *http.Request, minigameID, classroomID, questionID int, q types.FractionQuestion, errs map[string]string) error {
	data, err := buildFractionDrawerData(r.Context(), minigameID, classroomID, questionID, q, errs)
	if err != nil {
		return err
	}
	w.Header().Set("HX-Retarget", "#question-drawer")
	w.Header().Set("HX-Reswap", "innerHTML")
	w.WriteHeader(http.StatusUnprocessableEntity)
	return render(w, r, minigame.FractionDrawer(data))
}

func HandleAddFractions(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return err
	}
	minigameID, err := formInt(r, "minigameID")
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

	q, errs := validateFractionForm(r.Form, false)
	if len(errs) > 0 {
		return respondFractionDrawerErrors(w, r, minigameID, classroomID, 0, q, errs)
	}

	if err := database.AddFractionQuestions(w, r, classroomID); err != nil {
		return err
	}

	return respondFractionRowsSaved(w, r, minigameID, classroomID, 0, false, "questionSaved", "Saved ✓")
}

func HandleUpdateFractions(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return err
	}
	classroomID, err := formInt(r, "classroom_id")
	if err != nil {
		return err
	}
	minigameID, err := formInt(r, "minigame_id")
	if err != nil {
		return err
	}
	questionID, err := formInt(r, "question_id")
	if err != nil {
		return err
	}

	if err := assertOwnsClassroom(w, r, classroomID); err != nil {
		return err
	}

	q, errs := validateFractionForm(r.Form, false)
	if len(errs) > 0 {
		return respondFractionDrawerErrors(w, r, minigameID, classroomID, questionID, q, errs)
	}

	if err := database.UpdateFractions(w, r); err != nil {
		return err
	}

	return respondFractionRowsSaved(w, r, minigameID, classroomID, questionID, false, "questionSaved", "Saved ✓")
}

func HandleDeleteFractions(w http.ResponseWriter, r *http.Request) error {
	minigameID, err := formInt(r, "minigame_id")
	if err != nil {
		return err
	}
	questionID, err := formInt(r, "question_id")
	if err != nil {
		return err
	}
	classroomID, err := formInt(r, "classroom_id")
	if err != nil {
		return err
	}

	if err := assertOwnsClassroom(w, r, classroomID); err != nil {
		return err
	}

	if err := database.DeleteFractions(r.Context(), strconv.Itoa(minigameID), strconv.Itoa(questionID), strconv.Itoa(classroomID)); err != nil {
		return err
	}

	return respondFractionRowsSaved(w, r, minigameID, classroomID, 0, false, "questionDeleted", "Question deleted")
}

func HandleAddWorded(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return err
	}
	minigameID, err := formInt(r, "minigameID")
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

	q, errs := validateFractionForm(r.Form, true)
	if len(errs) > 0 {
		return respondFractionDrawerErrors(w, r, minigameID, classroomID, 0, q, errs)
	}

	if err := database.AddWordedQuestions(w, r, classroomID); err != nil {
		return err
	}

	return respondFractionRowsSaved(w, r, minigameID, classroomID, 0, true, "questionSaved", "Saved ✓")
}

func HandleUpdateWorded(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return err
	}
	minigameID, err := formInt(r, "minigameID")
	if err != nil {
		return err
	}
	classroomID, err := formInt(r, "classroomID")
	if err != nil {
		return err
	}
	questionID, err := formInt(r, "questionID")
	if err != nil {
		return err
	}

	if err := assertOwnsClassroom(w, r, classroomID); err != nil {
		return err
	}

	q, errs := validateFractionForm(r.Form, true)
	if len(errs) > 0 {
		return respondFractionDrawerErrors(w, r, minigameID, classroomID, questionID, q, errs)
	}

	if err := database.UpdateWordedQuestions(w, r); err != nil {
		return err
	}

	return respondFractionRowsSaved(w, r, minigameID, classroomID, questionID, true, "questionSaved", "Saved ✓")
}

func HandleDeleteWorded(w http.ResponseWriter, r *http.Request) error {
	minigameID, err := formInt(r, "minigameID")
	if err != nil {
		return err
	}
	questionID, err := formInt(r, "questionID")
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

	if err := database.DeleteWorded(r.Context(), minigameID, questionID, classroomID); err != nil {
		return err
	}

	return respondFractionRowsSaved(w, r, minigameID, classroomID, 0, true, "questionDeleted", "Question deleted")
}

func HandleGetMCQuestions(w http.ResponseWriter, r *http.Request) error {
	minigameID, err := formInt(r, "minigameID")
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

	rows, err := quizRows(r.Context(), minigameID, classroomID)
	if err != nil {
		return err
	}
	return render(w, r, minigame.QuizRows(rows))
}

// quizRows builds the quiz editor list's rows (T3.7): each question's text, its
// correct choice's text, and its class accuracy (GetQuizQuestionAccuracy, 02 §B3).
func quizRows(ctx context.Context, minigameID, classroomID int) ([]minigame.QuizRow, error) {
	questions, err := database.GetQuizQuestions(ctx, minigameID, classroomID)
	if err != nil {
		return nil, err
	}

	accuracy, err := database.GetQuizQuestionAccuracy(ctx, classroomID, minigameID)
	if err != nil {
		return nil, err
	}

	rows := make([]minigame.QuizRow, 0, len(questions))
	number := 0
	for _, q := range questions {
		// A question is always created with exactly 4 choices, but a partially
		// failed insert, a manual DB edit, or choices deleted independently of
		// their question could leave that invariant broken - skip rather than
		// show a row with no correct answer to point to.
		if len(q.Choices) != 4 {
			slog.Warn("skipping MC question row: expected exactly 4 choices",
				"question_id", q.QuestionID, "got", len(q.Choices))
			continue
		}
		number++

		row := minigame.QuizRow{
			MinigameID:   minigameID,
			ClassroomID:  classroomID,
			QuestionID:   q.QuestionID,
			Number:       number,
			QuestionText: q.QuestionText,
		}
		for _, c := range q.Choices {
			if c.IsCorrect {
				row.Answer = c.ChoiceText
				break
			}
		}
		if a, ok := accuracy[q.QuestionID]; ok {
			row.AccuracyTotal = a.Total
			if a.Total > 0 {
				row.AccuracyPct = a.Right * 100 / a.Total
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// respondQuizRowsSaved re-renders the whole #question-rows list after a successful
// add/update/delete and sets the HX-Trigger toast event (DEC-4) - mirrors
// respondFractionRowsSaved (T3.4); QuizRows doesn't take a sel highlight parameter.
func respondQuizRowsSaved(w http.ResponseWriter, r *http.Request, minigameID, classroomID int, event, message string) error {
	rows, err := quizRows(r.Context(), minigameID, classroomID)
	if err != nil {
		return err
	}
	triggerEvent(w, event, message)
	return render(w, r, minigame.QuizRows(rows))
}

// respondQuizDrawerErrors re-renders the drawer in place with inline field errors
// (DEC-4) - mirrors respondFractionDrawerErrors.
func respondQuizDrawerErrors(w http.ResponseWriter, r *http.Request, minigameID, classroomID, questionID int, q types.MultipleChoiceQuestion, errs map[string]string) error {
	data, err := buildQuizDrawerData(r.Context(), minigameID, classroomID, questionID, q, errs)
	if err != nil {
		return err
	}
	w.Header().Set("HX-Retarget", "#question-drawer")
	w.Header().Set("HX-Reswap", "innerHTML")
	w.WriteHeader(http.StatusUnprocessableEntity)
	return render(w, r, minigame.QuizDrawer(data))
}

func HandleAddMCQuestions(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return err
	}
	minigameID, err := formInt(r, "minigameID")
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

	q, errs := validateQuizForm(r.Form, true)
	if len(errs) > 0 {
		return respondQuizDrawerErrors(w, r, minigameID, classroomID, 0, q, errs)
	}

	if err := database.AddMCQuestions(w, r, classroomID); err != nil {
		return err
	}

	// "Add & next" (T3.8): the submitting button's own name=value ("next=1") comes
	// through on r.FormValue like any other field, since it was the activating
	// submitter. HX-Trigger-After-Settle is a separate header from the HX-Trigger the
	// toast uses below - htmx fires it once the swap has settled, so index.js's
	// questionReopenNew listener re-requests a blank drawer only after this response's
	// own #question-rows swap has already landed.
	if r.FormValue("next") == "1" {
		reopenURL := fmt.Sprintf("/question/new?minigameID=%d&classroomID=%d", minigameID, classroomID)
		w.Header().Set("HX-Trigger-After-Settle", fmt.Sprintf(`{"questionReopenNew":{"url":%s}}`, escJS(reopenURL)))
	}

	return respondQuizRowsSaved(w, r, minigameID, classroomID, "questionSaved", "Saved ✓")
}

func HandleUpdateMCQuestions(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return err
	}
	classroomID, err := formInt(r, "classroomID")
	if err != nil {
		return err
	}
	questionID, err := formInt(r, "questionID")
	if err != nil {
		return err
	}

	if err := assertOwnsClassroom(w, r, classroomID); err != nil {
		return err
	}

	// The edit form doesn't post minigameID (it's fixed for an existing question, not
	// something the teacher can change) - GetQuizQuestion hands it back alongside the
	// question itself, the same lookup buildQuizDrawerData uses for the edit drawer.
	_, minigameID, err := database.GetQuizQuestion(r.Context(), questionID, classroomID)
	if err != nil {
		return err
	}

	q, errs := validateQuizForm(r.Form, false)
	if len(errs) > 0 {
		return respondQuizDrawerErrors(w, r, minigameID, classroomID, questionID, q, errs)
	}

	if err := database.UpdateMCQuestions(w, r); err != nil {
		return err
	}

	return respondQuizRowsSaved(w, r, minigameID, classroomID, "questionSaved", "Saved ✓")
}

func HandleDeleteMCQuestions(w http.ResponseWriter, r *http.Request) error {
	minigameID, err := formInt(r, "minigameID")
	if err != nil {
		return err
	}
	questionID, err := formInt(r, "questionID")
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

	if err := database.DeleteMCQuestions(r.Context(), minigameID, questionID, classroomID); err != nil {
		return err
	}

	return respondQuizRowsSaved(w, r, minigameID, classroomID, "questionDeleted", "Question deleted")
}
