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
		return render(w, r, minigame.Worded(page, minigameIDStr, classroomIDStr, classroom.ClassroomName, scene.Name))
	case kindQuiz:
		return render(w, r, minigame.Quiz(page, minigameIDStr, classroomIDStr, classroom.ClassroomName, scene.Name))
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

	rows, err := fractionRows(r.Context(), minigameID, classroomID)
	if err != nil {
		return err
	}
	return render(w, r, minigame.FractionRows(rows, 0))
}

// fractionRows builds the editor list's rows (T3.4): each question's stacked-fraction
// display, its computed answer, and its class accuracy. GetFractionQuestions and
// GetFractionQuestionSummaries are two separate queries (02 §B3), joined here by
// question_id.
func fractionRows(ctx context.Context, minigameID, classroomID int) ([]minigame.FractionRow, error) {
	scene, ok := types.SceneByID(minigameID)
	if !ok {
		return nil, fmt.Errorf("no such minigame: %d", minigameID)
	}

	questions, err := database.GetFractionQuestions(ctx, minigameID, classroomID)
	if err != nil {
		return nil, err
	}

	summaries, err := database.GetFractionQuestionSummaries(ctx, classroomID, minigameID)
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
			MinigameID:  minigameID,
			ClassroomID: classroomID,
			QuestionID:  q.QuestionID,
			Number:      i + 1,
			Num1:        q.Fraction1_Numerator,
			Den1:        q.Fraction1_Denominator,
			Num2:        q.Fraction2_Numerator,
			Den2:        q.Fraction2_Denominator,
			Op:          scene.Op,
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
func respondFractionRowsSaved(w http.ResponseWriter, r *http.Request, minigameID, classroomID, sel int, event, message string) error {
	rows, err := fractionRows(r.Context(), minigameID, classroomID)
	if err != nil {
		return err
	}
	triggerEvent(w, event, message)
	return render(w, r, minigame.FractionRows(rows, sel))
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

// renderNoQuestions is shared by all three question-list fragments (fractions, worded,
// multiple choice) - each was rendering nothing at all for a freshly-created minigame,
// leaving just the "Add Question" form floating with no indication the game itself has
// no content yet (see FE-20).
func renderNoQuestions(w http.ResponseWriter) {
	fmt.Fprint(w, `<p class="text-white text-opacity-60 mt-4">No questions yet. Add one below to get started.</p>`)
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

	q, errs := validateFractionForm(r.Form)
	if len(errs) > 0 {
		return respondFractionDrawerErrors(w, r, minigameID, classroomID, 0, q, errs)
	}

	if err := database.AddFractionQuestions(w, r, classroomID); err != nil {
		return err
	}

	return respondFractionRowsSaved(w, r, minigameID, classroomID, 0, "questionSaved", "Saved ✓")
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

	q, errs := validateFractionForm(r.Form)
	if len(errs) > 0 {
		return respondFractionDrawerErrors(w, r, minigameID, classroomID, questionID, q, errs)
	}

	if err := database.UpdateFractions(w, r); err != nil {
		return err
	}

	return respondFractionRowsSaved(w, r, minigameID, classroomID, questionID, "questionSaved", "Saved ✓")
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

	return respondFractionRowsSaved(w, r, minigameID, classroomID, 0, "questionDeleted", "Question deleted")
}

func HandleGetWorded(w http.ResponseWriter, r *http.Request) error {
	minigameIDStr := r.FormValue("minigameID")
	minigameID, _ := strconv.Atoi(minigameIDStr)

	classroomIDStr := r.FormValue("classroomID")
	classroomID, _ := strconv.Atoi(classroomIDStr)

	if err := assertOwnsClassroom(w, r, classroomID); err != nil {
		return err
	}

	fractions, err := database.GetWordedQuestions(r.Context(), minigameID, classroomID)
	if err != nil {
		return err
	}

	if len(fractions) == 0 {
		renderNoQuestions(w)
		return nil
	}

	for _, fraction := range fractions {
		renderWordedCard(w, fraction, minigameID, classroomID, false)
	}
	return nil
}

// renderWordedCard is the worded-question equivalent of renderFractionCard above -
// see its comment for why the update form is hx-post now instead of a plain POST
// (FE-22), and why the fields are type="number" required (FE-23).
func renderWordedCard(w http.ResponseWriter, fraction types.FractionQuestion, minigameID int, classroomID int, saved bool) {
	savedText := ""
	if saved {
		savedText = `<span class="text-success mr-2">Saved <i class="fa-solid fa-check"></i></span>`
	}
	// See the matching comment in renderFractionCard above for why this form now
	// targets/swaps itself instead of a wrapping ancestor div.
	fmt.Fprintf(w, `
		<div class="w-full max-w-3xl bg-neutral py-10 px-8 rounded-xl mt-4">
		<div class="flex justify-end">
			<form action="/delete/worded" method="POST" onsubmit="return confirm('Delete this question? Students\' recorded answers to it will also be deleted.')">
				<input type="hidden" name="questionID" value="%d" />
				<input type="hidden" name="minigameID" value= "%d" />
				<input type="hidden" name="classroomID" value= "%d" />
				<button type="submit" class="btn btn-error" aria-label="Delete question"><i class="fa-solid fa-trash"></i></button>
			</form>
		</div>
		<form hx-post="/update/worded" hx-swap="outerHTML">
			<input type="hidden" name="questionID" value= "%d" />
			<input type="hidden" name="minigameID" value= "%d" />
			<input type="hidden" name="classroomID" value= "%d" />
			<div class="flex flex-wrap gap-4 mt-4 mb-4">
				<label class="form-control w-3/4 mr-16">
					<div class="label">
						<span class="label-text text-white">Question Text</span>
					</div>
					<input type="text" value="%s" name="question_text" required class="input input-bordered input-primary w-3/4 text-xl" />
				</label>
			</div>
			<div class="flex flex-wrap gap-4 mt-4">
				<label class="form-control w-xs mr-3">
					<div class="label">
						<span class="label-text text-white">Fraction 1 Numerator:</span>
					</div>
					<input type="number" inputmode="numeric" required min="0" value="%d" name="fraction1_numerator" class="input input-bordered input-primary w-xs text-xl" />
				</label>
				<label class="form-control w-xs mr-4">
					<div class="label">
						<span class="label-text text-white">Fraction 2 Numerator</span>
					</div>
					<input type="number" inputmode="numeric" required min="0" value="%d" name="fraction2_numerator" class="input input-bordered input-primary w-xs text-xl" />
				</label>
			</div>
			<div class="flex flex-wrap gap-4 mt-4">
				<label class="form-control w-xs">
					<div class="label">
						<span class="label-text text-white">Fraction 1 Denominator:</span>
					</div>
					<input type="number" inputmode="numeric" required min="1" value="%d" name="fraction1_denominator" class="input input-bordered input-primary w-xs text-xl" />
				</label>
				<label class="form-control w-xs">
					<div class="label">
						<span class="label-text text-white">Fraction 2 Denominator</span>
					</div>
					<input type="number" inputmode="numeric" required min="1" value="%d" name="fraction2_denominator" class="input input-bordered input-primary w-xs text-xl" />
				</label>
			</div>

			<div class="flex justify-end items-center">
				%s
				<button type="submit" class="btn btn-primary text-white">Save changes</button>
			</div>
		</form>
		</div>
	`, fraction.QuestionID, minigameID, classroomID, fraction.QuestionID, minigameID, classroomID,
		esc(fraction.QuestionText), fraction.Fraction1_Numerator, fraction.Fraction2_Numerator, fraction.Fraction1_Denominator, fraction.Fraction2_Denominator, savedText)
}

func HandleAddWorded(w http.ResponseWriter, r *http.Request) error {
	// get minigameID
	minigameIDStr := r.FormValue("minigameID")
	// get classroomID
	classroomIDStr := r.FormValue("classroomID")
	classroomID, _ := strconv.Atoi(classroomIDStr)

	if err := assertOwnsClassroom(w, r, classroomID); err != nil {
		return err
	}

	err := database.AddWordedQuestions(w, r, classroomID)
	if err != nil {
		return err
	}

	hxRedirect(w, r, "/minigame?minigameID="+minigameIDStr+"&classroomID="+classroomIDStr)
	return nil
}

func HandleUpdateWorded(w http.ResponseWriter, r *http.Request) error {
	// get minigameID here
	minigameIDStr := r.FormValue("minigameID")
	minigameID, _ := strconv.Atoi(minigameIDStr)
	// get classroomID
	classroomIDStr := r.FormValue("classroomID")
	classroomID, _ := strconv.Atoi(classroomIDStr)

	if err := assertOwnsClassroom(w, r, classroomID); err != nil {
		return err
	}

	if err := database.UpdateWordedQuestions(w, r); err != nil {
		return err
	}

	// Re-render just this card instead of redirecting back to the whole /minigame
	// page (see FE-22).
	questionID, err := formInt(r, "questionID")
	if err != nil {
		return err
	}
	fraction1Numerator, err := formInt(r, "fraction1_numerator")
	if err != nil {
		return err
	}
	fraction1Denominator, err := formInt(r, "fraction1_denominator")
	if err != nil {
		return err
	}
	fraction2Numerator, err := formInt(r, "fraction2_numerator")
	if err != nil {
		return err
	}
	fraction2Denominator, err := formInt(r, "fraction2_denominator")
	if err != nil {
		return err
	}

	renderWordedCard(w, types.FractionQuestion{
		QuestionID:            questionID,
		QuestionText:          r.FormValue("question_text"),
		Fraction1_Numerator:   fraction1Numerator,
		Fraction1_Denominator: fraction1Denominator,
		Fraction2_Numerator:   fraction2Numerator,
		Fraction2_Denominator: fraction2Denominator,
	}, minigameID, classroomID, true)
	return nil
}

func HandleDeleteWorded(w http.ResponseWriter, r *http.Request) error {
	minigameIDStr := r.FormValue("minigameID")
	classroomIDStr := r.FormValue("classroomID")
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
	hxRedirect(w, r, "/minigame?minigameID="+minigameIDStr+"&classroomID="+classroomIDStr)
	return nil
}

func HandleGetMCQuestions(w http.ResponseWriter, r *http.Request) error {

	// fmt.Print("GET MC QUESTIONS IS TRIGGERED")
	// get minigameID
	minigameIDStr := r.FormValue("minigameID")
	minigameID, _ := strconv.Atoi(minigameIDStr)
	// get classroomID
	classroomIDStr := r.FormValue("classroomID")
	classroomID, _ := strconv.Atoi(classroomIDStr)

	if err := assertOwnsClassroom(w, r, classroomID); err != nil {
		return err
	}

	questions, err := database.GetQuizQuestions(r.Context(), minigameID, classroomID)
	if err != nil {
		return err
	}

	if len(questions) == 0 {
		renderNoQuestions(w)
		return nil
	}

	for i, question := range questions {
		// A question is always created with exactly 4 choices, but a partially failed
		// insert, a manual DB edit, or choices deleted independently of their question
		// could leave that invariant broken. renderMCCard indexes
		// question.Choices[0..3] unconditionally, so guard against a panic here.
		if len(question.Choices) != 4 {
			slog.Warn("skipping MC question render: expected exactly 4 choices",
				"question_id", question.QuestionID, "got", len(question.Choices))
			continue
		}

		renderMCCard(w, question, minigameID, classroomID, i+1, false)
	}
	return nil
}

// renderMCCard is the multiple-choice equivalent of renderFractionCard above - see
// its comment for why the update form is hx-post now instead of a plain POST (FE-22).
// number is the question's 1-based display position ("Question N:"), passed in
// separately since it isn't part of the question data itself.
func renderMCCard(w http.ResponseWriter, question types.MultipleChoiceQuestion, minigameID int, classroomID int, number int, saved bool) {
	savedText := ""
	if saved {
		savedText = `<span class="text-success mr-2">Saved <i class="fa-solid fa-check"></i></span>`
	}
	// See the matching comment in renderFractionCard above for why this form now
	// targets/swaps itself instead of a wrapping ancestor div.
	fmt.Fprintf(w, `
		<div class="w-full max-w-3xl bg-neutral py-10 px-8 rounded-xl mt-4">
		<div class="flex justify-end">
			<form action="/delete/mcquestions" method="POST" onsubmit="return confirm('Delete this question? Students\' recorded answers to it will also be deleted.')">
				<input type="hidden" name="questionID" value="%d" />
				<input type="hidden" name="minigameID" value= "%d" />
				<input type="hidden" name="classroomID" value= "%d" />
				<button type="submit" class="btn btn-error" aria-label="Delete question"><i class="fa-solid fa-trash"></i></button>
			</form>
		</div>
		<form hx-post="/update/mcquestions" hx-swap="outerHTML">
			<input type="hidden" name="minigameID" value="%d" />
			<input type="hidden" name="questionID" value= "%d" />
			<input type="hidden" name="classroomID" value= "%d" />
			<input type="hidden" name="question_number" value="%d" />
			<label class="form-control w-3/4">
				<div class="label">
					<span class="label-text text-white">Question %d:</span>
				</div>
				<input type="text" value="%s" name="question" required class="input input-bordered input-primary w-3/4 text-lg" />
			</label>
			<div class="flex flex-wrap gap-4 mt-4">
				<label class="form-control w-full max-w-xs">
					<div class="label">
						<span class="label-text text-white">Option 1:</span>
					</div>
					<input type="text" value="%s" name="option1" required maxlength="255" class="input input-bordered input-primary w-full max-w-xs text-lg" />
				</label>
				<input type="hidden"  value="%d" name="option1_choiceID" />
				<label class="form-control w-full max-w-xs">
					<div class="label">
						<span class="label-text text-white">Option 2:</span>
					</div>
					<input type="text" value="%s" name="option2" required maxlength="255" class="input input-bordered input-primary w-full max-w-xs text-lg" />
				</label>
				<input type="hidden"  value="%d" name="option2_choiceID" />
			</div>
			<div class="flex flex-wrap gap-4 mt-4">
				<label class="form-control w-full max-w-xs">
					<div class="label">
						<span class="label-text text-white">Option 3:</span>
					</div>
					<input type="text" value="%s" name="option3" required maxlength="255" class="input input-bordered input-primary w-full max-w-xs text-lg" />
				</label>
				<input type="hidden"  value="%d" name="option3_choiceID" />
				<label class="form-control w-full max-w-xs">
					<div class="label">
						<span class="label-text text-white">Option 4:</span>
					</div>
					<input type="text" value="%s" name="option4" required maxlength="255" class="input input-bordered input-primary w-full max-w-xs text-lg" />
				</label>
				<input type="hidden"  value="%d" name="option4_choiceID" />
			</div>
			<div class="flex mt-4 relative inline-block w-64">
				<label class="form-control w-full max-w-xs">
					<div class="label">
						<span class="label-text text-white">Correct Answer: </span>
					</div>
					<select name="correct_answer" class="select select-bordered w-full max-w-xs">
						<option value="%d" %s>Option 1</option>
						<option value="%d" %s>Option 2</option>
						<option value="%d" %s>Option 3</option>
						<option value="%d" %s>Option 4</option>
					</select>
				</label>
			</div>

			<div class="flex justify-end items-center">
				%s
				<button type="submit" class="btn btn-primary text-white">Save changes</button>
			</div>
		</form>
		</div>
	`, question.QuestionID, minigameID, classroomID, minigameID, question.QuestionID, classroomID, number, number, esc(question.QuestionText),
		esc(question.Choices[0].ChoiceText), question.Choices[0].ChoiceID,
		esc(question.Choices[1].ChoiceText), question.Choices[1].ChoiceID,
		esc(question.Choices[2].ChoiceText), question.Choices[2].ChoiceID,
		esc(question.Choices[3].ChoiceText), question.Choices[3].ChoiceID,
		// correct_answer's <option value> is the choice_id, not the choice text -
		// text can collide between options, choice_id can't (see BUG-02 fix).
		question.Choices[0].ChoiceID, getCorrectAnswer(question.Choices[0].IsCorrect),
		question.Choices[1].ChoiceID, getCorrectAnswer(question.Choices[1].IsCorrect),
		question.Choices[2].ChoiceID, getCorrectAnswer(question.Choices[2].IsCorrect),
		question.Choices[3].ChoiceID, getCorrectAnswer(question.Choices[3].IsCorrect),
		savedText)
}

// helper function to get correct answer for GetMCQuestion function above
func getCorrectAnswer(isCorrect bool) string {
	if isCorrect {
		return "selected"
	}
	return ""
}

func HandleAddMCQuestions(w http.ResponseWriter, r *http.Request) error {
	// get minigameID
	minigameIDStr := r.FormValue("minigameID")
	// get classroomID
	classroomIDStr := r.FormValue("classroomID")
	classroomID, _ := strconv.Atoi(classroomIDStr)

	if err := assertOwnsClassroom(w, r, classroomID); err != nil {
		return err
	}

	err := database.AddMCQuestions(w, r, classroomID)
	if err != nil {
		return err
	}

	hxRedirect(w, r, "/minigame?minigameID="+minigameIDStr+"&classroomID="+classroomIDStr)
	return nil
}

func HandleUpdateMCQuestions(w http.ResponseWriter, r *http.Request) error {
	minigameIDStr := r.FormValue("minigameID")
	minigameID, _ := strconv.Atoi(minigameIDStr)
	classroomIDStr := r.FormValue("classroomID")
	classroomID, _ := strconv.Atoi(classroomIDStr)

	if err := assertOwnsClassroom(w, r, classroomID); err != nil {
		return err
	}

	if err := database.UpdateMCQuestions(w, r); err != nil {
		return err
	}

	// Re-render just this card instead of redirecting back to the whole /minigame
	// page (see FE-22) - reading back the same form fields database.UpdateMCQuestions
	// just validated and saved (including reusing database.ConstructChoices, the same
	// helper it used to build the choices it saved), not a fresh DB query.
	questionID, err := formInt(r, "questionID")
	if err != nil {
		return err
	}
	number, err := formInt(r, "question_number")
	if err != nil {
		return err
	}
	correctAnswerID, err := formInt(r, "correct_answer")
	if err != nil {
		return err
	}
	choices, err := database.ConstructChoices(r, correctAnswerID)
	if err != nil {
		return err
	}

	renderMCCard(w, types.MultipleChoiceQuestion{
		QuestionID:   questionID,
		QuestionText: r.FormValue("question"),
		Choices:      choices,
	}, minigameID, classroomID, number, true)
	return nil
}

func HandleDeleteMCQuestions(w http.ResponseWriter, r *http.Request) error {
	minigameIDStr := r.FormValue("minigameID")
	classroomIDStr := r.FormValue("classroomID")

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

	hxRedirect(w, r, "/minigame?minigameID="+minigameIDStr+"&classroomID="+classroomIDStr)
	return nil
}
