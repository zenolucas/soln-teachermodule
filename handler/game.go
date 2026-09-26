package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"soln-teachermodule/database"

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
		Token       string `json:"token"`
		ErrorText   string `json:"error_text"`
	}

	var response LoginResponse

	// authenticate student
	if database.AuthenticateGameUser(r.Context(), data.Username, data.Password) {
		// get classroomID student is enrolled in
		classroomID, err := database.GetClassroomID(r.Context(), data.Username)
		if err != nil {
			return err
		}

		// get userID of student
		studentID, err := database.GetStudentID(r.Context(), data.Username)
		if err != nil {
			return err
		}

		// Issue a signed token the client must send back (as "Authorization: Bearer
		// <token>") on every other /game/* request that acts on this student's data -
		// those routes derive student_id from this token instead of trusting one
		// posted in the request body.
		token, err := issueGameToken(studentID)
		if err != nil {
			return err
		}

		response = LoginResponse{Success: true, ClassroomID: classroomID, StudentID: studentID, Token: token}
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
	fmt.Print("at get fractions, we got classroom id ", data.ClassroomID)

	fractions, err := database.GetFractionQuestions(r.Context(), data.MinigameID, data.ClassroomID)
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
		MinigameID  int `json:"minigameID"`
		ClassroomID int `json:"classroomID"`
	}

	var data Data
	err = json.Unmarshal(body, &data)
	if err != nil {
		http.Error(w, "Failed to parse JSON", http.StatusBadRequest)
		return nil
	}

	fmt.Print("we got minigameID ", data.MinigameID)

	questions, err := database.GetWordedQuestions(r.Context(), data.MinigameID, data.ClassroomID)
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
		fmt.Print(err.Error())
		return nil
	}
	defer r.Body.Close()

	type Data struct {
		MinigameID  int `json:"MinigameID"`
		ClassroomID int `json:"ClassroomID"`
	}

	var data Data
	err = json.Unmarshal(body, &data)
	if err != nil {
		http.Error(w, "Failed to parse JSON", http.StatusBadRequest)
		fmt.Print(err.Error())
		return nil
	}

	questions, err := database.GetQuizQuestions(r.Context(), data.MinigameID, data.ClassroomID)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(questions)
	return nil
}

func HandleGetSaveData(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return nil
	}

	// student_id comes from the token issued at /game/login, not the request body -
	// a body-supplied student_id would let any client read any student's save data.
	studentID, err := authenticateGameRequest(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return err
	}

	doc, err := database.GetSaveData(r.Context(), studentID)
	if err != nil {
		return err
	}

	// The fixed-column response this replaced carried student_id; keep the payload shape.
	var save map[string]json.RawMessage
	if err := json.Unmarshal(doc, &save); err != nil {
		return fmt.Errorf("stored save for student %d isn't a JSON object: %w", studentID, err)
	}
	save["student_id"], _ = json.Marshal(studentID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(save)
	return nil
}

// maxSaveBytes caps a posted save. Today's is ~1.3 KB; this leaves room for the game to grow.
const maxSaveBytes = 64 << 10

func HandleUpdateSaveData(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return nil
	}

	// student_id comes from the token issued at /game/login, not the request body -
	// a body-supplied student_id would let any client overwrite any student's save
	// data. Authenticate before trusting anything else in the body.
	studentID, err := authenticateGameRequest(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return err
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxSaveBytes))
	if err != nil {
		http.Error(w, "save data is too large or unreadable", http.StatusBadRequest)
		return nil
	}
	defer r.Body.Close()

	patch, err := cleanSavePatch(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil
	}

	if err := database.MergeSaveData(r.Context(), studentID, patch); err != nil {
		return err
	}

	type Response struct {
		Success bool `json:"success"`
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{Success: true})
	return nil
}

// cleanSavePatch validates a posted save and returns the JSON to merge into the stored one. It must
// be a JSON object. student_id is dropped because the token decides whose save it is, and top-level
// nulls are dropped because a null in a merge patch deletes the key - DefaultSave's value included.
func cleanSavePatch(body []byte) ([]byte, error) {
	var patch map[string]json.RawMessage
	if err := json.Unmarshal(body, &patch); err != nil || patch == nil {
		return nil, errors.New("save data must be a JSON object")
	}
	delete(patch, "student_id")
	for key, value := range patch {
		if string(bytes.TrimSpace(value)) == "null" {
			delete(patch, key)
		}
	}
	return json.Marshal(patch)
}
