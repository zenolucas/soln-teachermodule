package database

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"soln-teachermodule/types"

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
