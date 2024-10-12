package handler

import (
	"fmt"
	"net/http"
	"soln-teachermodule/database"
	"soln-teachermodule/view/minigame"
	"strconv"
)

func HandleMinigame1Index(w http.ResponseWriter, r *http.Request) error {
	return render(w, r, minigame.Fractions("1"))
}

func HandleGetFractions(w http.ResponseWriter, r *http.Request) error {
	minigameIDStr := r.FormValue("minigameID")
	minigameID, _ := strconv.Atoi(minigameIDStr)

	fractions, err := database.GetFractionQuestions(minigameID)
	if err != nil {
		return err
	}

	for _, fraction := range fractions {
		fmt.Fprintf(w, `
			<div class="w-3/5 bg-neutral py-10 px-8 rounded-xl mt-4">
			<form action="/update/fractions" method="POST">
				<input type="hidden" name="question_id" value= "%d" />
				<input type="hidden" name="minigame_id" value= "%d" />
				<div class="flex gap-4 mt-4">
					<div class="label">
						<span class="label-text text-white">Fraction 1 Numerator:</span>
					</div>
					<input type="text" value="%d" name="fraction1_numerator" class="input input-bordered input-primary w-full max-w-xs" />
					<div class="label">
						<span class="label-text text-white">Fraction 1 Denominator:</span>
					</div>
					<input type="text" value="%d" name="fraction1_denominator" class="input input-bordered input-primary w-full max-w-xs" />
				</div>
				<div class="flex gap-4 mt-4">
				<div class="label">
					<span class="label-text text-white">Fraction 2 Numerator</span>
				</div>
					<input type="text" value="%d" name="fraction2_numerator" class="input input-bordered input-primary w-full max-w-xs" />
				<div class="label">
					<span class="label-text text-white">Fraction 2 Denominator</span>
				</div>
					<input type="text" value="%d" name="fraction2_denominator" class="input input-bordered input-primary w-full max-w-xs" />
				</div>

				<div class="flex justify-end">
					<button  type="submit" class="btn btn-primary text-white ">save changes</button>
				</div>
			</div>  	
			</form>
			</div>
		`, fraction.QuestionID, minigameID, fraction.Fraction1_Numerator, fraction.Fraction1_Denominator, fraction.Fraction2_Numerator, fraction.Fraction2_Denominator)
	}

	return nil
}

func HandleUpdateFractions(w http.ResponseWriter, r *http.Request) error {
	// get minigameID here
	minigameID := r.FormValue("minigame_id")
	if err := database.UpdateFractions(w, r); err != nil {
		return err
	}

	hxRedirect(w, r, "/minigame"+minigameID)
	return nil
}

func HandleMinigame2Index(w http.ResponseWriter, r *http.Request) error {
	return render(w, r, minigame.Fractions("2"))
}

func HandleMinigame3Index(w http.ResponseWriter, r *http.Request) error {
	return render(w, r, minigame.Worded("3"))
}

// return render(w, r, minigame.Worded("3"))
// }

func HandleGetWorded(w http.ResponseWriter, r *http.Request) error {
	minigameIDStr := r.FormValue("minigameID")
	minigameID, _ := strconv.Atoi(minigameIDStr)

	fractions, err := database.GetWordedQuestions(minigameID)
	if err != nil {
		return err
	}

	for _, fraction := range fractions {
		fmt.Fprintf(w, `
			<div class="w-3/5 bg-neutral py-10 px-8 rounded-xl mt-4">
			<form action="/update/fractions" method="POST">
				<input type="hidden" name="question_id" value= "%d" />
				<input type="hidden" name="minigame_id" value= "%d" />
				<div class="flex gap-4 mt-4 mb-4">
					<div class="label">
						<span class="label-text text-white">Question Text</span>
					</div>
					<input type="text" value="%s" name="question_text" class="input input-bordered input-primary w-full max-w-xs" />
				</div>	
				<div class="flex gap-4 mt-4">
					<div class="label">
						<span class="label-text text-white">Fraction 1 Numerator:</span>
					</div>
					<input type="text" value="%d" name="fraction1_numerator" class="input input-bordered input-primary w-full max-w-xs" />
					<div class="label">
						<span class="label-text text-white">Fraction 1 Denominator:</span>
					</div>
					<input type="text" value="%d" name="fraction1_denominator" class="input input-bordered input-primary w-full max-w-xs" />
				</div>
				<div class="flex gap-4 mt-4">
				<div class="label">
					<span class="label-text text-white">Fraction 2 Numerator</span>
				</div>
					<input type="text" value="%d" name="fraction2_numerator" class="input input-bordered input-primary w-full max-w-xs" />
				<div class="label">
					<span class="label-text text-white">Fraction 2 Denominator</span>
				</div>
					<input type="text" value="%d" name="fraction2_denominator" class="input input-bordered input-primary w-full max-w-xs" />
				</div>

				<div class="flex justify-end">
					<button  type="submit" class="btn btn-primary text-white ">save changes</button>
				</div>
			</div>  	
			</form>
			</div>
		`, fraction.QuestionID, minigameID, fraction.QuestionText, fraction.Fraction1_Numerator, fraction.Fraction1_Denominator, fraction.Fraction2_Numerator, fraction.Fraction2_Denominator)
	}
	return nil
}

