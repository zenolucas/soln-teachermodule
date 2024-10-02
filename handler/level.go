package handler

import (
	"fmt"
	"net/http"
	"soln-teachermodule/database"
	view "soln-teachermodule/view/level"
)

func HandleLevelIndex(w http.ResponseWriter, r *http.Request) error {
	return render(w, r, view.Level())
}

func HandleGetMCQuestions(w http.ResponseWriter, r *http.Request) error {
	questions, err := database.GetQuestionDictionary(1)
	if err != nil {
		return err
	}

	for i, question := range questions {
		fmt.Fprintf(w, `
			<div class="w-3/5 bg-neutral py-10 px-8 rounded-xl mt-4">
			<form action="/updatemcquestions" method="POST">
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
		`, i+1, question.QuestionText, question.Option1, question.Option2, question.Option3, question.Option4)
	}
	return err
}

func HandleUpdateMCQuestions(w http.ResponseWriter, r *http.Request) error {
	if err := database.UpdateMCQuestions(w, r); err != nil {
		return err
	}

	return render(w, r, view.Level())
}
