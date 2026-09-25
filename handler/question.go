package handler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"soln-teachermodule/database"
	"soln-teachermodule/types"
	"soln-teachermodule/util"
	"soln-teachermodule/view/minigame"
)

// HandleQuestionNew renders an empty drawer for "+ Add question" - FractionDrawer for
// fraction/worded (T3.4/T3.5), QuizDrawer for quiz (T3.7).
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

	switch minigameKinds[strconv.Itoa(minigameID)] {
	case kindFractions, kindWorded:
		data, err := buildFractionDrawerData(r.Context(), minigameID, classroomID, 0, types.FractionQuestion{}, nil)
		if err != nil {
			return err
		}
		return render(w, r, minigame.FractionDrawer(data))
	case kindQuiz:
		data, err := buildQuizDrawerData(r.Context(), minigameID, classroomID, 0, types.MultipleChoiceQuestion{}, nil)
		if err != nil {
			return err
		}
		return render(w, r, minigame.QuizDrawer(data))
	default:
		renderErrorPage(w, r, http.StatusBadRequest, "That question type isn't supported yet.")
		return errors.New("unsupported minigame kind for /question/new")
	}
}

// HandleQuestionEdit renders a drawer pre-filled with an existing question, for a list
// row's hx-get - FractionDrawer for fraction/worded (T3.4/T3.5), QuizDrawer for quiz
// (T3.7).
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

	switch minigameKinds[strconv.Itoa(minigameID)] {
	case kindFractions, kindWorded:
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
	case kindQuiz:
		q, _, err := database.GetQuizQuestion(r.Context(), questionID, classroomID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				renderErrorPage(w, r, http.StatusNotFound, "That question doesn't exist.")
				return err
			}
			return err
		}
		data, err := buildQuizDrawerData(r.Context(), minigameID, classroomID, questionID, q, nil)
		if err != nil {
			return err
		}
		return render(w, r, minigame.QuizDrawer(data))
	default:
		renderErrorPage(w, r, http.StatusBadRequest, "That question type isn't supported yet.")
		return errors.New("unsupported minigame kind for /question/edit")
	}
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
		MinigameID:   minigameID,
		ClassroomID:  classroomID,
		QuestionID:   questionID,
		SceneName:    scene.Name,
		Sprite:       scene.Image,
		Op:           scene.Op,
		Operation:    operationLabel(scene.Op),
		Worded:       scene.Kind == types.KindWorded,
		QuestionText: q.QuestionText,
		Num1:         q.Fraction1_Numerator,
		Den1:         q.Fraction1_Denominator,
		Num2:         q.Fraction2_Numerator,
		Den2:         q.Fraction2_Denominator,
		Errors:       errs,
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

