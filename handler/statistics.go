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

	for i, question := range questions {
		fmt.Fprintf(w, `
			<div class="w-3/5 bg-base-100 py-10 px-8 rounded-xl mt-4 mb-4">
				<div class="text-2xl mt-2 mb-2">Question: %d/%d + %d/%d ?</div>
				<canvas id="QuestionChart%d" width="300" height="200"></canvas>
			</div>
			<script>
				async function getClassStatistics%d() {
				const response = await fetch('/statistics/fraction/question/data?questionID=%d&classroomID=%d&minigameID=%d');
				const results = await response.json();
				return results
				}

				getClassStatistics%d().then(results => {
					results;
					const right = results.map(item => item.num_right_attempts);  
					const wrong = results.map(item => item.num_wrong_attempts);  
					var count = right.concat(wrong)
					renderChart%d(count);
				});

				function renderChart%d(count) {
					Chart.defaults.font.size = 30;  // Set the default font size globally
					var ctx%d = document.getElementById('QuestionChart%d').getContext('2d');
					var myChart%d = new Chart(ctx%d, {
						type: 'bar',  // Keep type as 'bar'
						data: {
							labels: ["Correct Attempts", "Wrong Attempts"], 
							datasets: [{
								data: count, 
								borderWidth: 1,
								categoryPercentage: 0.3,
								backgroundColor: [
									'rgba(75, 192, 192, 0.5)',
									'rgba(255, 99, 132, 0.5)'
									]
							}]
						},
						options: {
							indexAxis: 'x',  // This makes the bars horizontal
							scales: {
								y: {
									beginAtZero: true,  
									ticks: {
										stepSize: 1
									}
								}
							},
							plugins: {
								legend: {
									display: false
								}
							}
						}
					});
				}
			</script>
		`, question.Fraction1_Numerator, question.Fraction1_Denominator, question.Fraction2_Numerator, question.Fraction2_Denominator, i, i, question.QuestionID, classroomID, minigameID, i, i, i, i, i, i, i)
	}

	return nil
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

	for i, question := range questions {
		fmt.Fprintf(w, `
			<div class="w-3/5 bg-base-100 py-10 px-8 rounded-xl mt-4 mb-4">
				<div class="text-2xl mt-2 mb-2">Question: %s</div>
				<canvas id="QuestionChart%d" width="300" height="200"></canvas>
			</div>
			<script>
				async function getClassStatistics%d() {
				const response = await fetch('/statistics/worded/question/data?questionID=%d&classroomID=%d&minigameID=%d');
				const results = await response.json();
				return results
				}

				getClassStatistics%d().then(results => {
					results;
					const right = results.map(item => item.num_right_attempts);  
					const wrong = results.map(item => item.num_wrong_attempts);  
					var count = right.concat(wrong)
					renderChart%d(count);
				});

				function renderChart%d(count) {
					Chart.defaults.font.size = 30;  // Set the default font size globally
					var ctx%d = document.getElementById('QuestionChart%d').getContext('2d');
					var myChart%d = new Chart(ctx%d, {
						type: 'bar',  // Keep type as 'bar'
						data: {
							labels: ["Correct Attempts", "Wrong Attempts"], 
							datasets: [{
								data: count, 
								borderWidth: 1,
								categoryPercentage: 0.3,
								backgroundColor: [
									'rgba(75, 192, 192, 0.5)',
									'rgba(255, 99, 132, 0.5)'
									]
							}]
						},
						options: {
							indexAxis: 'x',  // This makes the bars horizontal
							scales: {
								y: {
									beginAtZero: true,  
									ticks: {
										stepSize: 1
									}
								}
							},
							plugins: {
								legend: {
									display: false
								}
							}
						}
					});
				}
			</script>
		`, esc(question.QuestionText), i, i, question.QuestionID, classroomID, minigameID, i, i, i, i, i, i, i)
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

	for i, question := range questions {
		fmt.Fprintf(w, `
			<div class="w-3/5 bg-base-100 py-10 px-8 rounded-xl mt-4 mb-4">
				<div class="text-2xl mt-2 mb-2">Question %d: %s</div>
				<canvas id="QuestionChart%d" width="300" height="200"></canvas>
			</div>
			<script>
				async function getClassStatistics%d() {
				const response = await fetch('/statistics/quiz/question/data?questionID=%d&classroomID=%d&minigameID=%d');
				const results = await response.json();
				return results
				}

				getClassStatistics%d().then(results => {
					results;
					const label = results.map(item => item.choice);  
					const count = results.map(item => item.count);  
					console.log(label)
					console.log(count)
					renderChart%d(label, count);
				});

				function renderChart%d(label, count) {
					Chart.defaults.font.size = 30;  // Set the default font size globally
					var ctx%d = document.getElementById('QuestionChart%d').getContext('2d');
					var myChart%d = new Chart(ctx%d, {
						type: 'bar',  // Keep type as 'bar'
						data: {
							labels: label, 
							datasets: [{
								label: 'number of responses',
								data: count, 
								%s
								borderWidth: 1
							}]
						},
						options: {
							indexAxis: 'y',  // This makes the bars horizontal
							scales: {
								x: {
									beginAtZero: true,  // X-axis starts at 0
									ticks: {
										stepSize: 1
									}
								}
							},
							plugins: {
								legend: {
									display: false
								}
							}
						}
					});
				}
			</script>
		`, i+1, esc(question.QuestionText), i, i, question.QuestionID, classroomID, minigameID, i, i, i, i, i, i, i, setColors(question))
	}
	return nil
}

// helper function to set bar colors for question statistics
func setColors(question types.MultipleChoiceQuestion) string {
	choices := question.Choices

	for i, choice := range choices {
		if choice.IsCorrect {
			if i == 0 {
				return `
					backgroundColor: [
						'rgba(75, 192, 192, 0.5)',
						'rgba(255, 99, 132, 0.5)',
						'rgba(255, 99, 132, 0.5)',
						'rgba(255, 99, 132, 0.5)'
						],`
			} else if i == 1 {
				return `
					backgroundColor: [
						'rgba(255, 99, 132, 0.5)',
						'rgba(75, 192, 192, 0.5)',
						'rgba(255, 99, 132, 0.5)',
						'rgba(255, 99, 132, 0.5)'
						],`
			} else if i == 2 {
				return `
					backgroundColor: [
						'rgba(255, 99, 132, 0.5)',
						'rgba(255, 99, 132, 0.5)',
						'rgba(75, 192, 192, 0.5)',
						'rgba(255, 99, 132, 0.5)'
						],`
			} else {
				return `
					backgroundColor: [
						'rgba(255, 99, 132, 0.5)',
						'rgba(255, 99, 132, 0.5)',
						'rgba(255, 99, 132, 0.5)',
						'rgba(75, 192, 192, 0.5)'
						],`
			}
		}
	}
	return ""
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
