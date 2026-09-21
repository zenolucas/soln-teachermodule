package database

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"soln-teachermodule/types"
	"strconv"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

var db *sql.DB

const (
	sessionAccessTokenKey = "access_token"
)

// formInt reads a form value and parses it as an int, returning a descriptive error
// instead of silently defaulting to 0 on missing or non-numeric input - a bare
// strconv.Atoi with the error discarded is what let a bad or missing ID quietly turn
// into a query against ID 0 (see BUG-03/BUG-16). This mirrors handler.formInt; it lives
// here too because several Update* functions in this package read r.FormValue directly
// rather than receiving already-parsed values from the handler layer.
func formInt(r *http.Request, key string) (int, error) {
	n, err := strconv.Atoi(r.FormValue(key))
	if err != nil {
		return 0, fmt.Errorf("invalid or missing %q: %w", key, err)
	}
	return n, nil
}

func InitializeDatabase() error {
	if err := godotenv.Load(); err != nil {
		log.Fatal(err)
		return err
	}

	// Capture connection properties.
	cfg := mysql.Config{
		User:                 os.Getenv("DBUSER"),
		Passwd:               os.Getenv("DBPASS"),
		Net:                  "tcp",
		Addr:                 "127.0.0.1:3306",
		DBName:               os.Getenv("DBNAME"),
		AllowNativePasswords: true,
	}

	// Get a database handle.
	var err error
	db, err = sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		log.Fatal(err)
	}

	pingErr := db.Ping()
	if pingErr != nil {
		log.Fatal(pingErr)
	}
	fmt.Println("Database connection established.")

	return err
}

func AuthenticateWebUser(username string, password string) error {
	var storedPassword string
	row := db.QueryRow("SELECT password FROM users WHERE username = ? AND usertype = ?", username, "teacher")
	if err := row.Scan(&storedPassword); err != nil {
		if err == sql.ErrNoRows {
			fmt.Print("authentication Error: incorrect username or password")
			return fmt.Errorf("authentication Error: incorrect username or password")
		} else {
			fmt.Print("database Error: ", err)
			return fmt.Errorf("database Error: %v", err)
		}
	}

	if password != storedPassword {
		fmt.Print("authentication Error: incorrect username or password")
		return fmt.Errorf("authentication Error: incorrect username or password")
	}
	fmt.Println("Login Success! Hello ", username, "!")
	// Authentication successful
	return nil
}

func AuthenticateGameUser(username string, password string) bool {

	var storedPassword string
	row := db.QueryRow("SELECT password FROM users WHERE username = ? AND usertype = ?", username, "student")
	if err := row.Scan(&storedPassword); err != nil {
		if err == sql.ErrNoRows {
			fmt.Print("authentication Error: incorrect username or password")
			return false
		} else {
			fmt.Print("database Error: ", err)
			return false
		}
	}

	if password != storedPassword {
		fmt.Print("authentication Error: incorrect username or password")
		return false
	}
	fmt.Println("Login Success! Hello ", username, "!")
	// Authentication successful
	return true
}

// gets classroomID of a student
func GetClassroomID(username string) (int, error) {
	var classroomID int
	var section string

	// first get section of student
	err := db.QueryRow("SELECT section FROM users WHERE username = ? AND usertype = ?", username, "student").Scan(&section)
	if err != nil {
		return 0, err
	}
	// then get classroomID given section
	err = db.QueryRow("SELECT classroom_id FROM classrooms WHERE section = ?", section).Scan(&classroomID)
	if err != nil {
		return 0, err
	}

	fmt.Print("returned classroomID is :", classroomID)
	return classroomID, nil
}

func GetStudentID(username string) (int, error) {
	var studentID int

	// first get section of student
	err := db.QueryRow("SELECT user_id FROM users WHERE username = ? AND usertype = ?", username, "student").Scan(&studentID)
	if err != nil {
		return 0, err
	}

	return studentID, nil
}

func GetStudent(userID int) (types.Student, error) {
	var student types.Student

	err := db.QueryRow("SELECT firstname, lastname FROM users WHERE user_id = ?", userID).Scan(&student.Firstname, &student.Lastname)
	if err != nil {
		return student, err
	}
	return student, nil
}

func RegisterAccount(w http.ResponseWriter, r *http.Request) error {
	userCreds := types.UserCredentials{
		Username: r.FormValue("username"),
		Password: r.FormValue("password"),
	}

	_, err := db.Exec("INSERT INTO users (username, password, usertype) VALUES (?, ?, ?)", userCreds.Username, userCreds.Password, "teacher")
	if err != nil {
		return err
	}
	return nil
}

