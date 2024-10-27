package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	// "os"
	"soln-teachermodule/database"
	"soln-teachermodule/view/statistics"

	"github.com/gorilla/sessions"
	// "github.com/gorilla/sessions"
)

func HandleGetStatistics(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	return render(w, r, statistics.Statistics())
}

func HandleGetClassStatistics(w http.ResponseWriter, r *http.Request) error {
	// Initialize the session store
	store := sessions.NewCookieStore([]byte(os.Getenv("SESSION_SECRET")))

	// Retrieve the session
	session, err := store.Get(r, sessionUserKey)
	if err != nil {
		http.Error(w, "Could not retrieve session", http.StatusInternalServerError)
		return err
	}

	// Retrieve classroomID from session safely
	classroomID, ok := session.Values["classroomID"].(int)
	if !ok {
		http.Error(w, "Invalid classroom ID in session", http.StatusBadRequest)
		return errors.New("classroomID not found or invalid in session")
	}
	fmt.Print("we got classroomID in handlegetclasstsatstics: ", classroomID)

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
