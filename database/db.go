package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"soln-teachermodule/types"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

var db *sql.DB

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
	// godotenv.Load failing (typically: no .env file) is fine - .env is a dev
	// convenience, not a requirement. Docker, systemd, and most PaaS environments set
	// these variables directly and never have a .env at all. What must exist is the
	// variables themselves, checked explicitly below.
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("loading .env: %w", err)
	}

	dbUser := os.Getenv("DBUSER")
	dbPass := os.Getenv("DBPASS")
	dbName := os.Getenv("DBNAME")
	var missing []string
	if dbUser == "" {
		missing = append(missing, "DBUSER")
	}
	if dbPass == "" {
		missing = append(missing, "DBPASS")
	}
	if dbName == "" {
		missing = append(missing, "DBNAME")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variable(s): %s", strings.Join(missing, ", "))
	}

	dbHost := os.Getenv("DBHOST")
	if dbHost == "" {
		dbHost = "127.0.0.1"
	}
	dbPort := os.Getenv("DBPORT")
	if dbPort == "" {
		dbPort = "3306"
	}

	// Capture connection properties.
	cfg := mysql.Config{
		User:                 dbUser,
		Passwd:               dbPass,
		Net:                  "tcp",
		Addr:                 dbHost + ":" + dbPort,
		DBName:               dbName,
		AllowNativePasswords: true,
	}

	// Get a database handle.
	var err error
	db, err = sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return fmt.Errorf("opening database connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		return fmt.Errorf("pinging database: %w", err)
	}
	fmt.Println("Database connection established.")

	// Go's default is unlimited open connections, so a classroom of 30 students
	// hitting /game/* at once could exhaust MySQL's max_connections; idle
	// connections that outlive MySQL's wait_timeout also produce intermittent
	// "invalid connection" errors without a lifetime cap.
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	return err
}

func AuthenticateWebUser(ctx context.Context, username string, password string) error {
	var storedHash string
	row := db.QueryRowContext(ctx, "SELECT password FROM users WHERE username = ? AND usertype = ?", username, "teacher")
	if err := row.Scan(&storedHash); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("authentication Error: incorrect username or password")
		}
		return fmt.Errorf("database Error: %v", err)
	}

	// bcrypt.CompareHashAndPassword runs in constant time regardless of where the
	// mismatch occurs, unlike the plain string comparison this replaced.
	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password)); err != nil {
		return fmt.Errorf("authentication Error: incorrect username or password")
	}
	// Authentication successful
	return nil
}

func AuthenticateGameUser(ctx context.Context, username string, password string) bool {
	var storedHash string
	row := db.QueryRowContext(ctx, "SELECT password FROM users WHERE username = ? AND usertype = ?", username, "student")
	if err := row.Scan(&storedHash); err != nil {
		return false
	}

	return bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password)) == nil
}

// gets classroomID of a student
func GetClassroomID(ctx context.Context, username string) (int, error) {
	var classroomID int

	// Resolve via the enrollments table (the actual source of truth, and the one the
	// teacher UI writes to) instead of matching on users.section - section is a free-text
	// field with no uniqueness constraint, so two teachers using the same section string
	// could silently cross-wire a student into the wrong teacher's classroom.
	// A student enrolled in more than one classroom returns the first match; the game's
	// login API only has room for a single classroom_id today, so picking one classroom
	// is the smallest fix that removes the section-string bug without redesigning that
	// wire contract.
	err := db.QueryRowContext(ctx, `
		SELECT e.classroom_id FROM enrollments e
		JOIN users u ON u.user_id = e.student_id
		WHERE u.username = ? AND u.usertype = 'student'
		LIMIT 1
	`, username).Scan(&classroomID)
	if err != nil {
		return 0, err
	}

	return classroomID, nil
}

func GetStudentID(ctx context.Context, username string) (int, error) {
	var studentID int

	// first get section of student
	err := db.QueryRowContext(ctx, "SELECT user_id FROM users WHERE username = ? AND usertype = ?", username, "student").Scan(&studentID)
	if err != nil {
		return 0, err
	}

	return studentID, nil
}

// GetTeacher reads a teacher's display identity. firstname is scanned via
// sql.NullString because it's NULL for every teacher registered through the current
// Register form, which only asks for a username (see 02 "Facts about the current data
// model").
func GetTeacher(ctx context.Context, teacherID int) (types.Teacher, error) {
	teacher := types.Teacher{UserID: teacherID}
	var firstname sql.NullString
	err := db.QueryRowContext(ctx, "SELECT username, firstname FROM users WHERE user_id = ?", teacherID).
		Scan(&teacher.Username, &firstname)
	if err != nil {
		return types.Teacher{}, err
	}
	teacher.Firstname = firstname.String
	return teacher, nil
}