func RegisterGameAccount(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return errors.New("Invalid Request Method")
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return err
	}
	defer r.Body.Close()

	type Data struct {
		FirstName   string
		Lastname    string
		Username    string
		Password    string
		Section     string
		ClassNumber string
	}

	var data Data

	err = json.Unmarshal(body, &data)
	if err != nil {
		http.Error(w, "Failed to parse JSON", http.StatusBadRequest)
		return err
	}
	fmt.Print("we got data: ", data)

	result, err := db.Exec("INSERT INTO users (username, usertype, firstname, lastname, section, class_number, password) VALUES (?, ?, ?, ?, ?, ?, ?)", data.Username, "student", data.FirstName, data.Lastname, data.Section, data.ClassNumber, data.Password)
	if err != nil {
		return err
	}

	studentID, err := result.LastInsertId()
	if err != nil {
		return err
	}

	// after creating account, create save state
	err = CreateSaveState(studentID)
	if err != nil {
		return err
	}

	return nil
}

func CreateSaveState(studentID int64) error {
	_, err := db.Exec("INSERT INTO save_states (student_id) VALUES (?)", studentID)
	if err != nil {
		return err
	}

	return nil
}

func GetStudents(classroomID int) ([]types.Student, error) {
	var students []types.Student
	// get students given classroomID
	rows, err := db.Query("SELECT users.firstname, users.lastname, users.user_id FROM enrollments e JOIN users ON e.student_id = users.user_id WHERE e.classroom_id = ? ", classroomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var student types.Student
		if err := rows.Scan(&student.Firstname, &student.Lastname, &student.UserID); err != nil {
			return nil, fmt.Errorf("GetStudents: %v", err)
		}
		students = append(students, student)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetUsers: %v", err)
	}

	return students, nil
}

func GetUnenrolledStudents(classroomID int) ([]types.Student, error) {
	var students []types.Student

	// get students given classroomID
	rows, err := db.Query("SELECT user_id, firstname, lastname FROM users WHERE usertype = ? AND user_id NOT IN (SELECT student_id FROM enrollments WHERE classroom_id = ?)", "student", classroomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var student types.Student
		if err := rows.Scan(&student.UserID, &student.Firstname, &student.Lastname); err != nil {
			return nil, fmt.Errorf("GetUnenrolledStudents: %v", err)
		}
		students = append(students, student)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetUnenrolledStudents: %v", err)
	}

	return students, nil
}

func AddStudents(studentIDs []string, classroomID int) error {
	for _, studentID := range studentIDs {

		fmt.Println("adding student", studentID)

		_, err := db.Exec("INSERT INTO enrollments (classroom_id, student_id) VALUES (?, ?)", classroomID, studentID)
		if err != nil {
			return err
		}
		fmt.Println("add success!")
	}

	return nil
}

func UnenrollStudent(studentID int, classroomID int) error {
	// Execute the DELETE query
	_, err := db.Exec("DELETE FROM enrollments WHERE student_id = ? AND classroom_id = ?", studentID, classroomID)
	if err != nil {
		return err
	}
	return nil
}

func InsertClassroom(classroom types.Classroom, teacherID int) error {
	_, err := db.Exec("INSERT INTO classrooms (classroom_name, section, description, teacher_ID) VALUES (?, ?, ?, ?)", classroom.ClassroomName, classroom.Section, classroom.Description, teacherID)
	if err != nil {
		return err
	}
	return nil
}

func GetTeacherID(w http.ResponseWriter, r *http.Request) (int, error) {
	userCreds := types.UserCredentials{
		Username: r.FormValue("username"),
	}

	var teacherID int
	err := db.QueryRow("SELECT user_id FROM users WHERE username = ? AND usertype = ?", userCreds.Username, "teacher").Scan(&teacherID)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusNotFound)
			return 0, err
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return 0, err
	}

	return teacherID, nil
}

func GetSection(classroomID int) (string, error) {
	var section string
	err := db.QueryRow("SELECT section FROM classrooms WHERE classroom_id = ?", classroomID).Scan(&section)
	if err != nil {
		return "", err
	}
	return section, nil
}

// Example function to save session token in the database
func SaveSessionToken(userID int, sessionToken string) error {
	query := `INSERT INTO sessions (user_id, session_token, expires_at) VALUES (?, ?, ?)`
	_, err := db.Exec(query, userID, sessionToken, time.Now().Add(24*time.Hour))
	return err
}

func GetClassrooms(teacherID int) ([]types.Classroom, error) {
	var classrooms []types.Classroom

	rows, err := db.Query("SELECT classroom_id, classroom_name, section, description FROM classrooms WHERE teacher_id = ?", teacherID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var classroom types.Classroom
		if err := rows.Scan(&classroom.ClassroomID, &classroom.ClassroomName, &classroom.Section, &classroom.Description); err != nil {
			return nil, err
		}
		classrooms = append(classrooms, classroom)
	}

	return classrooms, nil
}

