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

	if minigameID == "5" {
		return render(w, r, statistics.QuizStatistics(classroomIDStr, minigameID))
	} else {
		http.Error(w, "invalid minigame id", http.StatusBadRequest)
		return errors.New("bad request")
	}
}

func HandleGetClassStatistics(w http.ResponseWriter, r *http.Request) error {
	classroomIDStr := r.URL.Query().Get("classroomID")
	minigameIDStr := r.URL.Query().Get("minigameID")

	fmt.Printf("we got classroomid: %s, and minigameID %s", classroomIDStr, minigameIDStr)

	// convert string to int
	classroomID, _ := strconv.Atoi(classroomIDStr)
	minigameID, _ := strconv.Atoi(minigameIDStr)

	// Fetch the statistics from the database
	statistics, err := database.GetClassStatistics(classroomID, minigameID)
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

func HandleQuestionStatisticsIndex(w http.ResponseWriter, r *http.Request) error {
	// we get minigameID 1 or 10
	minigameIDStr := r.URL.Query().Get("minigameID")
	return render(w, r, statistics.QuestionStatistics(minigameIDStr))
}

func HandleGetQuestionCharts(w http.ResponseWriter, r *http.Request) error {
	// get classroomID from session values
	store := sessions.NewCookieStore([]byte(os.Getenv("SESSION_SECRET")))
	session, _ := store.Get(r, sessionUserKey)
	classroomID := session.Values["classroomID"].(int)

	// get minigameID
	minigameIDStr := r.URL.Query().Get("minigameID")
	minigameID, _ := strconv.Atoi(minigameIDStr)

	// questionIDs to put into the url parameters on async functions
	var questionIDs []int
	questionIDs, err := database.GetQuestionIDs(minigameID)
	if err != nil {
		return err
	}

	// questions := []types.MultipleChoiceQuestion{
	// 	{
	// 		QuestionID:    1,
	// 		QuestionText:  "What is the capital of France?",
	// 		Option1:       "Berlin",
	// 		Option2:       "Madrid",
	// 		Option3:       "Paris",
	// 		Option4:       "Rome",
	// 		CorrectAnswer: "Paris",
	// 	},
	// 	{
	// 		QuestionID:    2,
	// 		QuestionText:  "Which planet is known as the Red Planet?",
	// 		Option1:       "Earth",
	// 		Option2:       "Mars",
	// 		Option3:       "Jupiter",
	// 		Option4:       "Venus",
	// 		CorrectAnswer: "Mars",
	// 	},
	// 	{
	// 		QuestionID:    3,
	// 		QuestionText:  "What is the largest ocean on Earth?",
	// 		Option1:       "Indian Ocean",
	// 		Option2:       "Atlantic Ocean",
	// 		Option3:       "Arctic Ocean",
	// 		Option4:       "Pacific Ocean",
	// 		CorrectAnswer: "Pacific Ocean",
	// 	},
	// }

	for i, id := range questionIDs {
		fmt.Fprintf(w, `
			<div class="w-3/5 bg-base-100 py-10 px-8 rounded-xl mt-4 mb-4">
				<canvas id="QuestionsChart%d" width="300" height="200"></canvas>
			</div>
			<script>
				async function getClassStatistics() {
				const response = await fetch('http://localhost:3000/statistics/question/data?questionID=%d&classroomID=%d');
				const results = await response.json();
				return results
				}

				getClassStatistics().then(results => {
					results;
					const label = results.map(item => item.choice);  
					const count = results.map(item => item.count);  
					renderChart(label, count);
				});

				function renderChart(label, count) {
					Chart.defaults.font.size = 30;  // Set the default font size globally
					var ctx = document.getElementById('QuestionChart%d').getContext('2d');
					var myChart = new Chart(ctx, {
						type: 'bar',  // Keep type as 'bar'
						data: {
							labels: label, 
							datasets: [{
								data: count, 
								borderWidth: 1
							}]
						},
						options: {
							indexAxis: 'x',  // This makes the bars horizontal
							scales: {
								x: {
									beginAtZero: true  // X-axis starts at 0
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
		`, i, id, classroomID, i)
	}
	return nil
}

// func HandleGetQuestionStatistics(w http.ResponseWriter, r *http.Request) error {
// 	classroomIDStr := r.URL.Query().Get("questionID")
// 	questionIDStr := r.URL.Query().Get("questionID")

// 	classroomID, _ := strconv.Atoi(classroomIDStr)
// 	questionID, _ := strconv.Atoi(questionIDStr)

// 	statistics, err := database.GetQuestionStatistics(classroomID, minigameID, questionID)
// 	if err != nil {
// 		http.Error(w, "Error retrieving class statistics", http.StatusInternalServerError)
// 		return err
// 	}

// 	// Set headers and send the response
// 	w.Header().Set("Content-Type", "application/json")
// 	w.Header().Set("Access-Control-Allow-Origin", "*")
// 	json.NewEncoder(w).Encode(statistics)
// 	return nil

// }

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

	fmt.Print("POST RESPONSE IS EXECUTED!!!")

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
