package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"soln-teachermodule/types"
	"strconv"
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
	row := db.QueryRow("SELECT password FROM users WHERE username = ?", username)
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

// PROPER ERROR HANDLING TO BE IMPLEMENTED
func AuthenticateGameUser(username string, password string) bool {

	var storedPassword string
	row := db.QueryRow("SELECT password FROM users WHERE username = ?", username)
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
	}
	fmt.Println("Login Success! Hello ", username, "!")
	// Authentication successful
	return true
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
		return nil
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return nil
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

	_, err = db.Exec("INSERT INTO students (username, firstname, lastname, section, class_number, password) VALUES (?, ?, ?, ?, ?, ?)", data.Username, data.FirstName, data.Lastname, data.Section, data.ClassNumber, data.Password)
	if err != nil {
		return err
	}

	return nil
}

func GetStudents(classroomID int) ([]types.Student, error) {
	var students []types.Student
	// get students given classroomID
	rows, err := db.Query("SELECT students.username, students.student_id FROM enrollments e JOIN students ON e.student_id = students.student_id WHERE e.classroom_id = ? ", classroomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var student types.Student
		if err := rows.Scan(&student.Username, &student.UserID); err != nil {
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
	rows, err := db.Query("SELECT student_id, username FROM students WHERE section = ? AND student_id NOT IN (SELECT student_id FROM enrollments WHERE classroom_id = ?)", section, classroomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var student types.Student
		if err := rows.Scan(&student.UserID, &student.Username); err != nil {
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

func DeleteStudent(studentID int) error {
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
	err := db.QueryRow("SELECT user_ID FROM users WHERE username = ?", userCreds.Username).Scan(&teacherID)

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

func GetFractionQuestions(minigame_id int) ([]types.FractionQuestion, error) {
	var fractions []types.FractionQuestion

	rows, err := db.Query("SELECT question_id, fraction1_numerator, fraction1_denominator, fraction2_numerator, fraction2_denominator FROM fraction_questions WHERE minigame_id = ?", minigame_id)
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

	return fractions, nil
}

func AddFractionQuestions(w http.ResponseWriter, r *http.Request) error {
	MinigameIDStr := r.FormValue("minigame_id")
	Fraction1_NumeratorStr := r.FormValue("fraction1_numerator")
	Fraction1_DenominatorStr := r.FormValue("fraction1_denominator")
	Fraction2_NumeratorStr := r.FormValue("fraction2_numerator")
	Fraction2_DenominatorStr := r.FormValue("fraction2_denominator")

	MinigameID, _ := strconv.Atoi(MinigameIDStr)
	Fraction1_Numerator, _ := strconv.Atoi(Fraction1_NumeratorStr)
	Fraction1_Denominator, _ := strconv.Atoi(Fraction1_DenominatorStr)
	Fraction2_Numerator, _ := strconv.Atoi(Fraction2_NumeratorStr)
	Fraction2_Denominator, _ := strconv.Atoi(Fraction2_DenominatorStr)

	_, err := db.Exec("INSERT INTO fraction_questions (fraction1_numerator, fraction1_denominator, fraction2_numerator, fraction2_denominator, minigame_id) VALUES (?, ?, ?, ?, ?)",
		Fraction1_Numerator, Fraction1_Denominator, Fraction2_Numerator, Fraction2_Denominator, MinigameID)
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

func GetWordedQuestions(minigame_id int) ([]types.WordedQuestion, error) {
	var questions []types.WordedQuestion

	// get questiontext and correct answer
	rows, err := db.Query("SELECT question_id, question_text, fraction1_numerator, fraction1_denominator, fraction2_numerator, fraction2_denominator FROM worded_questions WHERE minigame_id = ?", minigame_id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var question types.WordedQuestion
		if err := rows.Scan(&question.QuestionID, &question.QuestionText, &question.Fraction1_Numerator, &question.Fraction1_Denominator, &question.Fraction2_Numerator, &question.Fraction2_Denominator); err != nil {
			return nil, err
		}
		questions = append(questions, question)
	}

	return questions, nil
}

func AddWordedQuestions(w http.ResponseWriter, r *http.Request) error {
	minigameIDStr := r.FormValue("minigame_id")
	questionText := r.FormValue("question_text")
	fraction1NumeratorStr := r.FormValue("fraction1_numerator")
	fraction1DenominatorStr := r.FormValue("fraction1_denominator")
	fraction2NumeratorStr := r.FormValue("fraction2_numerator")
	fraction2DenominatorStr := r.FormValue("fraction2_denominator")

	fmt.Print("addword from db is executes")

	minigameID, _ := strconv.Atoi(minigameIDStr)
	fraction1Numerator, _ := strconv.Atoi(fraction1NumeratorStr)
	fraction1Denominator, _ := strconv.Atoi(fraction1DenominatorStr)
	fraction2Numerator, _ := strconv.Atoi(fraction2NumeratorStr)
	fraction2Denominator, _ := strconv.Atoi(fraction2DenominatorStr)

	_, err := db.Exec("INSERT INTO worded_questions (question_text, fraction1_numerator, fraction1_denominator, fraction2_numerator, fraction2_denominator, minigame_id) VALUES (?, ?, ?, ?, ?, ?)",
		questionText, fraction1Numerator, fraction1Denominator, fraction2Numerator, fraction2Denominator, minigameID)
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

	_, err := db.Exec("UPDATE worded_questions SET question_text = ?, fraction1_numerator = ?,  fraction1_denominator = ?, fraction2_numerator = ?, fraction2_denominator = ? WHERE minigame_id = ? AND question_id = ?",
		questionText, fraction1Numerator, fraction1Denominator, fraction2Numerator, fraction2Denominator, minigameID, questionID)
	if err != nil {
		return err
	}

	return nil
}

func DeleteWorded(minigameID int, questionID int) error {
	// Execute the DELETE query
	_, err := db.Exec("DELETE FROM worded_questions WHERE minigame_id = ? AND question_id = ?", minigameID, questionID)
	if err != nil {
		return err
	}

	return nil
}

func GetQuestionDictionary(minigame_id int) ([]types.MultipleChoiceQuestion, error) {
	var questions []types.MultipleChoiceQuestion
	// get questiontext and correct answer
	rows, err := db.Query("SELECT question_id, question_text, correct_answer FROM multiple_choice_questions WHERE minigame_id = ?", minigame_id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var question types.MultipleChoiceQuestion
		if err := rows.Scan(&question.QuestionID, &question.QuestionText, &question.CorrectAnswer); err != nil {
			return nil, err
		}
		questions = append(questions, question)
	}

	for i := 0; i < len(questions); i++ {
		// then we get choices
		choicesRows, err := db.Query("SELECT option_1, option_2, option_3, option_4 FROM multiple_choice_choices WHERE question_id = ?", questions[i].QuestionID)
		if err != nil {
			return nil, err
		}
		defer choicesRows.Close()

		// then add choices to question in questions slice
		for choicesRows.Next() {
			if err := choicesRows.Scan(&questions[i].Option1, &questions[i].Option2, &questions[i].Option3, &questions[i].Option4); err != nil {
				return nil, err
			}
		}
	}

	// fmt.Println(questions)
	return questions, nil
}

func AddMCQuestions(w http.ResponseWriter, r *http.Request) error {
	MinigameIDStr := r.FormValue("minigame_id")
	minigameID, _ := strconv.Atoi(MinigameIDStr)
	question := types.MultipleChoiceQuestion{
		QuestionText:  r.FormValue("question_text"),
		Option1:       r.FormValue("option_1"),
		Option2:       r.FormValue("option_2"),
		Option3:       r.FormValue("option_3"),
		Option4:       r.FormValue("option_4"),
		CorrectAnswer: r.FormValue("correct_answer"),
	}
	var correctAnswer = ""
	// get correct answer
	if question.CorrectAnswer == "option_1" {
		correctAnswer = question.Option1
	} else if question.CorrectAnswer == "option_2" {
		correctAnswer = question.Option2
	} else if question.CorrectAnswer == "option_3" {
		correctAnswer = question.Option3
	} else if question.CorrectAnswer == "option_4" {
		correctAnswer = question.Option4
	}

	result, err := db.Exec(`INSERT INTO multiple_choice_questions (minigame_id, question_text, correct_answer) VALUES (?, ?, ?)`, minigameID, question.QuestionText, correctAnswer)
	if err != nil {
		return err
	}

	// Get the last inserted question_id
	questionID, err := result.LastInsertId()
	if err != nil {
		return err
	}

	// Insert choices into the multiple_choice_choices table using the questionID
	_, err = db.Exec(`INSERT INTO multiple_choice_choices (question_id, option_1, option_2, option_3, option_4) VALUES (?, ?, ?, ?, ?)`, questionID, question.Option1, question.Option2, question.Option3, question.Option4)
	if err != nil {
		return err
	}

	return nil
}

func UpdateMCQuestions(w http.ResponseWriter, r *http.Request) error {
	question := types.MultipleChoiceQuestion{
		QuestionText:  r.FormValue("question"),
		Option1:       r.FormValue("option1"),
		Option2:       r.FormValue("option2"),
		Option3:       r.FormValue("option3"),
		Option4:       r.FormValue("option4"),
		CorrectAnswer: r.FormValue("correct_answer"),
	}

	fmt.Print("we got correct answer: ", question.CorrectAnswer)

	// get question id
	questionIDStr := r.FormValue("question_id")
	questionID, _ := strconv.Atoi(questionIDStr)

	_, err := db.Exec("UPDATE multiple_choice_questions SET question_text = ?,  correct_answer = ? WHERE question_id = ?",
		question.QuestionText, question.CorrectAnswer, questionID)
	if err != nil {
		log.Fatal(err)
	}

	_, another_err := db.Exec("UPDATE multiple_choice_choices SET option_1 = ?, option_2 = ?, option_3 = ?, option_4 = ? WHERE question_id = 1",
		question.QuestionText, question.Option1, question.Option2, question.Option3, question.Option4, question.CorrectAnswer)
	if err != nil {
		log.Fatal(another_err)
	}

	return err
}

func UpdateStatistics(w http.ResponseWriter, r *http.Request) error {
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
		Username   string
		MinigameID int
		Score      int
	}

	var data Data

	err = json.Unmarshal(body, &data)
	if err != nil {
		http.Error(w, "Failed to parse JSON", http.StatusBadRequest)
		return err
	}
	fmt.Print("we got statistics data: ", data)

	_, err = db.Exec("INSERT INTO statistics (username, minigameID, score) VALUES (?, ?, ?)", data.Username, data.MinigameID, data.Score)
	if err != nil {
		return err
	}

	return nil
}

func UpdateSaisaiStatistics(w http.ResponseWriter, r *http.Request) error {
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
		Username           string
		Question_id        int
		Minigame_id        int
		Num_Right_Attempts int
		Num_Wrong_Attempts int
	}

	var data Data

	err = json.Unmarshal(body, &data)
	if err != nil {
		http.Error(w, "Failed to parse JSON", http.StatusBadRequest)
		return err
	}
	fmt.Print("we got statistics data: ", data)

	_, err = db.Exec("INSERT INTO saisai_statistics (username, question_id, minigame_id, num_right_attempts, num_wrong_attempts) VALUES (?, ?, ?, ?, ?)", data.Username, data.Question_id, data.Minigame_id, data.Num_Right_Attempts, data.Num_Wrong_Attempts)
	if err != nil {
		return err
	}

	return nil
}

func GetClassStatistics(classroomID int) ([]types.ClassStatistics, error) {
	// get classroomID
	// classroomID, _ := strconv.Atoi(classroomIDStr)

	var statistics []types.ClassStatistics

	// get scores and count per score
	rows, err := db.Query("SELECT score, COUNT(*) AS count_per_score FROM statistics WHERE classroom_id = ? GROUP BY score ORDER BY score", classroomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var statistic types.ClassStatistics
		if err := rows.Scan(&statistic.Score, &statistic.Count); err != nil {
			return nil, err
		}
		statistics = append(statistics, statistic)
	}
	fmt.Print("returned class statistics contains: ", statistics)

	return statistics, nil
}