func GetFractionQuestions(minigame_id int, classroom_id int) ([]types.FractionQuestion, error) {
	var fractions []types.FractionQuestion

	rows, err := db.Query("SELECT question_id, fraction1_numerator, fraction1_denominator, fraction2_numerator, fraction2_denominator FROM fraction_questions WHERE minigame_id = ? AND classroom_id = ?", minigame_id, classroom_id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var fraction types.FractionQuestion
		if err := rows.Scan(&fraction.QuestionID, &fraction.Fraction1_Numerator, &fraction.Fraction1_Denominator, &fraction.Fraction2_Numerator, &fraction.Fraction2_Denominator); err != nil {
			return nil, err
		}
		fractions = append(fractions, fraction)
	}

	fmt.Print("we got fractions: ", fractions)

	return fractions, nil
}

func AddFractionQuestions(w http.ResponseWriter, r *http.Request, classroomID int) error {
	MinigameIDStr := r.FormValue("minigameID")
	Fraction1_NumeratorStr := r.FormValue("fraction1_numerator")
	Fraction1_DenominatorStr := r.FormValue("fraction1_denominator")
	Fraction2_NumeratorStr := r.FormValue("fraction2_numerator")
	Fraction2_DenominatorStr := r.FormValue("fraction2_denominator")

	MinigameID, _ := strconv.Atoi(MinigameIDStr)
	Fraction1_Numerator, _ := strconv.Atoi(Fraction1_NumeratorStr)
	Fraction1_Denominator, _ := strconv.Atoi(Fraction1_DenominatorStr)
	Fraction2_Numerator, _ := strconv.Atoi(Fraction2_NumeratorStr)
	Fraction2_Denominator, _ := strconv.Atoi(Fraction2_DenominatorStr)

	_, err := db.Exec("INSERT INTO fraction_questions (fraction1_numerator, fraction1_denominator, fraction2_numerator, fraction2_denominator, minigame_id, classroom_id) VALUES (?, ?, ?, ?, ?, ?)",
		Fraction1_Numerator, Fraction1_Denominator, Fraction2_Numerator, Fraction2_Denominator, MinigameID, classroomID)
	if err != nil {
		return err
	}

	return nil
}

func UpdateFractions(w http.ResponseWriter, r *http.Request) error {
	MinigameID, err := formInt(r, "minigame_id")
	if err != nil {
		return err
	}
	QuestionID, err := formInt(r, "question_id")
	if err != nil {
		return err
	}
	Fraction1_Numerator, err := formInt(r, "fraction1_numerator")
	if err != nil {
		return err
	}
	Fraction1_Denominator, err := formInt(r, "fraction1_denominator")
	if err != nil {
		return err
	}
	Fraction2_Numerator, err := formInt(r, "fraction2_numerator")
	if err != nil {
		return err
	}
	Fraction2_Denominator, err := formInt(r, "fraction2_denominator")
	if err != nil {
		return err
	}

	_, err = db.Exec("UPDATE fraction_questions SET fraction1_numerator = ?,  fraction1_denominator = ?, fraction2_numerator = ?, fraction2_denominator = ? WHERE minigame_id = ? AND question_id = ?",
		Fraction1_Numerator, Fraction1_Denominator, Fraction2_Numerator, Fraction2_Denominator, MinigameID, QuestionID)
	if err != nil {
		return err
	}

	return nil
}

func DeleteFractions(minigameID string, questionID string) error {
	// Execute the DELETE query
	_, err := db.Exec("DELETE FROM fraction_questions WHERE minigame_id = ? AND question_id = ?", minigameID, questionID)
	if err != nil {
		return err
	}

	return nil
}

func GetWordedQuestions(minigame_id int, classroom_id int) ([]types.FractionQuestion, error) {
	var questions []types.FractionQuestion

	// get questiontext and correct answer
	rows, err := db.Query("SELECT question_id, question_text, fraction1_numerator, fraction1_denominator, fraction2_numerator, fraction2_denominator FROM fraction_questions WHERE minigame_id = ?", minigame_id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var question types.FractionQuestion
		if err := rows.Scan(&question.QuestionID, &question.QuestionText, &question.Fraction1_Numerator, &question.Fraction1_Denominator, &question.Fraction2_Numerator, &question.Fraction2_Denominator); err != nil {
			return nil, err
		}
		questions = append(questions, question)
	}

	return questions, nil
}

