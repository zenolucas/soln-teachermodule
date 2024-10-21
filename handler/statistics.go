package handler

import (
	"encoding/json"
	"net/http"
	"soln-teachermodule/database"
	"soln-teachermodule/view/statistics"
)

func HandleGetStatistics(w http.ResponseWriter, r *http.Request) error {
	return render(w, r, statistics.Statistics())
}

func HandleUpdateStatistics(w http.ResponseWriter, r *http.Request) error {

	type StatisticsResponse struct {
		Success bool `json:"success"`
	}

	err := database.UpdateStatisticsDatabase(w, r)
	if err != nil {
		response := StatisticsResponse{Success: false}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return err
	}

	response := StatisticsResponse{Success: false}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
	return nil
}
