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
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
)

var db *sql.DB

const (
	sessionUserKey        = "teacher"
	sessionAccessTokenKey = "access_token"
)

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

	// get section name given classroomID
	section, err := GetSection(classroomID)
	if err != nil {
		return nil, err
	}

	// get students given classroomID
	rows, err := db.Query("SELECT user_id, firstname, lastname FROM users WHERE section = ? AND usertype = ? AND user_id NOT IN (SELECT student_id FROM enrollments WHERE classroom_id = ?)", section, "student", classroomID)
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

func UnenrollStudent(studentID int) error {
	// Execute the DELETE query
	_, err := db.Exec("DELETE FROM enrollments WHERE student_id = ?", studentID)
	if err != nil {
		return err
	}
	return nil
}

func InsertClassroom(w http.ResponseWriter, r *http.Request) error {
	classroom := types.Classroom{
		ClassroomName: r.FormValue("classname"),
		Section:       r.FormValue("section"),
		Description:   r.FormValue("description"),
	}

	// then get teacherID from session
	store := sessions.NewCookieStore([]byte(os.Getenv("SESSION_SECRET")))
	session, _ := store.Get(r, sessionUserKey)
	teacherID := session.Values["teacherID"].(int)

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
	MinigameIDStr := r.FormValue("minigame_id")
	QuestionIDStr := r.FormValue("question_id")
	Fraction1_NumeratorStr := r.FormValue("fraction1_numerator")
	Fraction1_DenominatorStr := r.FormValue("fraction1_denominator")
	Fraction2_NumeratorStr := r.FormValue("fraction2_numerator")
	Fraction2_DenominatorStr := r.FormValue("fraction2_denominator")

	MinigameID, _ := strconv.Atoi(MinigameIDStr)
	QuestionID, _ := strconv.Atoi(QuestionIDStr)
	Fraction1_Numerator, _ := strconv.Atoi(Fraction1_NumeratorStr)
	Fraction1_Denominator, _ := strconv.Atoi(Fraction1_DenominatorStr)
	Fraction2_Numerator, _ := strconv.Atoi(Fraction2_NumeratorStr)
	Fraction2_Denominator, _ := strconv.Atoi(Fraction2_DenominatorStr)

	_, err := db.Exec("UPDATE fraction_questions SET fraction1_numerator = ?,  fraction1_denominator = ?, fraction2_numerator = ?, fraction2_denominator = ? WHERE minigame_id = ? AND question_id = ?",
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
	minigameIDStr := r.FormValue("minigame_id")
	questionIDStr := r.FormValue("question_id")
	questionText := r.FormValue("question_text")
	fraction1NumeratorStr := r.FormValue("fraction1_numerator")
	fraction1DenominatorStr := r.FormValue("fraction1_denominator")
	fraction2NumeratorStr := r.FormValue("fraction2_numerator")
	fraction2DenominatorStr := r.FormValue("fraction2_denominator")

	minigameID, _ := strconv.Atoi(minigameIDStr)
	questionID, _ := strconv.Atoi(questionIDStr)
	fraction1Numerator, _ := strconv.Atoi(fraction1NumeratorStr)
	fraction1Denominator, _ := strconv.Atoi(fraction1DenominatorStr)
	fraction2Numerator, _ := strconv.Atoi(fraction2NumeratorStr)
	fraction2Denominator, _ := strconv.Atoi(fraction2DenominatorStr)

	_, err := db.Exec("UPDATE fraction_questions SET question_text = ?, fraction1_numerator = ?,  fraction1_denominator = ?, fraction2_numerator = ?, fraction2_denominator = ? WHERE minigame_id = ? AND question_id = ?",
		questionText, fraction1Numerator, fraction1Denominator, fraction2Numerator, fraction2Denominator, minigameID, questionID)
	if err != nil {
		return err
	}

	return nil
}

func DeleteWorded(minigameID int, questionID int) error {
	// Execute the DELETE query
	_, err := db.Exec("DELETE FROM fraction_questions WHERE minigame_id = ? AND question_id = ?", minigameID, questionID)
	if err != nil {
		return err
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

	question := types.MultipleChoiceQuestion{
		QuestionText: r.FormValue("question_text"),
	}

	var choices []string
	choices = append(choices, r.FormValue("option_1"))
	choices = append(choices, r.FormValue("option_2"))
	choices = append(choices, r.FormValue("option_3"))
	choices = append(choices, r.FormValue("option_4"))

	correctAnswer := r.FormValue("correct_answer")

	// first insert question_text without the correct_answer id
	result, err := db.Exec(`INSERT INTO multiple_choice_questions (classroom_id, minigame_id, question_text) VALUES (?, ?, ?)`, classroomID, minigameID, question.QuestionText)
	if err != nil {
		return err
	}

	// Get the last inserted question_id
	questionID, _ := result.LastInsertId()
	var correct_answer_id int64

	// Insert choices into the multiple_choice_choices table using the questionID
	for _, choiceText := range choices {
		// to retrieve the choice_id of correct answer
		if choiceText == correctAnswer {
			result, err := db.Exec("INSERT INTO multiple_choice_choices (question_id, choice_text, is_correct) VALUES (?, ?, ?)", questionID, choiceText, true)
			if err != nil {
				return err
			}
			correct_answer_id, _ = result.LastInsertId()
			// else if choice is not the correct answer
		} else {
			_, err := db.Exec("INSERT INTO multiple_choice_choices (question_id, choice_text) VALUES (?, ?)", questionID, choiceText)
			if err != nil {
				return err
			}
		}
	}
	// // insert choice_id of correct_answer into table
	// _, err = db.Exec(`INSERT INTO multiple_choice_questions (correct_answer) VALUES (?)`, correct_answer_id)
	// if err != nil {
	// 	return err
	// }
	fmt.Print(correct_answer_id)

	return nil
}

func UpdateMCQuestions(w http.ResponseWriter, r *http.Request) error {
	question := types.MultipleChoiceQuestion{
		QuestionText: r.FormValue("question"),
	}

	correctAnswer := r.FormValue("correct_answer")
	// construct choices[]
	choices := constructChoices(r, correctAnswer)

	questionIDStr := r.FormValue("question_id")
	questionID, _ := strconv.Atoi(questionIDStr)

	_, err := db.Exec("UPDATE multiple_choice_questions SET question_text = ? WHERE question_id = ?",
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
func constructChoices(r *http.Request, correctAnswer string) []types.Choice {
	var choices []types.Choice
	var choice types.Choice

	option1 := r.FormValue("option1")
	choice.ChoiceText = option1
	choice.ChoiceID, _ = strconv.Atoi(r.FormValue("option1_choiceID"))
	choice.IsCorrect = getCorrectAnswer(option1, correctAnswer)
	choices = append(choices, choice)

	option2 := r.FormValue("option2")
	choice.ChoiceText = option2
	choice.ChoiceID, _ = strconv.Atoi(r.FormValue("option2_choiceID"))
	choice.IsCorrect = getCorrectAnswer(option2, correctAnswer)
	choices = append(choices, choice)

	option3 := r.FormValue("option3")
	choice.ChoiceText = option3
	choice.ChoiceID, _ = strconv.Atoi(r.FormValue("option3_choiceID"))
	choice.IsCorrect = getCorrectAnswer(option3, correctAnswer)
	choices = append(choices, choice)

	option4 := r.FormValue("option4")
	choice.ChoiceText = option4
	choice.ChoiceID, _ = strconv.Atoi(r.FormValue("option4_choiceID"))
	choice.IsCorrect = getCorrectAnswer(option4, correctAnswer)
	choices = append(choices, choice)

	return choices
}

// helper func for construct choices to get correct answer
func getCorrectAnswer(option string, correctAnswer string) bool {
	if option == correctAnswer {
		return true
	}
	return false
}

func DeleteMCQuestions(minigameID int, questionID int) error {
	// Execute the DELETE query
	_, err := db.Exec("DELETE FROM multiple_choice_questions WHERE minigame_id = ? AND question_id = ?", minigameID, questionID)
	if err != nil {
		return err
	}

	return nil

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

func GetFractionResponseStatistics(classroomID int, minigameID int, questionID int) ([]types.FractionClassStatistics, error) {
	var statistics []types.FractionClassStatistics

	// get count of right and wrong responses
	rows, err := db.Query("SELECT SUM(num_right_attempts), SUM(num_wrong_attempts) FROM fraction_responses WHERE classroom_id = ? AND minigame_id = ? AND question_id = ?", classroomID, minigameID, questionID)
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

func GetWordedResponseStatistics(classroomID int, minigameID int, questionID int) ([]types.FractionClassStatistics, error) {
	// we can reuse types.FractionClassStatistics because they're the same structure (number of right or wrong attempts)
	var statistics []types.FractionClassStatistics

	// get count of right and wrong responses
	rows, err := db.Query("SELECT SUM(num_right_attempts), SUM(num_wrong_attempts) FROM fraction_responses WHERE classroom_id = ? AND minigame_id = ? AND question_id = ?", classroomID, minigameID, questionID)
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
			c.choice_text;
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

	// Get all questions for the given minigameID
	questionsQuery := `SELECT question_id, question_text FROM multiple_choice_questions WHERE minigame_id = ?`
	questionRows, err := db.Query(questionsQuery, minigameID)
	if err != nil {
		return nil, err
	}
	defer questionRows.Close()

	var questions []struct {
		QuestionID   int
		QuestionText string
	}

	for questionRows.Next() {
		var q struct {
			QuestionID   int
			QuestionText string
		}
		if err := questionRows.Scan(&q.QuestionID, &q.QuestionText); err != nil {
			return nil, fmt.Errorf("failed to scan question: %v", err)
		}
		questions = append(questions, q)
	}

	// Generate placeholders and get correct answers
	var questionIDs []int
	for _, q := range questions {
		questionIDs = append(questionIDs, q.QuestionID)
	}

	// Generate placeholders for the IN clause based on the number of question IDs
	placeholders := make([]string, len(questionIDs))
	for i := range placeholders {
		placeholders[i] = "?"
	}
	placeholdersStr := strings.Join(placeholders, ",")

	// Query for correct answers using the dynamically generated placeholders
	correctAnswersQuery := fmt.Sprintf(
		`SELECT mcc.question_id, mcc.choice_text 
		FROM multiple_choice_choices mcc
		WHERE mcc.is_correct = TRUE AND mcc.question_id IN (%s)`, placeholdersStr)

	correctAnswersRows, err := db.Query(correctAnswersQuery, convertToInterfaceSlice(questionIDs)...)
	if err != nil {
		return nil, err
	}
	defer correctAnswersRows.Close()

	var correctAnswers = make(map[int]string)
	for correctAnswersRows.Next() {
		var questionID int
		var choiceText string
		if err := correctAnswersRows.Scan(&questionID, &choiceText); err != nil {
			return nil, fmt.Errorf("failed to scan correct answer: %v", err)
		}
		correctAnswers[questionID] = choiceText
	}

	// Get the user's answers for those questions
	userAnswersQuery := `
		SELECT mcr.question_id, mcc.choice_text 
		FROM multiple_choice_responses mcr
		JOIN multiple_choice_choices mcc ON mcr.choice_id = mcc.choice_id
		WHERE mcr.student_id = ? AND mcr.minigame_id = ?`

	userAnswersRows, err := db.Query(userAnswersQuery, userID, minigameID)
	if err != nil {
		return nil, err
	}
	defer userAnswersRows.Close()

	var userAnswers = make(map[int]string)
	for userAnswersRows.Next() {
		var questionID int
		var choiceText string
		if err := userAnswersRows.Scan(&questionID, &choiceText); err != nil {
			return nil, fmt.Errorf("failed to scan user answer: %v", err)
		}
		userAnswers[questionID] = choiceText
	}

	// Combine the results and calculate the score
	for _, question := range questions {
		userAnswer := userAnswers[question.QuestionID]
		correctAnswer := correctAnswers[question.QuestionID]
		score := 0
		if userAnswer == correctAnswer {
			score = 1
		}

		statistic := types.StudentQuizStatistics{
			QuestionText:  question.QuestionText,
			CorrectAnswer: correctAnswer,
			UserAnswer:    userAnswer,
			Score:         score,
		}
		statistics = append(statistics, statistic)
	}

	return statistics, nil
}

// Helper function to convert []int to []interface{}
func convertToInterfaceSlice(slice []int) []interface{} {
	interfaceSlice := make([]interface{}, len(slice))
	for i, v := range slice {
		interfaceSlice[i] = v
	}
	return interfaceSlice
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
