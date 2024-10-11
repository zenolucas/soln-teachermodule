package handler

import (
	"encoding/json"
	"net/http"
	"soln-teachermodule/database"
)


func HandleGameRegister(w http.ResponseWriter, r *http.Request) error {

	type RegisterResponse struct {
		Success bool `json:"success"`
	}

	err := database.RegisterGameAccount(w, r)
	if err != nil {
		response := RegisterResponse{Success: false}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return err
	}

	response := RegisterResponse{Success: false}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
	return nil 
}