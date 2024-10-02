package database

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"soln-teachermodule/types"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

var db *sql.DB

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
func AuthenticateSolnUser(username string, password string) bool {

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
		log.Fatal(err)
	}
	return err
}

func GetStudents() ([]types.Student, error) {
	var students []types.Student

	rows, err := db.Query("SELECT username FROM users WHERE usertype = 'student'")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var student types.Student
		if err := rows.Scan(&student.Username); err != nil {
			return nil, fmt.Errorf("GetStudents: %v", err)
		}
		students = append(students, student)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetUsers: %v", err)
	}

	return students, nil
}

func InsertClassroom(w http.ResponseWriter, r *http.Request) error {
	userCreds := types.UserCredentials{
		Username: r.FormValue("username"),
		Password: r.FormValue("password"),
	}

	// then get teacherID from request.
	user := r.Context().Value(types.UserContextKey).(types.AuthenticatedUser)

	_, err := db.Exec("INSERT INTO classrooms (classroomname, section, teacherID) VALUES (?, ?, ?)", userCreds.Username, userCreds.Password, user.UserID)
	if err != nil {
		log.Fatal(err)
	}
	return err
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

// Example function to save session token in the database
func SaveSessionToken(userID int, sessionToken string) error {
	query := `INSERT INTO sessions (user_id, session_token, expires_at) VALUES (?, ?, ?)`
	_, err := db.Exec(query, userID, sessionToken, time.Now().Add(24*time.Hour))
	return err
}

func GetClassrooms(w http.ResponseWriter, r http.Request) error {
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

	fmt.Println(questions)
	return questions, nil
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

	// get questionID

	_, err := db.Exec("UPDATE multiple_choice_questions SET question_text = ?,  correct_answer = ? WHERE question_id = 1",
		question.QuestionText, question.CorrectAnswer)
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