func AddWordedQuestions(w http.ResponseWriter, r *http.Request, classroomID int) error {
	// get minigameID
	minigameIDStr := r.FormValue("minigameID")
	minigameID, _ := strconv.Atoi(minigameIDStr)

	questionText := r.FormValue("question_text")
	fraction1NumeratorStr := r.FormValue("fraction1_numerator")
	fraction1DenominatorStr := r.FormValue("fraction1_denominator")
	fraction2NumeratorStr := r.FormValue("fraction2_numerator")
	fraction2DenominatorStr := r.FormValue("fraction2_denominator")

	fraction1Numerator, _ := strconv.Atoi(fraction1NumeratorStr)
	fraction1Denominator, _ := strconv.Atoi(fraction1DenominatorStr)
	fraction2Numerator, _ := strconv.Atoi(fraction2NumeratorStr)
	fraction2Denominator, _ := strconv.Atoi(fraction2DenominatorStr)

	_, err := db.Exec("INSERT INTO fraction_questions (question_text, fraction1_numerator, fraction1_denominator, fraction2_numerator, fraction2_denominator, minigame_id, classroom_id) VALUES (?, ?, ?, ?, ?, ?, ?)",
		questionText, fraction1Numerator, fraction1Denominator, fraction2Numerator, fraction2Denominator, minigameID, classroomID)
	if err != nil {
		return err
	}

	return nil
}

func UpdateWordedQuestions(w http.ResponseWriter, r *http.Request) error {
	// parse form to fix bug where the values in request aren't retrieved.
	err := r.ParseForm()
	if err != nil {
		fmt.Println("Error parsing form:", err)
		return err
	}

	questionText := r.FormValue("question_text")

	minigameID, err := formInt(r, "minigameID")
	if err != nil {
		return err
	}
	classroomID, err := formInt(r, "classroomID")
	if err != nil {
		return err
	}
	questionID, err := formInt(r, "questionID")
	if err != nil {
		return err
	}
	fraction1Numerator, err := formInt(r, "fraction1_numerator")
	if err != nil {
		return err
	}
	fraction1Denominator, err := formInt(r, "fraction1_denominator")
	if err != nil {
		return err
	}
	fraction2Numerator, err := formInt(r, "fraction2_numerator")
	if err != nil {
		return err
	}
	fraction2Denominator, err := formInt(r, "fraction2_denominator")
	if err != nil {
		return err
	}

	_, err = db.Exec("UPDATE fraction_questions SET question_text = ?, fraction1_numerator = ?,  fraction1_denominator = ?, fraction2_numerator = ?, fraction2_denominator = ? WHERE minigame_id = ? AND question_id = ? AND classroom_id = ?",
		questionText, fraction1Numerator, fraction1Denominator, fraction2Numerator, fraction2Denominator, minigameID, questionID, classroomID)
	if err != nil {
		return err
	}

	return nil
}

func DeleteWorded(minigameID int, questionID int) error {
	// Execute the DELETE query
	result, err := db.Exec("DELETE FROM fraction_questions WHERE minigame_id = ? AND question_id = ?", minigameID, questionID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("DeleteWorded: no question found with minigame_id=%d question_id=%d", minigameID, questionID)
	}

	return nil
}

func GetQuizQuestions(minigame_id int, classroom_id int) ([]types.MultipleChoiceQuestion, error) {
	var questions []types.MultipleChoiceQuestion
	// get questiontext and correct answer
	rows, err := db.Query("SELECT question_id, question_text FROM multiple_choice_questions WHERE minigame_id = ? AND classroom_id = ?", minigame_id, classroom_id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var question types.MultipleChoiceQuestion
		if err := rows.Scan(&question.QuestionID, &question.QuestionText); err != nil {
			return nil, err
		}
		questions = append(questions, question)
	}

	// then we get choices
	for i, question := range questions {
		var choices []types.Choice
		choicesRow, err := db.Query("SELECT choice_id, choice_text, is_correct FROM multiple_choice_choices WHERE question_id = ?", question.QuestionID)
		if err != nil {
			return nil, err
		}
		defer choicesRow.Close()

		for choicesRow.Next() {
			var choice types.Choice
			if err := choicesRow.Scan(&choice.ChoiceID, &choice.ChoiceText, &choice.IsCorrect); err != nil {
				return nil, err
			}
			choices = append(choices, choice)
		}
		questions[i].Choices = choices
	}

	fmt.Println(questions)
	return questions, nil
}

func AddMCQuestions(w http.ResponseWriter, r *http.Request, classroomID int) error {
	minigameIDStr := r.FormValue("minigameID")
	minigameID, _ := strconv.Atoi(minigameIDStr)

	questionText := r.FormValue("question_text")

	// The "Add Question" form's correct-answer <select> submits the literal option
	// key ("option_1".."option_4"), not the option's text - the form fields share
	// that same naming, so we can match the correct answer by position rather than
	// by comparing choice text (which breaks if two options have identical text).
	optionKeys := []string{"option_1", "option_2", "option_3", "option_4"}
	correctAnswer := r.FormValue("correct_answer")

	// first insert question_text without the correct_answer id
	result, err := db.Exec(`INSERT INTO multiple_choice_questions (classroom_id, minigame_id, question_text) VALUES (?, ?, ?)`, classroomID, minigameID, questionText)
	if err != nil {
		return err
	}

	// Get the last inserted question_id
	questionID, _ := result.LastInsertId()

	// Insert choices into the multiple_choice_choices table using the questionID
	for _, key := range optionKeys {
		choiceText := r.FormValue(key)
		isCorrect := key == correctAnswer
		if _, err := db.Exec("INSERT INTO multiple_choice_choices (question_id, choice_text, is_correct) VALUES (?, ?, ?)", questionID, choiceText, isCorrect); err != nil {
			return err
		}
	}

	return nil
}

