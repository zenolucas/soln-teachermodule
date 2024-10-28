package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"soln-teachermodule/database"
	"soln-teachermodule/view/statistics"

	"github.com/gorilla/sessions"
)

func HandleStatisticsIndex(w http.ResponseWriter, r *http.Request) error {
	store := sessions.NewCookieStore([]byte(os.Getenv("SESSION_SECRET")))
	session, _ := store.Get(r, sessionUserKey)
	classroomID := session.Values["classroomID"].(int)
	// fmt.Print("in statistics classroomID is ", session.Values["classroomID"])
	fmt.Print("in statistics we got classroomID: ", classroomID)
	classroomIDStr := strconv.Itoa(classroomID)

	w.Header().Set("Access-Control-Allow-Origin", "*")
	return render(w, r, statistics.Statistics(classroomIDStr))
}

func HandleGetClassStatistics(w http.ResponseWriter, r *http.Request) error {
	classroomIDStr := r.URL.Query().Get("classroomID")
	fmt.Print("BUT THEN classroomID is : ", classroomIDStr)

	// convert string to int
	classroomID, _ := strconv.Atoi(classroomIDStr)

	// Retrieve classroomID from session safely
	// Fetch the statistics from the database
	statistics, err := database.GetClassStatistics(classroomID)
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
	return render(w, r, statistics.QuestionStatistics())
}





// game functions below

func HandleUpdateStatistics(w http.ResponseWriter, r *http.Request) error {
	type StatisticsResponse struct {
		Success bool `json:"success"`
	}

	err := database.UpdateStatistics(w, r)
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

func HandleUpdateSaisaiStatistics(w http.ResponseWriter, r *http.Request) error {
	type StatisticsResponse struct {
		Success bool `json:"success"`
	}

	err := database.UpdateSaisaiStatistics(w, r)
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