// func HandleGetWorded1(w http.ResponseWriter, r *http.Request) error {

// }

// func HandleGetWorded2(w http.ResponseWriter, r *http.Request) error {

// }

// func HandleMinigame4Index(w http.ResponseWriter, r *http.Request) error {
// 	return render(w, r, minigame.Worded3())
// }

// func HandleGetWorded3(w http.ResponseWriter, r *http.Request) error {

// }

func HandleMinigame5Index(w http.ResponseWriter, r *http.Request) error {
	return render(w, r, minigame.Quiz("5"))
}

func HandleGetMCQuestions(w http.ResponseWriter, r *http.Request) error {
	minigameIDStr := r.FormValue("minigameID")
	minigameID, _ := strconv.Atoi(minigameIDStr)

	questions, err := database.GetQuestionDictionary(minigameID)
	if err != nil {
		return err
	}

	for i, question := range questions {
		fmt.Fprintf(w, `
			<div class="w-3/5 bg-neutral py-10 px-8 rounded-xl mt-4">
			<form action="/update/mcquestions" method="POST">
				<span class="label-text text-white">Question %d:</span>
				<input type="text" value="%s" name="question" class="input input-bordered input-primary w-full max-w-xs" />
				<div class="flex gap-4 mt-4">
					<div class="label">
						<span class="label-text text-white">Option 1:</span>
					</div>
					<input type="text" value="%s" name="option1" class="input input-bordered input-primary w-full max-w-xs" />
					<div class="label">
						<span class="label-text text-white">Option 2:</span>
					</div>
					<input type="text" value="%s" name="option2" class="input input-bordered input-primary w-full max-w-xs" />
				</div>
				<div class="flex gap-4 mt-4">
				<div class="label">
					<span class="label-text text-white">Option 3:</span>
				</div>
					<input type="text" value="%s" name="option3" class="input input-bordered input-primary w-full max-w-xs" />
				<div class="label">
					<span class="label-text text-white">Option 4:</span>
				</div>
					<input type="text" value="%s" name="option4" class="input input-bordered input-primary w-full max-w-xs" />
				</div>
				<div class="flex mt-4 relative inline-block w-64">
					<label for="dropdown" class="block text-white">Correct Answer</label>
						<select id="dropdown" name="correct_answer" class="mt-1 block w-full px-3 py-2 bg-white border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm">
							<option value="option1">Option 1</option>
							<option value="option2">Option 2</option>
							<option value="option3">Option 3</option>
						</select>
				</div>

				<div class="flex justify-end">
					<button  type="submit" class="btn btn-primary text-white ">save changes</button>
				</div>
			</div>  	
			</form>
			</div>
		`, i+1, question.QuestionText, question.Option1, question.Option2, question.Option3, question.Option4)
	}
	return err
}

func HandleUpdateMCQuestions(w http.ResponseWriter, r *http.Request) error {
	// we want minigameID
	if err := database.UpdateMCQuestions(w, r); err != nil {
		return err
	}

	hxRedirect(w, r, "/minigame5")
	return nil
}