func UpdateMCQuestions(w http.ResponseWriter, r *http.Request) error {
	question := types.MultipleChoiceQuestion{
		QuestionText: r.FormValue("question"),
	}

	// correct_answer holds the choice_id of the option the teacher picked (see the
	// <select> in HandleGetMCQuestions), so compare by ID rather than by choice text -
	// text comparison breaks if two options happen to have identical text.
	correctAnswerID, err := formInt(r, "correct_answer")
	if err != nil {
		return err
	}
	// construct choices[]
	choices, err := constructChoices(r, correctAnswerID)
	if err != nil {
		return err
	}

	questionID, err := formInt(r, "questionID")
	if err != nil {
		return err
	}

	_, err = db.Exec("UPDATE multiple_choice_questions SET question_text = ? WHERE question_id = ?",
		question.QuestionText, questionID)
	if err != nil {
		return err
	}

	// given the choices[]
	// loop through, each choice gets to execute an update
	for _, choice := range choices {
		_, err = db.Exec("UPDATE multiple_choice_choices SET choice_text = ?, is_correct = ? WHERE choice_id = ?",
			choice.ChoiceText, choice.IsCorrect, choice.ChoiceID)
		if err != nil {
			return err
		}
	}

	return err
}

// helper func to construct choices[]
func constructChoices(r *http.Request, correctAnswerID int) ([]types.Choice, error) {
	var choices []types.Choice

	optionKeys := []string{"option1", "option2", "option3", "option4"}
	choiceIDKeys := []string{"option1_choiceID", "option2_choiceID", "option3_choiceID", "option4_choiceID"}

	for i, key := range optionKeys {
		var choice types.Choice
		choice.ChoiceText = r.FormValue(key)
		choiceID, err := formInt(r, choiceIDKeys[i])
		if err != nil {
			return nil, err
		}
		choice.ChoiceID = choiceID
		choice.IsCorrect = choice.ChoiceID == correctAnswerID
		choices = append(choices, choice)
	}

	return choices, nil
}

func DeleteMCQuestions(minigameID int, questionID int) error {
	// multiple_choice_choices.question_id and multiple_choice_responses.question_id/choice_id
	// are foreign keys with the default RESTRICT, so the parent question can't be deleted
	// while either still references it. Delete children first, in one transaction.
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM multiple_choice_responses WHERE question_id = ?", questionID); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM multiple_choice_choices WHERE question_id = ?", questionID); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM multiple_choice_questions WHERE minigame_id = ? AND question_id = ?", minigameID, questionID); err != nil {
		return err
	}

	return tx.Commit()
}

func AddQuizStatistics(classroomID int, minigameID int, student_id, score int) error {
	_, err := db.Exec("INSERT INTO multiple_choice_scores (classroom_id, minigame_id, student_id, score) VALUES (?, ?, ?, ?)", classroomID, minigameID, student_id, score)
	if err != nil {
		return err
	}

	return nil
}

// adds statistics for minigames 1 and 2 (simple fraction gameplay), also for substraction simple fraction gameplay
func AddFractionStatistics(w http.ResponseWriter, r *http.Request) error {
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
		ClassroomID        int `json:"classroom_id"`
		StudentID          int `json:"student_id"`
		QuestionID         int `json:"question_id"`
		MinigameID         int `json:"minigame_id"`
		Num_Right_Attempts int `json:"num_right_attempts"`
		Num_Wrong_Attempts int `json:"num_wrong_attempts"`
	}

	var data Data

	err = json.Unmarshal(body, &data)
	if err != nil {
		http.Error(w, "Failed to parse JSON", http.StatusBadRequest)
		return err
	}
	fmt.Print("we got statistics data: ", data)

	_, err = db.Exec("INSERT INTO fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts) VALUES (?, ?, ?, ?, ?, ?)", data.ClassroomID, data.MinigameID, data.QuestionID, data.StudentID, data.Num_Right_Attempts, data.Num_Wrong_Attempts)
	if err != nil {
		return err
	}

	return nil
}

