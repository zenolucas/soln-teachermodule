package handler

import (
	"encoding/json"
	"net/http"

	// "os"
	"soln-teachermodule/database"
	"soln-teachermodule/view/statistics"
	// "github.com/gorilla/sessions"
)

func HandleGetStatistics(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	return render(w, r, statistics.Statistics())
}

func HandleGetClassStatistics(w http.ResponseWriter, r *http.Request) error {
	// get classroomID from session
	// store := sessions.NewCookieStore([]byte(os.Getenv("SESSION_SECRET")))
	// session, _ := store.Get(r, sessionUserKey)
	// classroomID := session.Values["classroomID"].(string)
	classroomID := 1
	statistics, err := database.GetClassStatistics(classroomID)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(statistics)
	return err
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
