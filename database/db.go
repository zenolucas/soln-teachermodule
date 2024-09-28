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

func GetQuestionDictionary(minigame_id int) ([]types.Question, error) {
	var questions []types.Question

	// get questiontext and correct answer
	rows, err := db.Query("SELECT question_id, question_text, correct_answer FROM questions WHERE minigame_id = ?", minigame_id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var question types.Question
		if err := rows.Scan(&question.QuestionID, &question.QuestionText, &question.CorrectAnswer); err != nil {
			return nil, nil
		}
		questions = append(questions, question)
	}

	for i := 0; i < len(questions); i++ {
		// then we get choices
		rows, err := db.Query("SELECT C1, C2, C3, C4 FROM choices WHERE question_id = ?", questions[i].QuestionID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		type choiceHolder struct {
			C1 string
			C2 string
			C3 string
			C4 string
		}

		var holder choiceHolder
		// then add choices to question in questions slice
		for rows.Next() {
			if err := rows.Scan(&holder.C1, &holder.C2, &holder.C3, &holder.C4); err != nil {
				return nil, err
			}
			questions[i].Choice1 = holder.C1
			questions[i].Choice2 = holder.C2
			questions[i].Choice3 = holder.C3
			questions[i].Choice4 = holder.C4
		}
	}

	fmt.Println(questions)
	return questions, nil
}