// GetFractionResponseStatistics returns the count of right/wrong attempts for a single
// question. Used for both simple-fraction and worded-fraction minigames - both question
// types live in the same fraction_questions/fraction_responses tables, so one function
// serves both (GetWordedResponseStatistics used to be a byte-identical duplicate of this).
func GetFractionResponseStatistics(classroomID int, minigameID int, questionID int) ([]types.FractionClassStatistics, error) {
	var statistics []types.FractionClassStatistics

	// COALESCE: a bare SUM() with no matching rows still returns exactly one row, with
	// both columns NULL - which fails to Scan into an int. A freshly created question
	// with zero responses so far hits this on every load without the COALESCE.
	rows, err := db.Query("SELECT COALESCE(SUM(num_right_attempts), 0), COALESCE(SUM(num_wrong_attempts), 0) FROM fraction_responses WHERE classroom_id = ? AND minigame_id = ? AND question_id = ?", classroomID, minigameID, questionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var statistic types.FractionClassStatistics
		if err := rows.Scan(&statistic.RightAttemptsCount, &statistic.WrongAttemptsCount); err != nil {
			return nil, err
		}
		statistics = append(statistics, statistic)
	}

	return statistics, nil
}

func GetQuizClassStatistics(classroomID int, minigameID int) ([]types.QuizClassStatistics, error) {
	var statistics []types.QuizClassStatistics

	// get scores and count per score
	rows, err := db.Query("SELECT score, COUNT(*) AS count_per_score FROM multiple_choice_scores WHERE classroom_id = ? AND minigame_id = ? GROUP BY score ORDER BY score", classroomID, minigameID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var statistic types.QuizClassStatistics
		if err := rows.Scan(&statistic.Score, &statistic.Count); err != nil {
			return nil, err
		}
		statistics = append(statistics, statistic)
	}
	fmt.Print("returned class statistics contains: ", statistics)

	return statistics, nil
}

func GetQuizResponseStatistics(classroomID int, minigameID int, questionID int) ([]types.QuizResponseStatistics, error) {
	var responseStatistics []types.QuizResponseStatistics

	rows, err := db.Query(`
			SELECT
			c.choice_text,
			COUNT(r.choice_id) AS response_count
		FROM
			multiple_choice_choices AS c
		LEFT JOIN
			multiple_choice_responses AS r ON c.choice_id = r.choice_id
			AND r.question_id = ?
			AND r.minigame_id = ?
			AND r.classroom_id = ?
		WHERE
			c.question_id = ?
		GROUP BY
			c.choice_id, c.choice_text;
	`, questionID, minigameID, classroomID, questionID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var statistic types.QuizResponseStatistics
		if err := rows.Scan(&statistic.Choice, &statistic.Count); err != nil {
			return nil, err
		}
		responseStatistics = append(responseStatistics, statistic)
	}

	return responseStatistics, nil
}

func AddQuizResponse(classroomID int, minigameID int, questionID int, studentID int, choiceID int) error {
	_, err := db.Exec("INSERT INTO multiple_choice_responses (classroom_id, minigame_id, question_id, student_id, choice_id) VALUES (?, ?, ?, ?, ?)", classroomID, minigameID, questionID, studentID, choiceID)
	if err != nil {
		return err
	}
	return nil
}

func GetSavedData(studentID int) (types.SaveData, error) {
	var save_data types.SaveData
	var badges types.Badges

	// Get all saved data except badges
	row := db.QueryRow("SELECT student_id, current_floor, current_quest, saved_scene, vector_x, vector_y, first_time_init_floor1, first_time_init_floor2, first_time_init_floor3 FROM save_states WHERE student_id = ?", studentID)
	err := row.Scan(&save_data.StudentID, &save_data.CurrentFloor, &save_data.CurrentQuest, &save_data.SavedScene, &save_data.VectorX, &save_data.VectorY, &save_data.FirstTimeInitFloor1, &save_data.FirstTimeInitFloor2, &save_data.FirstTimeInitFloor3)
	if err != nil {
		return save_data, err
	}

	// Retrieve all badges
	row = db.QueryRow("SELECT badge_rock, badge_bowl, badge_carrot, badge_cake, badge_sword, badge_mushroom, badge_bucket1, badge_flask, badge_bucket2, badge_bucket3, badge_crystal_ball, badge_shell, badge_original_robot FROM save_states WHERE student_id = ?", studentID)
	err = row.Scan(&badges.ShinyRock, &badges.Bowl, &badges.Carrot, &badges.Cake, &badges.Sword, &badges.Mushroom, &badges.Bucket1, &badges.Flask, &badges.Bucket2, &badges.Bucket3, &badges.CrystalBall, &badges.Shell, &badges.OriginalRobot)
	if err != nil {
		return save_data, err
	}

	// retrieve actionables
	row = db.QueryRow("SELECT rock_removed, disable_rock_removed, raket_sneaking_quest_complete, unlock_cave_collision, raket_sword_complete, raket_quest_progress, disable_dead_robot_quest, do_raket_blacksmith_animation, sword_bottom, sword_guard, sword_lower_blade, sword_middle_blade, sword_top_blade, disable_raket_stealing_quest, disable_fresh_dialogue_quest, disable_water_logged_1_quest, disable_water_logged_2_quest, disable_water_logged_3_quest, disable_chip_quest, disable_rat_wizard_training_quest FROM save_states WHERE student_id = ?", studentID)
	err = row.Scan(&save_data.RockRemoved, &save_data.DisableRockRemoved, &save_data.RaketSneakingQuestComplete, &save_data.UnlockCaveCollision, &save_data.RaketSwordComplete, &save_data.RaketQuestProgress, &save_data.DisableDeadRobotQuest, &save_data.DoRaketBlacksmithAnimation, &save_data.SwordBottom, &save_data.SwordGuard, &save_data.SwordLowerBlade, &save_data.SwordMiddleBlade, &save_data.SwordTopBlade, &save_data.DisableRaketStealingQuest, &save_data.DisableFreshDialogueQuest, &save_data.DisableWaterLogged1Quest, &save_data.DisableWaterLogged2Quest, &save_data.DisableWaterLogged3Quest, &save_data.DisableChipQuest, &save_data.DisableRatWizardTrainingQuest)
	if err != nil {
		return save_data, err
	}

	save_data.PlayerBadges = badges

	fmt.Print("saved data is: ", save_data)

	return save_data, nil
}

