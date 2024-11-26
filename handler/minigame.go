package handler

import (
	"errors"
	"fmt"
	"net/http"
	"soln-teachermodule/database"
	"soln-teachermodule/view/minigame"
	"strconv"
)

func HandleMinigameIndex(w http.ResponseWriter, r *http.Request) error {
	minigameIDStr := r.URL.Query().Get("minigameID")
	classroomIDStr := r.URL.Query().Get("classroomID")

	if minigameIDStr == "1" {
		return render(w, r, minigame.Fractions("1", classroomIDStr))
	} else if minigameIDStr == "2" {
		return render(w, r, minigame.Fractions("2", classroomIDStr))
	} else if minigameIDStr == "3" {
		return render(w, r, minigame.Worded("3", classroomIDStr))
	} else if minigameIDStr == "4" {
		return render(w, r, minigame.Worded("4", classroomIDStr))
	} else if minigameIDStr == "5" {
		return render(w, r, minigame.Quiz("5", classroomIDStr))
	} else if minigameIDStr == "6" {
		return render(w, r, minigame.Fractions("6", classroomIDStr))
	} else if minigameIDStr == "7" {
		return render(w, r, minigame.Fractions("7", classroomIDStr))
	} else if minigameIDStr == "8" {
		return render(w, r, minigame.Fractions("8", classroomIDStr))
	} else if minigameIDStr == "9" {
		return render(w, r, minigame.Fractions("9", classroomIDStr))
	} else if minigameIDStr == "10" {
		return render(w, r, minigame.Worded("10", classroomIDStr))
	} else if minigameIDStr == "11" {
		return render(w, r, minigame.Quiz("11", classroomIDStr))
	} else if minigameIDStr == "12" {
		return render(w, r, minigame.Quiz("12", classroomIDStr))
	} else {
		http.Error(w, "invalid minigame id", http.StatusBadRequest)
		return errors.New("bad request")
	}
}

func HandleGetFractions(w http.ResponseWriter, r *http.Request) error {
	minigameIDStr := r.FormValue("minigameID")
	minigameID, _ := strconv.Atoi(minigameIDStr)

	classroomIDStr := r.FormValue("classroomID")
	classroomID, _ := strconv.Atoi(classroomIDStr)

	fmt.Print("minigameID = ", minigameID)
	fmt.Print("classroomID= ", classroomID)

	fractions, err := database.GetFractionQuestions(minigameID, classroomID)
	if err != nil {
		return err
	}

	for _, fraction := range fractions {
		fmt.Fprintf(w, `
			<div class="w-3/5 bg-neutral py-10 px-8 rounded-xl mt-4">
			<div class="flex justify-end">
				<form action="/delete/fractions" method="POST">
					<input type="hidden" name="question_id" value="%d" />
					<input type="hidden" name="minigame_id" value= "%d" />
					<input type="hidden" name="classroom_id" value= "%d" />
					<button type="submit" class="btn btn-danger"><i class="fa-solid fa-trash" style="color: #f66151;"></i></button>
				</form>
			</div>
			<form action="/update/fractions" method="POST">
				<input type="hidden" name="question_id" value= "%d" />
				<input type="hidden" name="minigame_id" value= "%d" />
				<input type="hidden" name="classroom_id" value= "%d" />
				<div class="flex gap-4 mt-4">
					<div class="label mr-4">
						<span class="label-text text-white">Fraction 1 Numerator:</span>
					</div>
					<input type="text" value="%d" name="fraction1_numerator" class="input input-bordered input-primary w-xs text-xl" />
				<div class="label mr-4">
					<span class="label-text text-white">Fraction 2 Numerator</span>
				</div>
					<input type="text" value="%d" name="fraction2_numerator" class="input input-bordered input-primary w-xs text-xl" />
				</div>
				<div class="flex gap-4 mt-4">
					<div class="label">
						<span class="label-text text-white">Fraction 1 Denominator:</span>
					</div>
					<input type="text" value="%d" name="fraction1_denominator" class="input input-bordered input-primary w-xs text-xl" />
				<div class="label">
					<span class="label-text text-white">Fraction 2 Denominator</span>
				</div>
					<input type="text" value="%d" name="fraction2_denominator" class="input input-bordered input-primary w-xs text-xl" />
				</div>

				<div class="flex justify-end">
					<button  type="submit" class="btn btn-primary text-white ">save changes</button>
				</div>
			</div>  	
			</form>
			</div>
		`, fraction.QuestionID, minigameID, classroomID, fraction.QuestionID, minigameID, classroomID, fraction.Fraction1_Numerator, fraction.Fraction2_Numerator, fraction.Fraction1_Denominator, fraction.Fraction2_Denominator)
	}

	return nil
}

