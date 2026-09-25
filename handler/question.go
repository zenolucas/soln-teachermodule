package handler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"soln-teachermodule/database"
	"soln-teachermodule/types"
	"soln-teachermodule/util"
	"soln-teachermodule/view/minigame"
)

// HandleQuestionNew renders an empty FractionDrawer for "+ Add question" (T3.4). Only
// the fraction kind is handled here - worded (T3.5) and quiz (T3.7) share the route
// space by minigame ID but aren't wired up yet, so any other kind is a 400 rather than
// silently rendering the wrong drawer.
func HandleQuestionNew(w http.ResponseWriter, r *http.Request) error {
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
	if minigameKinds[strconv.Itoa(minigameID)] != kindFractions {
		renderErrorPage(w, r, http.StatusBadRequest, "That question type isn't supported yet.")
		return errors.New("unsupported minigame kind for /question/new")
	}

	data, err := buildFractionDrawerData(r.Context(), minigameID, classroomID, 0, types.FractionQuestion{}, nil)
	if err != nil {
		return err
	}
	return render(w, r, minigame.FractionDrawer(data))
}

// HandleQuestionEdit renders a FractionDrawer pre-filled with an existing question,
// for a list row's hx-get (T3.4). See HandleQuestionNew for why only kindFractions is
// handled.
func HandleQuestionEdit(w http.ResponseWriter, r *http.Request) error {
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
	if minigameKinds[strconv.Itoa(minigameID)] != kindFractions {
		renderErrorPage(w, r, http.StatusBadRequest, "That question type isn't supported yet.")
		return errors.New("unsupported minigame kind for /question/edit")
	}

	q, _, err := database.GetFractionQuestion(r.Context(), questionID, classroomID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			renderErrorPage(w, r, http.StatusNotFound, "That question doesn't exist.")
			return err
		}
		return err
	}

	data, err := buildFractionDrawerData(r.Context(), minigameID, classroomID, questionID, q, nil)
	if err != nil {
		return err
	}
	return render(w, r, minigame.FractionDrawer(data))
}

// buildFractionDrawerData assembles everything FractionDrawer needs. q is the values
// to show in the Problem builder: a freshly fetched question when editing, the
// zero-value FractionQuestion for a new one, or the teacher's just-submitted values
// when errs is non-empty and the drawer is redisplaying a validation failure (DEC-4).
// RowNumber and the accuracy callout are only looked up when questionID != 0, since a
// new, unsaved question has neither.
func buildFractionDrawerData(ctx context.Context, minigameID, classroomID, questionID int, q types.FractionQuestion, errs map[string]string) (minigame.FractionDrawerData, error) {
	scene, ok := types.SceneByID(minigameID)
	if !ok {
		return minigame.FractionDrawerData{}, fmt.Errorf("no such minigame: %d", minigameID)
	}

	data := minigame.FractionDrawerData{
		MinigameID:  minigameID,
		ClassroomID: classroomID,
		QuestionID:  questionID,
		SceneName:   scene.Name,
		Op:          scene.Op,
		Operation:   operationLabel(scene.Op),
		Num1:        q.Fraction1_Numerator,
		Den1:        q.Fraction1_Denominator,
		Num2:        q.Fraction2_Numerator,
		Den2:        q.Fraction2_Denominator,
		Errors:      errs,
	}
	if answer, ok := util.Combine(util.Frac{Num: q.Fraction1_Numerator, Den: q.Fraction1_Denominator}, util.Frac{Num: q.Fraction2_Numerator, Den: q.Fraction2_Denominator}, scene.Op); ok {
		data.Answer = answer.String()
	}

	if questionID == 0 {
		return data, nil
	}
	data.Editing = true

	all, err := database.GetFractionQuestions(ctx, minigameID, classroomID)
	if err != nil {
		return minigame.FractionDrawerData{}, err
	}
	for i, fq := range all {
		if fq.QuestionID == questionID {
			data.RowNumber = i + 1
			break
		}
	}

	summaries, err := database.GetFractionQuestionSummaries(ctx, classroomID, minigameID)
	if err != nil {
		return minigame.FractionDrawerData{}, err
	}
	for _, s := range summaries {
		if s.QuestionID != questionID {
			continue
		}
		total := s.RightAttemptsCount + s.WrongAttemptsCount
		if total > 0 {
			data.ShowAccuracy = true
			data.AccuracyRight = s.RightAttemptsCount
			data.AccuracyTotal = total
		}
		break
	}

	return data, nil
}

// operationLabel is a small local mapping from Scene.Op to the drawer subtitle's
// operation name - kept here rather than on types.Scene since types/scene.go isn't
// among this task's files.
func operationLabel(op string) string {
	switch op {
	case "+":
		return "Addition"
	case "-", "−":
		return "Subtraction"
	default:
		return ""
	}
}

// validateFractionForm reads and validates a fraction question's numerator/
// denominator fields from vals - the same field names for both the add and update
// forms (X4). Pure over url.Values so it's cheap to table-test (question_test.go). The
// error map is keyed by field name, for the drawer's inline per-field errors (DEC-4).
// A field left unparseable is 0 in the returned question; one that parses but fails
// its range check (e.g. a denominator of 0) still gets its literal value, so a
// rejected submission redisplays exactly what the teacher typed rather than silently
// resetting it.
func validateFractionForm(vals url.Values) (types.FractionQuestion, map[string]string) {
	errs := map[string]string{}
	q := types.FractionQuestion{
		Fraction1_Numerator:   validateFractionField(vals, "fraction1_numerator", 0, errs),
		Fraction1_Denominator: validateFractionField(vals, "fraction1_denominator", 1, errs),
		Fraction2_Numerator:   validateFractionField(vals, "fraction2_numerator", 0, errs),
		Fraction2_Denominator: validateFractionField(vals, "fraction2_denominator", 1, errs),
	}
	return q, errs
}

// validateFractionField parses one numeric field and requires it be >= min, recording
// an error keyed by field's own name when it isn't a whole number or falls short.
func validateFractionField(vals url.Values, field string, min int, errs map[string]string) int {
	msg := fmt.Sprintf("Must be a whole number, %d or more.", min)
	n, err := strconv.Atoi(vals.Get(field))
	if err != nil {
		errs[field] = msg
		return 0
	}
	if n < min {
		errs[field] = msg
	}
	return n
}

// triggerEvent sets an HX-Trigger response header naming a client-side event (DEC-4) -
// "questionSaved" or "questionDeleted", each carrying {message} for the toast (see
// public/js/index.js). escJS keeps the message safe to embed in a header value
// regardless of what characters it contains.
func triggerEvent(w http.ResponseWriter, event, message string) {
	w.Header().Set("HX-Trigger", fmt.Sprintf(`{%s:{"message":%s}}`, escJS(event), escJS(message)))
}