func GetStudentScores(classroomID int, minigameID int) ([]types.StudentQuizScore, error) {
	var studentScores []types.StudentQuizScore

	rows, err := db.Query("SELECT u.firstname, u.lastname, mcs.score FROM multiple_choice_scores AS mcs JOIN users AS u ON mcs.student_id = u.user_id WHERE mcs.classroom_id = ? AND mcs.minigame_id = ? ORDER BY mcs.score DESC", classroomID, minigameID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var student types.StudentQuizScore
		if err := rows.Scan(&student.FirstName, &student.LastName, &student.Score); err != nil {
			return nil, fmt.Errorf("GetStudentScores: %v", err)
		}
		studentScores = append(studentScores, student)
	}

	return studentScores, nil
}

func GetStudentFractionStatistics(userID int, minigameID int) ([]types.StudentFractionStatistics, error) {
	var statistics []types.StudentFractionStatistics

	rows, err := db.Query("SELECT fq.fraction1_numerator AS f1num, fq.fraction1_denominator AS f1den, fq.fraction2_numerator AS f2num, fq.fraction2_denominator AS f2den, IFNULL(fr.num_wrong_attempts, 0) AS num_wrong, IFNULL(fr.num_right_attempts, 0) AS num_right FROM fraction_questions fq LEFT JOIN fraction_responses fr ON fq.question_id = fr.question_id AND fr.student_id = ? AND fr.minigame_id = ? WHERE fq.minigame_id = ?", userID, minigameID, minigameID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var statistic types.StudentFractionStatistics
		if err := rows.Scan(&statistic.Fraction1_Numerator, &statistic.Fraction1_Denominator, &statistic.Fraction2_Numerator, &statistic.Fraction2_Denominator, &statistic.WrongAttemptsCount, &statistic.RightAttemptsCount); err != nil {
			return nil, fmt.Errorf("GetStudentScores: %v", err)
		}
		statistics = append(statistics, statistic)
	}

	return statistics, nil
}

func GetStudentWordedStatistics(userID int, minigameID int) ([]types.StudentFractionStatistics, error) {
	var statistics []types.StudentFractionStatistics

	rows, err := db.Query("SELECT fq.question_text, IFNULL(fr.num_wrong_attempts, 0) AS num_wrong, IFNULL(fr.num_right_attempts, 0) AS num_right FROM fraction_questions fq LEFT JOIN fraction_responses fr ON fq.question_id = fr.question_id AND fr.student_id = ? AND fr.minigame_id = ? WHERE fq.minigame_id = ?", userID, minigameID, minigameID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var statistic types.StudentFractionStatistics
		if err := rows.Scan(&statistic.QuestionText, &statistic.WrongAttemptsCount, &statistic.RightAttemptsCount); err != nil {
			return nil, fmt.Errorf("GetStudentScores: %v", err)
		}
		statistics = append(statistics, statistic)
	}

	// fmt.Print(statistics)

	return statistics, nil
}

