package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"soln-teachermodule/database"
	"soln-teachermodule/view/auth"
	"soln-teachermodule/view/home"
)

func HandleLoginIndex(w http.ResponseWriter, r *http.Request) error {

	return auth.Login().Render(r.Context(), w)
}

type LoginParams struct {
	Username			string
	Password			string
}

type LoginErrors struct {
	Username			string
	Password			string
	InvalidCredentials  string
}

func HandleLoginCreate(w http.ResponseWriter, r *http.Request) error {
	// authenticate the user
	if err := database.AuthenticateWebUser(w, r); err != nil {
		return err
	} else {
		return home.Index().Render(r.Context(), w)
	}
}

func HandleLoginGame(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return nil
	}

	// Read the JSON data from the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return nil
	}
	defer r.Body.Close()

	// Parse the JSON data into the Data struct
	type Data struct {
		Username string
		Password string
	}

	var data Data
	err = json.Unmarshal(body, &data)
	if err != nil {
		http.Error(w, "Failed to parse JSON", http.StatusBadRequest)
		return nil
	}

	type LoginResponse struct {
		Success bool `json:"success"`
	}

	log.Printf("Received data: %+v", data.Password)
	// next is to perform sql commands
	if database.AuthenticateSolnUser(data.Username, data.Password) {
		response := LoginResponse{Success: true}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	} else {
		response := LoginResponse{Success: false}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}

	return nil
}

func HandleRegisterIndex(w http.ResponseWriter, r *http.Request) error {
	return auth.Register().Render(r.Context(), w)
}

func HandleRegisterCreate(w http.ResponseWriter, r *http.Request) error {
	if err := database.RegisterAccount(w, r); err != nil {
		return err
	} else {
	fmt.Print("Account registered successfully!")
	return auth.Login().Render(r.Context(), w)
	}
}

type RegisterParams struct {
	Username			string
	Password			string
	ConfirmPassword		string
}