func GetStudent(ctx context.Context, userID int) (types.Student, error) {
	var student types.Student

	err := db.QueryRowContext(ctx, "SELECT firstname, lastname FROM users WHERE user_id = ?", userID).Scan(&student.Firstname, &student.Lastname)
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

	hash, err := bcrypt.GenerateFromPassword([]byte(userCreds.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(r.Context(), "INSERT INTO users (username, password, usertype) VALUES (?, ?, ?)", userCreds.Username, string(hash), "teacher")
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

	hash, err := bcrypt.GenerateFromPassword([]byte(data.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// No save_states row here: a student's first save creates it, and until then loading returns
	// DefaultSave (SAVE-01). That also removes the second, non-transactional write (BUG-14).
	_, err = db.ExecContext(r.Context(), "INSERT INTO users (username, usertype, firstname, lastname, section, class_number, password) VALUES (?, ?, ?, ?, ?, ?, ?)", data.Username, "student", data.FirstName, data.Lastname, data.Section, data.ClassNumber, string(hash))
	return err
}

func GetStudents(ctx context.Context, classroomID int) ([]types.Student, error) {
	var students []types.Student
	// get students given classroomID
	rows, err := db.QueryContext(ctx, "SELECT users.firstname, users.lastname, users.user_id FROM enrollments e JOIN users ON e.student_id = users.user_id WHERE e.classroom_id = ? ", classroomID)
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

// likeEscaper escapes % and _ - the two characters that are wildcards inside a LIKE
// pattern - so a search term containing either matches literally instead of quietly
// widening the search.
var likeEscaper = strings.NewReplacer("%", `\%`, "_", `\_`)

// GetUnenrolledStudents lists students not already enrolled in classroomID (02 §C6),
// optionally narrowed by q (matched against first or last name). OtherClass is
// populated via a LEFT JOIN through enrollments to classrooms, picking one class name
// (MIN, so the result is deterministic) if the student is already enrolled elsewhere -
// under DEC-25's one-classroom-per-student rule that's at most one, but MIN also keeps
// this safe if that's ever violated in existing data (see X14: the DB doesn't enforce
// it yet).
func GetUnenrolledStudents(ctx context.Context, classroomID int, q string) ([]types.Student, error) {
	query := `
		SELECT u.user_id, u.firstname, u.lastname, MIN(c2.classroom_name)
		FROM users u
		LEFT JOIN enrollments e2 ON e2.student_id = u.user_id
		LEFT JOIN classrooms c2 ON c2.classroom_id = e2.classroom_id
		WHERE u.usertype = 'student'
		  AND u.user_id NOT IN (SELECT student_id FROM enrollments WHERE classroom_id = ?)`
	args := []any{classroomID}

	if q != "" {
		like := "%" + likeEscaper.Replace(q) + "%"
		query += ` AND (u.firstname LIKE ? OR u.lastname LIKE ?)`
		args = append(args, like, like)
	}
	query += ` GROUP BY u.user_id, u.firstname, u.lastname`

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var students []types.Student
	for rows.Next() {
		var student types.Student
		var otherClass sql.NullString
		if err := rows.Scan(&student.UserID, &student.Firstname, &student.Lastname, &otherClass); err != nil {
			return nil, fmt.Errorf("GetUnenrolledStudents: %v", err)
		}
		student.OtherClass = otherClass.String
		students = append(students, student)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetUnenrolledStudents: %v", err)
	}

	return students, nil
}

// AddStudents enrolls each student, silently skipping one already enrolled anywhere -
// under DEC-25's one-classroom-per-student rule (V4), the UI already disables an
// already-enrolled student's checkbox (see HandleGetUnenrolledStudents), so reaching
// this is only possible via a hand-crafted request. The check and insert happen in one
// statement (WHERE NOT EXISTS) rather than a separate SELECT then INSERT, so a second
// request racing the same student in between can't both pass the check and enroll them
// twice.
func AddStudents(ctx context.Context, studentIDs []string, classroomID int) error {
	for _, studentID := range studentIDs {
		_, err := db.ExecContext(ctx,
			"INSERT INTO enrollments (classroom_id, student_id) SELECT ?, ? FROM DUAL WHERE NOT EXISTS (SELECT 1 FROM enrollments WHERE student_id = ?)",
			classroomID, studentID, studentID)
		if err != nil {
			return err
		}
	}

	return nil
}

func UnenrollStudent(ctx context.Context, studentID int, classroomID int) error {
	// Execute the DELETE query
	_, err := db.ExecContext(ctx, "DELETE FROM enrollments WHERE student_id = ? AND classroom_id = ?", studentID, classroomID)
	if err != nil {
		return err
	}
	return nil
}

func InsertClassroom(ctx context.Context, classroom types.Classroom, teacherID int) error {
	_, err := db.ExecContext(ctx, "INSERT INTO classrooms (classroom_name, section, description, teacher_ID) VALUES (?, ?, ?, ?)", classroom.ClassroomName, classroom.Section, classroom.Description, teacherID)
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
	err := db.QueryRowContext(r.Context(), "SELECT user_id FROM users WHERE username = ? AND usertype = ?", userCreds.Username, "teacher").Scan(&teacherID)

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

// GetClassroomTeacherID returns the teacher_id that owns a classroom, so a caller can
// verify the authenticated teacher actually owns the classroom they're about to act on
// (see SEC-06 - without this, any authenticated teacher can view or modify any other
// teacher's classroom just by changing a URL/form parameter).
func GetClassroomTeacherID(ctx context.Context, classroomID int) (int, error) {
	var teacherID int
	err := db.QueryRowContext(ctx, "SELECT teacher_id FROM classrooms WHERE classroom_id = ?", classroomID).Scan(&teacherID)
	if err != nil {
		return 0, err
	}
	return teacherID, nil
}

// IsEnrolled reports whether a student is currently enrolled in a classroom (X6, T5.6):
// the student statistics handlers checked that the teacher owns classroomID, but never
// that userID is actually one of that classroom's students, so any teacher could read
// any student's name and answers by posting a foreign userID alongside their own
// classroomID.
func IsEnrolled(ctx context.Context, studentID, classroomID int) (bool, error) {
	var exists bool
	err := db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM enrollments WHERE student_id = ? AND classroom_id = ?)", studentID, classroomID).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// GetClassroom returns a single classroom's name/section/description, so the
// classroom page can show which classroom the teacher is actually looking at (see
// FE-12 - previously the page showed nothing but tabs, with the name visible only in
// the sidebar list). Callers must check ownership themselves (assertOwnsClassroom);
// this has no teacherID filter, matching GetClassroomTeacherID above.
func GetClassroom(ctx context.Context, classroomID int) (types.Classroom, error) {
	var classroom types.Classroom
	classroom.ClassroomID = strconv.Itoa(classroomID)
	err := db.QueryRowContext(ctx, "SELECT classroom_name, section, description FROM classrooms WHERE classroom_id = ?", classroomID).
		Scan(&classroom.ClassroomName, &classroom.Section, &classroom.Description)
	if err != nil {
		return types.Classroom{}, err
	}
	return classroom, nil
}

func GetClassrooms(ctx context.Context, teacherID int) ([]types.Classroom, error) {
	var classrooms []types.Classroom

	rows, err := db.QueryContext(ctx, "SELECT classroom_id, classroom_name, section, description FROM classrooms WHERE teacher_id = ?", teacherID)
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

func GetFractionQuestions(ctx context.Context, minigame_id int, classroom_id int) ([]types.FractionQuestion, error) {
	var fractions []types.FractionQuestion

	rows, err := db.QueryContext(ctx, "SELECT question_id, fraction1_numerator, fraction1_denominator, fraction2_numerator, fraction2_denominator FROM fraction_questions WHERE minigame_id = ? AND classroom_id = ?", minigame_id, classroom_id)
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
	// Every strconv.Atoi error used to be discarded, so a blank field (a teacher
	// submitting the Add Question form with an empty input) silently inserted as 0 -
	// including a 0 denominator, which then gets served to a student's game as an
	// unsolvable fraction (see FE-23). formInt fails loudly instead, matching the
	// pattern UpdateFractions below already uses.
	MinigameID, err := formInt(r, "minigameID")
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
	if Fraction1_Denominator == 0 || Fraction2_Denominator == 0 {
		return errors.New("fraction denominator must not be 0")
	}

	_, err = db.ExecContext(r.Context(), "INSERT INTO fraction_questions (fraction1_numerator, fraction1_denominator, fraction2_numerator, fraction2_denominator, minigame_id, classroom_id) VALUES (?, ?, ?, ?, ?, ?)",
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
	ClassroomID, err := formInt(r, "classroom_id")
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

	// A teacher's session is only checked against the classroom_id they claim
	// (handler.assertOwnsClassroom), so a foreign question_id posted alongside their
	// own classroom_id would otherwise update someone else's question (see X5/SEC-06).
	// RowsAffected can't stand in for this check: MySQL/MariaDB reports 0 rows affected
	// when an UPDATE's new values match the existing row, not just when no row matched.
	var count int
	if err := db.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM fraction_questions WHERE question_id = ? AND minigame_id = ? AND classroom_id = ?", QuestionID, MinigameID, ClassroomID).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("UpdateFractions: no question found with question_id=%d minigame_id=%d classroom_id=%d", QuestionID, MinigameID, ClassroomID)
	}

	_, err = db.ExecContext(r.Context(), "UPDATE fraction_questions SET fraction1_numerator = ?,  fraction1_denominator = ?, fraction2_numerator = ?, fraction2_denominator = ? WHERE minigame_id = ? AND question_id = ? AND classroom_id = ?",
		Fraction1_Numerator, Fraction1_Denominator, Fraction2_Numerator, Fraction2_Denominator, MinigameID, QuestionID, ClassroomID)
	if err != nil {
		return err
	}

	return nil
}

func DeleteFractions(ctx context.Context, minigameID string, questionID string, classroomID string) error {
	// fraction_responses.question_id is a foreign key with the default RESTRICT, so the
	// parent question can't be deleted while responses still reference it (see INFRA-04).
	// Delete children first, in one transaction - same pattern as DeleteMCQuestions.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "DELETE FROM fraction_responses WHERE question_id = ?", questionID); err != nil {
		return err
	}

	// classroom_id must be in the WHERE clause, not just checked by the caller against
	// the session - otherwise a teacher who owns classroomID but supplies a questionID
	// that actually belongs to a different classroom would delete someone else's
	// question (see SEC-06).
	result, err := tx.ExecContext(ctx, "DELETE FROM fraction_questions WHERE minigame_id = ? AND question_id = ? AND classroom_id = ?", minigameID, questionID, classroomID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("DeleteFractions: no question found with minigame_id=%s question_id=%s classroom_id=%s", minigameID, questionID, classroomID)
	}

	return tx.Commit()
}

func GetWordedQuestions(ctx context.Context, minigame_id int, classroom_id int) ([]types.FractionQuestion, error) {
	var questions []types.FractionQuestion

	// get questiontext and correct answer
	rows, err := db.QueryContext(ctx, "SELECT question_id, question_text, fraction1_numerator, fraction1_denominator, fraction2_numerator, fraction2_denominator FROM fraction_questions WHERE minigame_id = ? AND classroom_id = ?", minigame_id, classroom_id)
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
	questionText := r.FormValue("question_text")
	if strings.TrimSpace(questionText) == "" {
		return errors.New("question text must not be blank")
	}

	// See AddFractionQuestions above for why this uses formInt instead of a bare
	// strconv.Atoi with the error discarded (FE-23).
	minigameID, err := formInt(r, "minigameID")
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
	if fraction1Denominator == 0 || fraction2Denominator == 0 {
		return errors.New("fraction denominator must not be 0")
	}

	_, err = db.ExecContext(r.Context(), "INSERT INTO fraction_questions (question_text, fraction1_numerator, fraction1_denominator, fraction2_numerator, fraction2_denominator, minigame_id, classroom_id) VALUES (?, ?, ?, ?, ?, ?, ?)",
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

	_, err = db.ExecContext(r.Context(), "UPDATE fraction_questions SET question_text = ?, fraction1_numerator = ?,  fraction1_denominator = ?, fraction2_numerator = ?, fraction2_denominator = ? WHERE minigame_id = ? AND question_id = ? AND classroom_id = ?",
		questionText, fraction1Numerator, fraction1Denominator, fraction2Numerator, fraction2Denominator, minigameID, questionID, classroomID)
	if err != nil {
		return err
	}

	return nil
}

func DeleteWorded(ctx context.Context, minigameID int, questionID int, classroomID int) error {
	// fraction_responses.question_id is a foreign key with the default RESTRICT, so the
	// parent question can't be deleted while responses still reference it (see INFRA-04).
	// Delete children first, in one transaction - same pattern as DeleteMCQuestions.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "DELETE FROM fraction_responses WHERE question_id = ?", questionID); err != nil {
		return err
	}

	// classroom_id must be in the WHERE clause, not just checked by the caller against
	// the session - otherwise a teacher who owns classroomID but supplies a questionID
	// that actually belongs to a different classroom would delete someone else's
	// question (see SEC-06).
	result, err := tx.ExecContext(ctx, "DELETE FROM fraction_questions WHERE minigame_id = ? AND question_id = ? AND classroom_id = ?", minigameID, questionID, classroomID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("DeleteWorded: no question found with minigame_id=%d question_id=%d classroom_id=%d", minigameID, questionID, classroomID)
	}

	return tx.Commit()
}

// GetQuizQuestions used to run one query for the questions, then one more query per
// question for its choices (11 round trips for 10 questions), with a defer inside the
// per-question loop that kept every one of those choice result sets open until the
// whole function returned. A single LEFT JOIN, grouped in Go by question_id, replaces
// all of it with one round trip.
func GetQuizQuestions(ctx context.Context, minigame_id int, classroom_id int) ([]types.MultipleChoiceQuestion, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT q.question_id, q.question_text, c.choice_id, c.choice_text, c.is_correct
		FROM multiple_choice_questions q
		LEFT JOIN multiple_choice_choices c ON c.question_id = q.question_id
		WHERE q.minigame_id = ? AND q.classroom_id = ?
		ORDER BY q.question_id, c.choice_id
	`, minigame_id, classroom_id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var questions []types.MultipleChoiceQuestion
	// Maps question_id to its index in questions, so the repeated rows the LEFT
	// JOIN produces (one per choice) append to the right question instead of each
	// creating a new entry.
	index := make(map[int]int)

	for rows.Next() {
		var questionID int
		var questionText string
		var choiceID sql.NullInt64
		var choiceText sql.NullString
		var isCorrect sql.NullBool

		if err := rows.Scan(&questionID, &questionText, &choiceID, &choiceText, &isCorrect); err != nil {
			return nil, err
		}

		i, ok := index[questionID]
		if !ok {
			questions = append(questions, types.MultipleChoiceQuestion{
				QuestionID:   questionID,
				QuestionText: questionText,
			})
			i = len(questions) - 1
			index[questionID] = i
		}

		// choice_id is NULL only when a question has zero choices (the LEFT JOIN
		// finds no match) - skip appending rather than adding a zero-valued choice.
		if choiceID.Valid {
			questions[i].Choices = append(questions[i].Choices, types.Choice{
				ChoiceID:   int(choiceID.Int64),
				ChoiceText: choiceText.String,
				IsCorrect:  isCorrect.Bool,
			})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return questions, nil
}

func AddMCQuestions(w http.ResponseWriter, r *http.Request, classroomID int) error {
	// See AddFractionQuestions above for why this uses formInt instead of a bare
	// strconv.Atoi with the error discarded (FE-23).
	minigameID, err := formInt(r, "minigameID")
	if err != nil {
		return err
	}

	questionText := r.FormValue("question_text")
	if strings.TrimSpace(questionText) == "" {
		return errors.New("question text must not be blank")
	}

	// The "Add Question" form's correct-answer <select> submits the literal option
	// key ("option_1".."option_4"), not the option's text - the form fields share
	// that same naming, so we can match the correct answer by position rather than
	// by comparing choice text (which breaks if two options have identical text).
	optionKeys := []string{"option_1", "option_2", "option_3", "option_4"}
	correctAnswer := r.FormValue("correct_answer")
	for _, key := range optionKeys {
		if strings.TrimSpace(r.FormValue(key)) == "" {
			return fmt.Errorf("%s must not be blank", key)
		}
	}

	// first insert question_text without the correct_answer id
	result, err := db.ExecContext(r.Context(), `INSERT INTO multiple_choice_questions (classroom_id, minigame_id, question_text) VALUES (?, ?, ?)`, classroomID, minigameID, questionText)
	if err != nil {
		return err
	}

	// Get the last inserted question_id
	questionID, _ := result.LastInsertId()

	// Insert choices into the multiple_choice_choices table using the questionID
	for _, key := range optionKeys {
		choiceText := r.FormValue(key)
		isCorrect := key == correctAnswer
		if _, err := db.ExecContext(r.Context(), "INSERT INTO multiple_choice_choices (question_id, choice_text, is_correct) VALUES (?, ?, ?)", questionID, choiceText, isCorrect); err != nil {
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
	choices, err := ConstructChoices(r, correctAnswerID)
	if err != nil {
		return err
	}

	questionID, err := formInt(r, "questionID")
	if err != nil {
		return err
	}
	classroomID, err := formInt(r, "classroomID")
	if err != nil {
		return err
	}

	// A teacher's session is only checked against the classroomID they claim
	// (handler.assertOwnsClassroom), so a foreign questionID posted alongside their own
	// classroomID would otherwise update someone else's question and choices (see
	// X5/SEC-06). Everything below runs in one transaction, following DeleteMCQuestions'
	// pattern, so a classroom mismatch leaves nothing partially applied.
	tx, err := db.BeginTx(r.Context(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var count int
	if err := tx.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM multiple_choice_questions WHERE question_id = ? AND classroom_id = ?", questionID, classroomID).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("UpdateMCQuestions: no question found with question_id=%d classroom_id=%d", questionID, classroomID)
	}

	if _, err = tx.ExecContext(r.Context(), "UPDATE multiple_choice_questions SET question_text = ? WHERE question_id = ? AND classroom_id = ?",
		question.QuestionText, questionID, classroomID); err != nil {
		return err
	}

	// given the choices[]
	// loop through, each choice gets to execute an update
	for _, choice := range choices {
		if _, err = tx.ExecContext(r.Context(), "UPDATE multiple_choice_choices SET choice_text = ?, is_correct = ? WHERE choice_id = ? AND question_id = ?",
			choice.ChoiceText, choice.IsCorrect, choice.ChoiceID, questionID); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// helper func to construct choices[]
// ConstructChoices is exported so handler.HandleUpdateMCQuestions can reuse the exact
// same option/choiceID/correct-answer parsing to rebuild the just-saved question's
// choices for its response, instead of duplicating this logic or doing an extra
// round-trip query (see FE-22).
func ConstructChoices(r *http.Request, correctAnswerID int) ([]types.Choice, error) {
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

func DeleteMCQuestions(ctx context.Context, minigameID int, questionID int, classroomID int) error {
	// multiple_choice_choices.question_id and multiple_choice_responses.question_id/choice_id
	// are foreign keys with the default RESTRICT, so the parent question can't be deleted
	// while either still references it. Delete children first, in one transaction.
	//
	// classroom_id must be in the final DELETE's WHERE clause, not just checked by the
	// caller against the session - otherwise a teacher who owns classroomID but supplies
	// a questionID that actually belongs to a different classroom would delete someone
	// else's question (see SEC-06). Checking RowsAffected on that delete and rolling
	// back (via the deferred tx.Rollback, since we return before tx.Commit) if it
	// affected nothing means a classroom mismatch also undoes the child deletes above,
	// rather than leaving them applied against a question that didn't end up deleted.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "DELETE FROM multiple_choice_responses WHERE question_id = ?", questionID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM multiple_choice_choices WHERE question_id = ?", questionID); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, "DELETE FROM multiple_choice_questions WHERE minigame_id = ? AND question_id = ? AND classroom_id = ?", minigameID, questionID, classroomID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("DeleteMCQuestions: no question found with minigame_id=%d question_id=%d classroom_id=%d", minigameID, questionID, classroomID)
	}

	return tx.Commit()
}

func AddQuizStatistics(ctx context.Context, classroomID int, minigameID int, student_id, score int) error {
	_, err := db.ExecContext(ctx, "INSERT INTO multiple_choice_scores (classroom_id, minigame_id, student_id, score) VALUES (?, ?, ?, ?)", classroomID, minigameID, student_id, score)
	if err != nil {
		return err
	}

	return nil
}

// CountQuizQuestions returns how many questions a quiz minigame has, so a posted score
// can be sanity-checked against it - a score higher than the question count can't be
// legitimate (see SEC-04).
func CountQuizQuestions(ctx context.Context, minigameID int, classroomID int) (int, error) {
	var count int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM multiple_choice_questions WHERE minigame_id = ? AND classroom_id = ?", minigameID, classroomID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// adds statistics for minigames 1 and 2 (simple fraction gameplay), also for substraction simple fraction gameplay
// studentID comes from the caller, which resolves it from the game token issued at
// /game/login rather than trusting one posted in the request body - a body-supplied
// student_id would let any client record fraction-question attempts as any student.
func AddFractionStatistics(w http.ResponseWriter, r *http.Request, studentID int) error {
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

	_, err = db.ExecContext(r.Context(), "INSERT INTO fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts) VALUES (?, ?, ?, ?, ?, ?)", data.ClassroomID, data.MinigameID, data.QuestionID, studentID, data.Num_Right_Attempts, data.Num_Wrong_Attempts)
	if err != nil {
		return err
	}

	return nil
}


func AddQuizResponse(ctx context.Context, classroomID int, minigameID int, questionID int, studentID int, choiceID int) error {
	_, err := db.ExecContext(ctx, "INSERT INTO multiple_choice_responses (classroom_id, minigame_id, question_id, student_id, choice_id) VALUES (?, ?, ?, ?, ?)", classroomID, minigameID, questionID, studentID, choiceID)
	if err != nil {
		return err
	}
	return nil
}

// GetStudentFractionStatistics is one student's per-question performance in a
// fraction minigame. Aggregated (SUM right, SUM wrong, MAX wrong per question, fixing
// X7): a plain LEFT JOIN without GROUP BY produced one row per fraction_responses
// row, so a student who replayed a question showed up as duplicate rows here instead
// of one combined one.
func GetStudentFractionStatistics(ctx context.Context, userID int, minigameID int, classroomID int) ([]types.StudentFractionStatistics, error) {
	var statistics []types.StudentFractionStatistics

	rows, err := db.QueryContext(ctx, `
		SELECT fq.question_id, fq.fraction1_numerator, fq.fraction1_denominator, fq.fraction2_numerator, fq.fraction2_denominator,
		       COALESCE(SUM(fr.num_right_attempts), 0), COALESCE(SUM(fr.num_wrong_attempts), 0), COALESCE(MAX(fr.num_wrong_attempts), 0)
		FROM fraction_questions fq
		LEFT JOIN fraction_responses fr ON fq.question_id = fr.question_id AND fr.student_id = ? AND fr.minigame_id = ?
		WHERE fq.minigame_id = ? AND fq.classroom_id = ?
		GROUP BY fq.question_id, fq.fraction1_numerator, fq.fraction1_denominator, fq.fraction2_numerator, fq.fraction2_denominator
	`, userID, minigameID, minigameID, classroomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var statistic types.StudentFractionStatistics
		if err := rows.Scan(&statistic.QuestionID, &statistic.Fraction1_Numerator, &statistic.Fraction1_Denominator, &statistic.Fraction2_Numerator, &statistic.Fraction2_Denominator, &statistic.RightAttemptsCount, &statistic.WrongAttemptsCount, &statistic.MaxWrongAttemptsCount); err != nil {
			return nil, fmt.Errorf("GetStudentFractionStatistics: %v", err)
		}
		statistics = append(statistics, statistic)
	}

	return statistics, nil
}

// GetStudentWordedStatistics is GetStudentFractionStatistics' worded-question
// equivalent - same fix (X7: aggregated, not one row per replay).
func GetStudentWordedStatistics(ctx context.Context, userID int, minigameID int, classroomID int) ([]types.StudentFractionStatistics, error) {
	var statistics []types.StudentFractionStatistics

	rows, err := db.QueryContext(ctx, `
		SELECT fq.question_id, fq.question_text,
		       COALESCE(SUM(fr.num_right_attempts), 0), COALESCE(SUM(fr.num_wrong_attempts), 0), COALESCE(MAX(fr.num_wrong_attempts), 0)
		FROM fraction_questions fq
		LEFT JOIN fraction_responses fr ON fq.question_id = fr.question_id AND fr.student_id = ? AND fr.minigame_id = ?
		WHERE fq.minigame_id = ? AND fq.classroom_id = ?
		GROUP BY fq.question_id, fq.question_text
	`, userID, minigameID, minigameID, classroomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var statistic types.StudentFractionStatistics
		if err := rows.Scan(&statistic.QuestionID, &statistic.QuestionText, &statistic.RightAttemptsCount, &statistic.WrongAttemptsCount, &statistic.MaxWrongAttemptsCount); err != nil {
			return nil, fmt.Errorf("GetStudentWordedStatistics: %v", err)
		}
		statistics = append(statistics, statistic)
	}

	return statistics, nil
}

// GetFractionQuestionSummaries returns one row per question with right/wrong attempt
// counts summed across every student in the classroom - the class-wide equivalent of
// GetStudentFractionStatistics's single-student LEFT JOIN, reusing the same
// types.StudentFractionStatistics shape since a class summary is a per-question
// right/wrong count either way. Backs the sortable summary table that replaced a
// one-chart-per-question stack (see FE-30, D6).
func GetFractionQuestionSummaries(ctx context.Context, classroomID int, minigameID int) ([]types.StudentFractionStatistics, error) {
	var summaries []types.StudentFractionStatistics

	rows, err := db.QueryContext(ctx, "SELECT fq.question_id, fq.fraction1_numerator, fq.fraction1_denominator, fq.fraction2_numerator, fq.fraction2_denominator, COALESCE(SUM(fr.num_right_attempts), 0), COALESCE(SUM(fr.num_wrong_attempts), 0) FROM fraction_questions fq LEFT JOIN fraction_responses fr ON fq.question_id = fr.question_id AND fr.classroom_id = fq.classroom_id AND fr.minigame_id = fq.minigame_id WHERE fq.minigame_id = ? AND fq.classroom_id = ? GROUP BY fq.question_id, fq.fraction1_numerator, fq.fraction1_denominator, fq.fraction2_numerator, fq.fraction2_denominator", minigameID, classroomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var summary types.StudentFractionStatistics
		if err := rows.Scan(&summary.QuestionID, &summary.Fraction1_Numerator, &summary.Fraction1_Denominator, &summary.Fraction2_Numerator, &summary.Fraction2_Denominator, &summary.RightAttemptsCount, &summary.WrongAttemptsCount); err != nil {
			return nil, fmt.Errorf("GetFractionQuestionSummaries: %v", err)
		}
		summaries = append(summaries, summary)
	}

	return summaries, nil
}

// GetWordedQuestionSummaries is GetFractionQuestionSummaries' worded-question
// equivalent (see FE-30, D6) - same class-wide aggregation, question_text instead of
// the fraction fields as the displayed question. It still selects the fraction1/2
// fields too (added for T5.4): a worded question is still backed by the same two
// fractions and operator as a fraction question - question_text is just the word
// problem wrapped around them - and the statistics page's "Answer" column
// (util.Combine) needs them for worded scenes exactly as it does for fraction ones.
func GetWordedQuestionSummaries(ctx context.Context, classroomID int, minigameID int) ([]types.StudentFractionStatistics, error) {
	var summaries []types.StudentFractionStatistics

	rows, err := db.QueryContext(ctx, "SELECT fq.question_id, fq.question_text, fq.fraction1_numerator, fq.fraction1_denominator, fq.fraction2_numerator, fq.fraction2_denominator, COALESCE(SUM(fr.num_right_attempts), 0), COALESCE(SUM(fr.num_wrong_attempts), 0) FROM fraction_questions fq LEFT JOIN fraction_responses fr ON fq.question_id = fr.question_id AND fr.classroom_id = fq.classroom_id AND fr.minigame_id = fq.minigame_id WHERE fq.minigame_id = ? AND fq.classroom_id = ? GROUP BY fq.question_id, fq.question_text, fq.fraction1_numerator, fq.fraction1_denominator, fq.fraction2_numerator, fq.fraction2_denominator", minigameID, classroomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var summary types.StudentFractionStatistics
		if err := rows.Scan(&summary.QuestionID, &summary.QuestionText, &summary.Fraction1_Numerator, &summary.Fraction1_Denominator, &summary.Fraction2_Numerator, &summary.Fraction2_Denominator, &summary.RightAttemptsCount, &summary.WrongAttemptsCount); err != nil {
			return nil, fmt.Errorf("GetWordedQuestionSummaries: %v", err)
		}
		summaries = append(summaries, summary)
	}

	return summaries, nil
}

func GetStudentQuizStatistics(ctx context.Context, userID int, minigameID int, classroomID int) ([]types.StudentQuizStatistics, error) {
	var statistics []types.StudentQuizStatistics

	// One query replaces what used to be three (questions, then correct answers via a
	// dynamically-built IN (...), then the student's answers) plus a manual join in Go.
	// The IN (...) version broke with a SQL syntax error whenever a minigame had zero
	// questions (an empty placeholder list builds "IN ()"); grouping by question here
	// sidesteps that failure mode entirely instead of needing a special case for it.
	// COALESCE covers a question with no choice marked correct, and a question the
	// student hasn't answered - both would otherwise scan as SQL NULL into a Go string.
	rows, err := db.QueryContext(ctx, `
		SELECT
			q.question_id,
			q.question_text,
			COALESCE(MAX(CASE WHEN c.is_correct THEN c.choice_text END), '') AS correct_answer,
			COALESCE(MAX(CASE WHEN r.choice_id = c.choice_id THEN c.choice_text END), '') AS user_answer
		FROM multiple_choice_questions q
		JOIN multiple_choice_choices c ON c.question_id = q.question_id
		LEFT JOIN multiple_choice_responses r ON r.question_id = q.question_id
			AND r.student_id = ? AND r.minigame_id = ?
		WHERE q.minigame_id = ? AND q.classroom_id = ?
		GROUP BY q.question_id, q.question_text
		ORDER BY q.question_id
	`, userID, minigameID, minigameID, classroomID)
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