func HandleAddFractions(w http.ResponseWriter, r *http.Request) error {
	// get minigameID
	minigameID := r.FormValue("minigameID")
	// get classroomID
	classroomIDStr := r.FormValue("classroomID")
	classroomID, _ := strconv.Atoi(classroomIDStr)

	err := database.AddFractionQuestions(w, r, classroomID)
	if err != nil {
		return err
	}

	hxRedirect(w, r, "/minigame?minigameID="+minigameID+"&classroomID="+classroomIDStr)
	return nil
}

func HandleUpdateFractions(w http.ResponseWriter, r *http.Request) error {
	// get classroomID
	classroomIDStr := r.FormValue("classroom_id")
	// get minigameID
	minigameID := r.FormValue("minigame_id")
	if err := database.UpdateFractions(w, r); err != nil {
		return err
	}

	hxRedirect(w, r, "/minigame?minigameID="+minigameID+"&classroomID="+classroomIDStr)
	return nil
}

func HandleDeleteFractions(w http.ResponseWriter, r *http.Request) error {
	minigameID := r.FormValue("minigame_id")
	questionID := r.FormValue("question_id")
	classroomIDStr := r.FormValue("classroom_id")
	fmt.Print("we got minigameID", minigameID)
	fmt.Print("we got questionID", questionID)
	if err := database.DeleteFractions(minigameID, questionID); err != nil {
		return err
	}
	hxRedirect(w, r, "/minigame?minigameID="+minigameID+"&classroomID="+classroomIDStr)
	return nil
}

func HandleGetWorded(w http.ResponseWriter, r *http.Request) error {
	minigameIDStr := r.FormValue("minigameID")
	minigameID, _ := strconv.Atoi(minigameIDStr)

	classroomIDStr := r.FormValue("classroomID")
	classroomID, _ := strconv.Atoi(classroomIDStr)

	fractions, err := database.GetWordedQuestions(minigameID, classroomID)
	if err != nil {
		return err
	}

	for _, fraction := range fractions {
		fmt.Fprintf(w, `
			<div class="w-3/5 bg-neutral py-10 px-8 rounded-xl mt-4">
			<div class="flex justify-end">
				<form action="/delete/worded" method="POST">
					<input type="hidden" name="question_id" value="%d" />
					<input type="hidden" name="minigameID" value= "%d" />
					<input type="hidden" name="classroomID" value= "%d" />
					<button type="submit" class="btn btn-danger"><i class="fa-solid fa-trash" style="color: #f66151;"></i></button>
				</form>
			</div>
			<form action="/update/worded" method="POST">
				<input type="hidden" name="question_id" value= "%d" />
				<input type="hidden" name="minigameID" value= "%d" />
				<input type="hidden" name="classroomID" value= "%d" />
				<div class="flex gap-4 mt-4 mb-4">
					<div class="label mr-16">
						<span class="label-text text-white">Question Text</span>
					</div>
					<input type="text" value="%s" name="question_text" class="input input-bordered input-primary w-3/4 text-xl" />
				</div>	
				<div class="flex gap-4 mt-4">
					<div class="label mr-3">
						<span class="label-text text-white">Fraction 1 Numerator:</span>
					</div>
					<input type="text" value="%d" name="fraction1_numerator" class="input input-bordered input-primary w-xs text-xl" />
				<div class="label mr-4">
					<span class="label-text text-white">Fraction 2 Numerator</span>
				</div>
					<input type="text" value="%d" name="fraction2_numerator" class="input input-bordered input-primary w-xs text-xl" />
				</div>
				<div class="flex gap-4 mt-4">
					<div class="label">
						<span class="label-text text-white">Fraction 1 Denominator:</span>
					</div>
					<input type="text" value="%d" name="fraction1_denominator" class="input input-bordered input-primary w-xs text-xl" />
				<div class="label">
					<span class="label-text text-white">Fraction 2 Denominator</span>
				</div>
					<input type="text" value="%d" name="fraction2_denominator" class="input input-bordered input-primary w-xs text-xl" />
				</div>

				<div class="flex justify-end">
					<button  type="submit" class="btn btn-primary text-white ">save changes</button>
				</div>
			</div>  	
			</form>
			</div>
		`, fraction.QuestionID, minigameID, classroomID, fraction.QuestionID, minigameID, classroomID, fraction.QuestionText, fraction.Fraction1_Numerator, fraction.Fraction2_Numerator, fraction.Fraction1_Denominator, fraction.Fraction2_Denominator)
	}
	return nil
}