// validateFractionForm reads and validates a fraction/worded question's fields from
// vals - the same field names for both the add and update forms (X4). Pure over
// url.Values so it's cheap to table-test (question_test.go). The error map is keyed by
// field name, for the drawer's inline per-field errors (DEC-4). A field left
// unparseable is 0 in the returned question; one that parses but fails its range check
// (e.g. a denominator of 0) still gets its literal value, so a rejected submission
// redisplays exactly what the teacher typed rather than silently resetting it.
// requireText additionally requires a non-blank "question_text" (T3.5's worded forms;
// fraction forms don't have that field at all, so pass false there).
func validateFractionForm(vals url.Values, requireText bool) (types.FractionQuestion, map[string]string) {
	errs := map[string]string{}
	q := types.FractionQuestion{
		Fraction1_Numerator:   validateFractionField(vals, "fraction1_numerator", 0, errs),
		Fraction1_Denominator: validateFractionField(vals, "fraction1_denominator", 1, errs),
		Fraction2_Numerator:   validateFractionField(vals, "fraction2_numerator", 0, errs),
		Fraction2_Denominator: validateFractionField(vals, "fraction2_denominator", 1, errs),
	}
	if requireText {
		q.QuestionText = vals.Get("question_text")
		if strings.TrimSpace(q.QuestionText) == "" {
			errs["question_text"] = "Question text can't be blank."
		}
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

// buildQuizDrawerData assembles everything QuizDrawer needs - mirrors
// buildFractionDrawerData (T3.4). q is the values to show: a freshly fetched question
// when editing, the zero-value MultipleChoiceQuestion for a new one, or the teacher's
// just-submitted values when errs is non-empty (DEC-4).
func buildQuizDrawerData(ctx context.Context, minigameID, classroomID, questionID int, q types.MultipleChoiceQuestion, errs map[string]string) (minigame.QuizDrawerData, error) {
	scene, ok := types.SceneByID(minigameID)
	if !ok {
		return minigame.QuizDrawerData{}, fmt.Errorf("no such minigame: %d", minigameID)
	}

	data := minigame.QuizDrawerData{
		MinigameID:   minigameID,
		ClassroomID:  classroomID,
		QuestionID:   questionID,
		SceneName:    scene.Name,
		QuestionText: q.QuestionText,
		Errors:       errs,
	}
	for i := 0; i < 4 && i < len(q.Choices); i++ {
		data.Options[i] = minigame.QuizOption{
			Text:     q.Choices[i].ChoiceText,
			ChoiceID: q.Choices[i].ChoiceID,
			Correct:  q.Choices[i].IsCorrect,
		}
	}

	if questionID == 0 {
		return data, nil
	}
	data.Editing = true

	// RowNumber must skip malformed (not-exactly-4-choices) questions the same way
	// quizRows does, so "Edit question N" matches the row the teacher actually clicked.
	all, err := database.GetQuizQuestions(ctx, minigameID, classroomID)
	if err != nil {
		return minigame.QuizDrawerData{}, err
	}
	number := 0
	for _, mq := range all {
		if len(mq.Choices) != 4 {
			continue
		}
		number++
		if mq.QuestionID == questionID {
			data.RowNumber = number
			break
		}
	}

	return data, nil
}

// validateQuizForm reads and validates a quiz question's text and 4 choices from vals.
// Field names differ between new and edit mode (X4; quiz.templ's quizOptionField/
// quizRadioValue mirror this exactly): new mode uses question_text/option_1..4/
// correct_answer=option_N, edit mode uses question/option1..4/correct_answer=choice_id.
// Pure over url.Values so it's cheap to table-test (question_test.go).
func validateQuizForm(vals url.Values, isNew bool) (types.MultipleChoiceQuestion, map[string]string) {
	errs := map[string]string{}
	var q types.MultipleChoiceQuestion

	textField := "question"
	if isNew {
		textField = "question_text"
	}
	q.QuestionText = vals.Get(textField)
	if strings.TrimSpace(q.QuestionText) == "" {
		errs[textField] = "Question text can't be blank."
	} else if len(q.QuestionText) > 500 {
		errs[textField] = "Question text must be 500 characters or fewer."
	}

	correctAnswer := vals.Get("correct_answer")
	haveCorrect := false
	q.Choices = make([]types.Choice, 4)
	for i := 0; i < 4; i++ {
		field := fmt.Sprintf("option_%d", i+1)
		if !isNew {
			field = fmt.Sprintf("option%d", i+1)
		}
		text := vals.Get(field)
		q.Choices[i].ChoiceText = text
		if strings.TrimSpace(text) == "" {
			errs[field] = "Option text can't be blank."
		} else if len(text) > 255 {
			errs[field] = "Option text must be 255 characters or fewer."
		}

		if isNew {
			q.Choices[i].IsCorrect = correctAnswer == fmt.Sprintf("option_%d", i+1)
		} else {
			choiceID, _ := strconv.Atoi(vals.Get(fmt.Sprintf("option%d_choiceID", i+1)))
			q.Choices[i].ChoiceID = choiceID
			q.Choices[i].IsCorrect = correctAnswer != "" && strconv.Itoa(choiceID) == correctAnswer
		}
		if q.Choices[i].IsCorrect {
			haveCorrect = true
		}
	}
	if !haveCorrect {
		errs["correct_answer"] = "Choose the correct answer."
	}

	return q, errs
}