func GetStudentQuizStatistics(userID int, minigameID int) ([]types.StudentQuizStatistics, error) {
	var statistics []types.StudentQuizStatistics

	// One query replaces what used to be three (questions, then correct answers via a
	// dynamically-built IN (...), then the student's answers) plus a manual join in Go.
	// The IN (...) version broke with a SQL syntax error whenever a minigame had zero
	// questions (an empty placeholder list builds "IN ()"); grouping by question here
	// sidesteps that failure mode entirely instead of needing a special case for it.
	// COALESCE covers a question with no choice marked correct, and a question the
	// student hasn't answered - both would otherwise scan as SQL NULL into a Go string.
	rows, err := db.Query(`
		SELECT
			q.question_id,
			q.question_text,
			COALESCE(MAX(CASE WHEN c.is_correct THEN c.choice_text END), '') AS correct_answer,
			COALESCE(MAX(CASE WHEN r.choice_id = c.choice_id THEN c.choice_text END), '') AS user_answer
		FROM multiple_choice_questions q
		JOIN multiple_choice_choices c ON c.question_id = q.question_id
		LEFT JOIN multiple_choice_responses r ON r.question_id = q.question_id
			AND r.student_id = ? AND r.minigame_id = ?
		WHERE q.minigame_id = ?
		GROUP BY q.question_id, q.question_text
		ORDER BY q.question_id
	`, userID, minigameID, minigameID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var questionID int
		var statistic types.StudentQuizStatistics
		if err := rows.Scan(&questionID, &statistic.QuestionText, &statistic.CorrectAnswer, &statistic.UserAnswer); err != nil {
			return nil, fmt.Errorf("failed to scan student quiz statistic: %v", err)
		}
		if statistic.UserAnswer == statistic.CorrectAnswer {
			statistic.Score = 1
		}
		statistics = append(statistics, statistic)
	}

	return statistics, nil
}

func SaveData(data types.SaveData) error {

	_, err := db.Exec(`
    UPDATE save_states
    SET 
        current_floor = ?, 
        current_quest = ?, 
        saved_scene = ?, 
        vector_x = ?, 
        vector_y = ?, 
        rock_removed = ?,
        disable_rock_removed = ?,
        raket_sneaking_quest_complete = ?,
        unlock_cave_collision = ?,
        raket_sword_complete = ?,
        raket_quest_progress = ?,
        do_raket_blacksmith_animation = ?,
        sword_bottom = ?,
        sword_guard = ?,
        sword_lower_blade = ?,
        sword_middle_blade = ?,
        sword_top_blade = ?,
        badge_rock = ?, 
        badge_bowl = ?, 
        badge_carrot = ?, 
        badge_cake = ?, 
        badge_sword = ?, 
        badge_mushroom = ?, 
        badge_bucket1 = ?, 
        badge_flask = ?, 
        badge_bucket2 = ?, 
        badge_bucket3 = ?, 
        badge_crystal_ball = ?, 
        badge_shell = ?,
        badge_original_robot = ?, 
        first_time_init_floor1 = ?, 
        first_time_init_floor2 = ?, 
        first_time_init_floor3 = ?, 
        disable_dead_robot_quest = ?, 
        disable_raket_stealing_quest = ?, 
        disable_fresh_dialogue_quest = ?, 
        disable_water_logged_1_quest = ?, 
        disable_water_logged_2_quest = ?, 
        disable_water_logged_3_quest = ?, 
        disable_chip_quest = ?, 
        disable_rat_wizard_training_quest = ?
    WHERE student_id = ?
`,
		data.CurrentFloor, data.CurrentQuest, data.SavedScene, data.VectorX, data.VectorY,
		data.RockRemoved, data.DisableRockRemoved, data.RaketSneakingQuestComplete, data.UnlockCaveCollision,
		data.RaketSwordComplete, data.RaketQuestProgress, data.DoRaketBlacksmithAnimation, data.SwordBottom,
		data.SwordGuard, data.SwordLowerBlade, data.SwordMiddleBlade, data.SwordTopBlade,
		data.PlayerBadges.ShinyRock, data.PlayerBadges.Bowl, data.PlayerBadges.Carrot, data.PlayerBadges.Cake,
		data.PlayerBadges.Sword, data.PlayerBadges.Mushroom, data.PlayerBadges.Bucket1, data.PlayerBadges.Flask,
		data.PlayerBadges.Bucket2, data.PlayerBadges.Bucket3, data.PlayerBadges.CrystalBall, data.PlayerBadges.Shell,
		data.PlayerBadges.OriginalRobot, data.FirstTimeInitFloor1, data.FirstTimeInitFloor2, data.FirstTimeInitFloor3,
		data.DisableDeadRobotQuest, data.DisableRaketStealingQuest, data.DisableFreshDialogueQuest,
		data.DisableWaterLogged1Quest, data.DisableWaterLogged2Quest, data.DisableWaterLogged3Quest,
		data.DisableChipQuest, data.DisableRatWizardTrainingQuest,
		data.StudentID,
	)

	if err != nil {
		return err
	}

	fmt.Print("Update Save State Success!")
	return nil
}