func HandleAddWorded(w http.ResponseWriter, r *http.Request) error {
	// get minigameID
	minigameIDStr := r.FormValue("minigameID")
	// get classroomID
	classroomIDStr := r.FormValue("classroomID")
	classroomID, _ := strconv.Atoi(classroomIDStr)

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
	// get classroomID
	classroomIDStr := r.FormValue("classroomID")
	if err := database.UpdateWordedQuestions(w, r); err != nil {
		return err
	}

	hxRedirect(w, r, "/minigame?minigameID="+minigameIDStr+"&classroomID="+classroomIDStr)
	return nil
}

func HandleDeleteWorded(w http.ResponseWriter, r *http.Request) error {
	minigameIDStr := r.FormValue("minigameID")
	questionIDStr := r.FormValue("question_id")
	classroomIDStr := r.FormValue("classroomID")
	minigameID, _ := strconv.Atoi(minigameIDStr)
	questionID, _ := strconv.Atoi(questionIDStr)
	if err := database.DeleteWorded(minigameID, questionID); err != nil {
		return err
	}
	hxRedirect(w, r, "/minigame?minigameID="+minigameIDStr+"&classroomID="+classroomIDStr)
	return nil
}

func HandleGetMCQuestions(w http.ResponseWriter, r *http.Request) error {
	// get minigameID
	minigameIDStr := r.FormValue("minigameID")
	minigameID, _ := strconv.Atoi(minigameIDStr)
	// get classroomID
	classroomIDStr := r.FormValue("classroomID")
	classroomID, _ := strconv.Atoi(classroomIDStr)

	questions, err := database.GetQuizQuestions(minigameID, classroomID)
	if err != nil {
		return err
	}

	for i, question := range questions {
		fmt.Fprintf(w, `
			<div class="w-3/5 bg-neutral py-10 px-8 rounded-xl mt-4">
			<div class="flex justify-end">
				<form action="/delete/mcquestions" method="POST">
					<input type="hidden" name="questionID" value="%d" />
					<input type="hidden" name="minigameID" value= "%d" />
					<input type="hidden" name="classroomID" value= "%d" />
					<button type="submit" class="btn btn-danger"><i class="fa-solid fa-trash" style="color: #f66151;"></i></button>
				</form>
			</div>
			<form action="/update/mcquestions" method="POST">
				<input type="hidden" name="minigameID" value="%d" />
				<input type="hidden" name="questionID" value= "%d" />
				<input type="hidden" name="classroomID" value= "%d" />
				<span class="label-text text-white">Question %d:</span>
				<input type="text" value="%s" name="question" class="input input-bordered input-primary w-3/4 text-lg" />
				<div class="flex gap-4 mt-4">
					<div class="label">
						<span class="label-text text-white">Option 1:</span>
					</div>
					<input type="text" value="%s" name="option1" class="input input-bordered input-primary w-full max-w-xs text-lg" />
					<input type="hidden"  value="%d" name="option1_choiceID" />
					<div class="label">
						<span class="label-text text-white">Option 2:</span>
					</div>
					<input type="text" value="%s" name="option2" class="input input-bordered input-primary w-full max-w-xs text-lg" />
					<input type="hidden"  value="%d" name="option2_choiceID" />
				</div>
				<div class="flex gap-4 mt-4">
				<div class="label">
					<span class="label-text text-white">Option 3:</span>
				</div>
					<input type="text" value="%s" name="option3" class="input input-bordered input-primary w-full max-w-xs text-lg" />
					<input type="hidden"  value="%d" name="option3_choiceID" />
				<div class="label">
					<span class="label-text text-white">Option 4:</span>
				</div>
					<input type="text" value="%s" name="option4" class="input input-bordered input-primary w-full max-w-xs text-lg" />
					<input type="hidden"  value="%d" name="option4_choiceID" />
				</div>
				<div class="flex mt-4 relative inline-block w-64">
				<div class="label">
					<span class="label-text text-white">Correct Answer: </span>
				</div>
					<select name="correct_answer" class="select select-bordered w-full max-w-xs">
						<option value="%s" %s>Option 1</option> 
						<option value="%s" %s>Option 2</option>
						<option value="%s" %s>Option 3</option>
						<option value="%s" %s>Option 4</option>
					</select>
				</div>

				<div class="flex justify-end">
					<button  type="submit" class="btn btn-primary text-white ">save changes</button>
				</div>
			</div>  	
			</form>
			</div>
		`, question.QuestionID, minigameID, classroomID, minigameID, question.QuestionID, classroomID, i+1, question.QuestionText,
			question.Choices[0].ChoiceText, question.Choices[0].ChoiceID,
			question.Choices[1].ChoiceText, question.Choices[1].ChoiceID,
			question.Choices[2].ChoiceText, question.Choices[2].ChoiceID,
			question.Choices[3].ChoiceText, question.Choices[3].ChoiceID,
			question.Choices[0].ChoiceText, getCorrectAnswer(question.Choices[0].IsCorrect),
			question.Choices[1].ChoiceText, getCorrectAnswer(question.Choices[1].IsCorrect),
			question.Choices[2].ChoiceText, getCorrectAnswer(question.Choices[2].IsCorrect),
			question.Choices[3].ChoiceText, getCorrectAnswer(question.Choices[3].IsCorrect))
	}
	return err
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

	err := database.AddMCQuestions(w, r, classroomID)
	if err != nil {
		return err
	}

	hxRedirect(w, r, "/minigame?minigameID="+minigameIDStr+"&classroomID="+classroomIDStr)
	return nil
}

func HandleUpdateMCQuestions(w http.ResponseWriter, r *http.Request) error {
	minigameIDStr := r.FormValue("minigameID")
	classroomIDStr := r.FormValue("classroomID")
	if err := database.UpdateMCQuestions(w, r); err != nil {
		return err
	}

	hxRedirect(w, r, "/minigame?minigameID="+minigameIDStr+"&classroomID="+classroomIDStr)
	return nil
}

func HandleDeleteMCQuestions(w http.ResponseWriter, r *http.Request) error {
	minigameIDStr := r.FormValue("minigameID")
	questionIDStr := r.FormValue("questionID")
	classroomIDStr := r.FormValue("classroomID")

	minigameID, _ := strconv.Atoi(minigameIDStr)
	questionID, _ := strconv.Atoi(questionIDStr)

	if err := database.DeleteMCQuestions(minigameID, questionID); err != nil {
		return err
	}

	hxRedirect(w, r, "/minigame?minigameID="+minigameIDStr+"&classroomID="+classroomIDStr)
	return nil
}
