package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"soln-teachermodule/database"
	"soln-teachermodule/types"

	"github.com/go-sql-driver/mysql"
)

func HandleGameLogin(w http.ResponseWriter, r *http.Request) error {
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
		Username string
		Password string
	}

	var data Data
	err = json.Unmarshal(body, &data)
	if err != nil {
		http.Error(w, "Failed to parse JSON", http.StatusBadRequest)
		return err
	}

	type LoginResponse struct {
		Success     bool   `json:"success"`
		ClassroomID int    `json:"classroom_id"`
		StudentID   int    `json:"student_id"`
		ErrorText   string `json:"error_text"`
	}

	var response LoginResponse

	log.Printf("Received data: %+v", data.Password)
	// authenticate student
	if database.AuthenticateGameUser(data.Username, data.Password) {
		// get classroomID student is enrolled in
		classroomID, err := database.GetClassroomID(data.Username)
		if err != nil {
			return err
		}

		// get userID of student
		studentID, err := database.GetStudentID(data.Username)
		if err != nil {
			return err
		}
		response = LoginResponse{Success: true, ClassroomID: classroomID, StudentID: studentID}
	} else {
		response = LoginResponse{Success: false, ErrorText: "wrong username or password"}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
	return nil
}

func HandleGameRegister(w http.ResponseWriter, r *http.Request) error {

	type RegisterResponse struct {
		Success   bool   `json:"success"`
		ErrorText string `json:"error_text"`
	}

	var response RegisterResponse

	if err := database.RegisterGameAccount(w, r); err != nil {
		// check what kind of error
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			response = RegisterResponse{Success: false, ErrorText: "username is already taken"}
		} else {
			fmt.Print(err)
			response = RegisterResponse{Success: false, ErrorText: "register error"}
		}
	} else {
		response = RegisterResponse{Success: true}
	}


	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
	return nil
}

func HandleGetGameFractions(w http.ResponseWriter, r *http.Request) error {
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
		MinigameID  int `json:"minigameID"`
		ClassroomID int `json:"classroomID"`
	}

	var data Data
	err = json.Unmarshal(body, &data)
	if err != nil {
		http.Error(w, "Failed to parse JSON", http.StatusBadRequest)
		return nil
	}

	fmt.Print("at get fractions, we got minigame id ", data.MinigameID)

	fractions, err := database.GetFractionQuestions(data.MinigameID, data.ClassroomID)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(fractions)
	return nil
}

func HandleGetGameWorded(w http.ResponseWriter, r *http.Request) error {
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
		MinigameID  int `json:"minigame_id"`
		ClassroomID int `json:"classroom_id"`
	}

	var data Data
	err = json.Unmarshal(body, &data)
	if err != nil {
		http.Error(w, "Failed to parse JSON", http.StatusBadRequest)
		return nil
	}

	fmt.Print("we got minigameID ", data.MinigameID)

	questions, err := database.GetWordedQuestions(data.MinigameID, data.ClassroomID)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(questions)
	return nil
}

func HandleGetGameMCQuestions(w http.ResponseWriter, r *http.Request) error {
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
		MinigameID  int `json:"minigame_id"`
		ClassroomID int `json:"classroom_id"`
	}

	var data Data
	err = json.Unmarshal(body, &data)
	if err != nil {
		http.Error(w, "Failed to parse JSON", http.StatusBadRequest)
		return nil
	}

	questions, err := database.GetQuizQuestions(data.MinigameID, data.ClassroomID)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(questions)
	return nil
}

func HandleGetSaveData(w http.ResponseWriter, r *http.Request) error {
	fmt.Print("getsavedata is triggered!")
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
		StudentID int `json:"student_id"`
	}

	var data Data
	err = json.Unmarshal(body, &data)
	if err != nil {
		http.Error(w, "Failed to parse JSON", http.StatusBadRequest)
		return err
	}

	var response types.SaveData

	response, error := database.GetSavedData(data.StudentID)
	if error != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
	return nil
}

func HandlePostSaveData(w http.ResponseWriter, r *http.Request) error {

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

	type Response struct {
		Success bool `json:"success"`
	}

	var data types.SaveData
	err = json.Unmarshal(body, &data)
	if err != nil {
		http.Error(w, "Failed to parse JSON", http.StatusBadRequest)
		return err
	}

	fmt.Print(data)

	err = database.SaveData(data)
	if err != nil {
		return err
	}

	response := Response{Success: true}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
	return nil
}
