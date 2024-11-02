package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"soln-teachermodule/database"
	"soln-teachermodule/types"

	// "soln-teachermodule/types"
	"soln-teachermodule/view/statistics"

	"github.com/gorilla/sessions"
)

func HandleStatisticsIndex(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// get classroomID from session
	store := sessions.NewCookieStore([]byte(os.Getenv("SESSION_SECRET")))
	session, _ := store.Get(r, sessionUserKey)
	classroomID := session.Values["classroomID"].(int)
	classroomIDStr := strconv.Itoa(classroomID)

	// get minigameID
	minigameID := r.URL.Query().Get("minigameID")

	fmt.Print("loading up statistics, we got minigameID: ", minigameID)

	if minigameID == "1" {
		return render(w, r, statistics.FractionStatistics(classroomIDStr, minigameID))
	} else if minigameID == "2" {
		return render(w, r, statistics.FractionStatistics(classroomIDStr, minigameID))
	}

	if minigameID == "5" {
		return render(w, r, statistics.QuizStatistics(classroomIDStr, minigameID))
	} else {
		http.Error(w, "invalid minigame id", http.StatusBadRequest)
		return errors.New("bad request")
	}
}

func HandleFractionQuestionCharts(w http.ResponseWriter, r *http.Request) error {
	// get classroomID from session values
	store := sessions.NewCookieStore([]byte(os.Getenv("SESSION_SECRET")))
	session, _ := store.Get(r, sessionUserKey)
	classroomID := session.Values["classroomID"].(int)

	// get minigameID
	minigameIDStr := r.URL.Query().Get("minigameID")
	minigameID, _ := strconv.Atoi(minigameIDStr)

	// questionIDs to put into the url parameters on async functions
	var questions []types.FractionQuestion
	questions, err := database.GetFractionQuestions(minigameID)
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
				const response = await fetch('http://localhost:3000/statistics/fraction/question/data?questionID=%d&classroomID=%d&minigameID=%d');
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

	statistics, err := database.GetFractionResponseStatistics(classroomID, minigameID, questionID)
	if err != nil {
		http.Error(w, "Error retrieving class statistics", http.StatusInternalServerError)
		return err
	}

	fmt.Print("on fractionrepsonsestats: we got : ", statistics)

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

	// Fetch the statistics from the database
	statistics, err := database.GetQuizClassStatistics(classroomID, minigameID)
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
	return render(w, r, statistics.QuestionStatistics(minigameIDStr))
}

func HandleQuizQuestionCharts(w http.ResponseWriter, r *http.Request) error {
	// get classroomID from session values
	store := sessions.NewCookieStore([]byte(os.Getenv("SESSION_SECRET")))
	session, _ := store.Get(r, sessionUserKey)
	classroomID := session.Values["classroomID"].(int)

	// get minigameID
	minigameIDStr := r.URL.Query().Get("minigameID")
	minigameID, _ := strconv.Atoi(minigameIDStr)

	// questionIDs to put into the url parameters on async functions
	var questions []types.MultipleChoiceQuestion
	questions, err := database.GetQuizQuestions(minigameID)
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
				const response = await fetch('http://localhost:3000/statistics/quiz/question/data?questionID=%d&classroomID=%d&minigameID=%d');
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
		`, i+1, question.QuestionText, i, i, question.QuestionID, classroomID, minigameID, i, i, i, i, i, i, i, setColors(question))
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

	statistics, err := database.GetQuizResponseStatistics(classroomID, minigameID, questionID)
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

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return nil
	}
	defer r.Body.Close()

	type Data struct {
		ClassroomID int
		MinigameID  int
		StudentID   int
		Score       int
	}

	var data Data
	err = json.Unmarshal(body, &data)
	if err != nil {
		http.Error(w, "Failed to parse JSON", http.StatusBadRequest)
		return err
	}

	fmt.Print("we recieved data: ", data)

	type LoginResponse struct {
		Success bool `json:"success"`
	}

	// record quiz statistics
	err = database.AddQuizStatistics(data.ClassroomID, data.MinigameID, data.StudentID, data.Score)
	if err != nil {
		return err
	}

	return nil
}

func HandleQuizResponse(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return nil
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return nil
	}
	defer r.Body.Close()

	type Data struct {
		ClassroomID int
		MinigameID  int
		QuestionID  int
		StudentID   int
		ChoiceID    int
	}

	var data Data
	err = json.Unmarshal(body, &data)
	if err != nil {
		http.Error(w, "Failed to parse JSON", http.StatusBadRequest)
		return err
	}

	fmt.Print("we recieved data: ", data)

	type LoginResponse struct {
		Success bool `json:"success"`
	}

	// record quiz statistics
	err = database.AddQuizResponse(data.ClassroomID, data.MinigameID, data.QuestionID, data.StudentID, data.ChoiceID)
	if err != nil {
		return err
	}

	return nil
}

func HandleUpdateSaisaiStatistics(w http.ResponseWriter, r *http.Request) error {
	type StatisticsResponse struct {
		Success bool `json:"success"`
	}

	err := database.AddSaisaiStatistics(w, r)
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
