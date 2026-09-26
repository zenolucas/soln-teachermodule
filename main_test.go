package main

// Handler-level integration tests (CLEANUP-02): the real router over HTTP against a real, throwaway
// MariaDB database. They run only when SOLN_TEST_DB_ROOT_DSN is set, e.g.
//
//	SOLN_TEST_DB_ROOT_DSN='root@tcp(127.0.0.1:3306)/' go test -run Integration -v .
//
// Each run drops and recreates the database "soln_test" (never the dev database), loads soln_db.sql
// plus testdata/showcase_seed.sql into it, adds the fixtures below, and connects as a dedicated "soln_test"
// user. The tests rely on the showcase data only for teacher / teacher owning classroom 1; everything else
// they need comes from testFixtures, so the showcase data can change freely.

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"soln-teachermodule/database"
	"soln-teachermodule/handler"

	"github.com/go-sql-driver/mysql"
)

const (
	testDBName = "soln_test"
	testDBUser = "soln_test"
	testDBPass = "soln_test_pw"
)

// testFixtures adds what the tests need on top of the showcase data: another teacher's classroom (for
// ownership checks) and a student with password "pw" and a save, enrolled in classroom 1.
const testFixtures = `
INSERT INTO users (username, usertype, password) VALUES
('it_other_teacher', 'teacher', '$2a$10$9JvwzEQIZIg1MhN9Q3E8TeIKtLCjHU/6MVcIApnaSWxxGPP8QuOua');
INSERT INTO classrooms (classroom_name, section, description, teacher_id)
SELECT 'IT other class', 'x', '', user_id FROM users WHERE username = 'it_other_teacher';
INSERT INTO users (username, firstname, lastname, usertype, section, class_number, password) VALUES
('it_student', 'IT', 'Student', 'student', 'x', '1', '$2a$10$9JvwzEQIZIg1MhN9Q3E8TeIKtLCjHU/6MVcIApnaSWxxGPP8QuOua');
INSERT INTO enrollments (classroom_id, student_id) SELECT 1, user_id FROM users WHERE username = 'it_student';
INSERT INTO save_states (student_id, save_data)
SELECT user_id, '{"current_floor": 1, "current_quest": "share_pie_with_racket", "player_badges": {"shiny_rock": true, "bowl": true}}'
FROM users WHERE username = 'it_student';
`

var (
	testServer *httptest.Server
	testDB     *sql.DB // for assertions, connected as the test user
	skipReason string
)

func TestMain(m *testing.M) {
	rootDSN := os.Getenv("SOLN_TEST_DB_ROOT_DSN")
	if rootDSN == "" {
		skipReason = "SOLN_TEST_DB_ROOT_DSN not set"
		os.Exit(m.Run())
	}
	if err := setupTestDatabase(rootDSN); err != nil {
		fmt.Fprintln(os.Stderr, "integration test setup:", err)
		os.Exit(1)
	}
	code := m.Run()
	testServer.Close()
	os.Exit(code)
}

func setupTestDatabase(rootDSN string) error {
	cfg, err := mysql.ParseDSN(rootDSN)
	if err != nil {
		return fmt.Errorf("parsing SOLN_TEST_DB_ROOT_DSN: %w", err)
	}
	host, port, err := net.SplitHostPort(cfg.Addr)
	if err != nil {
		return fmt.Errorf("SOLN_TEST_DB_ROOT_DSN address %q: %w", cfg.Addr, err)
	}
	cfg.DBName = ""
	cfg.MultiStatements = true
	root, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return err
	}
	defer root.Close()
	if _, err := root.Exec(fmt.Sprintf(`
		DROP DATABASE IF EXISTS %[1]s;
		CREATE DATABASE %[1]s;
		CREATE USER IF NOT EXISTS '%[2]s'@'%%' IDENTIFIED BY '%[3]s';
		GRANT ALL PRIVILEGES ON %[1]s.* TO '%[2]s'@'%%';
		FLUSH PRIVILEGES;`, testDBName, testDBUser, testDBPass)); err != nil {
		return fmt.Errorf("creating %s: %w", testDBName, err)
	}

	cfg.DBName = testDBName
	loader, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return err
	}
	defer loader.Close()
	loader.SetMaxOpenConns(1)
	schema, err := os.ReadFile("soln_db.sql")
	if err != nil {
		return err
	}
	// soln_db.sql targets the dev database by name; point it at the test database instead.
	schemaSQL := strings.NewReplacer(
		"CREATE DATABASE IF NOT EXISTS soln_db;", "",
		"USE soln_db;", "USE "+testDBName+";",
	).Replace(string(schema))
	if _, err := loader.Exec(schemaSQL); err != nil {
		return fmt.Errorf("loading soln_db.sql: %w", err)
	}
	seed, err := os.ReadFile("testdata/showcase_seed.sql")
	if err != nil {
		return err
	}
	if _, err := loader.Exec(string(seed)); err != nil {
		return fmt.Errorf("loading showcase_seed.sql: %w", err)
	}
	if _, err := loader.Exec(testFixtures); err != nil {
		return fmt.Errorf("loading test fixtures: %w", err)
	}

	for key, value := range map[string]string{
		"DBUSER": testDBUser, "DBPASS": testDBPass, "DBNAME": testDBName, "DBHOST": host, "DBPORT": port,
		"SESSION_SECRET":    "integration-test-session-secret-0123456789",
		"GAME_TOKEN_SECRET": "integration-test-game-token-secret-012345",
	} {
		os.Setenv(key, value)
	}
	if err := database.InitializeDatabase(); err != nil {
		return err
	}
	if err := handler.InitSessionStore(); err != nil {
		return err
	}
	if err := handler.InitGameTokenStore(); err != nil {
		return err
	}
	testDB, err = sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s)/%s", testDBUser, testDBPass, cfg.Addr, testDBName))
	if err != nil {
		return err
	}
	testServer = httptest.NewServer(newRouter())
	return nil
}

func requireDB(t *testing.T) {
	t.Helper()
	if skipReason != "" {
		t.Skip(skipReason)
	}
}

// newClient keeps cookies but doesn't follow redirects, so tests can assert on them.
func newClient(t *testing.T) *http.Client {
	t.Helper()
	jar, _ := cookiejar.New(nil)
	return &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

func do(t *testing.T, c *http.Client, method, path string, body io.Reader, header map[string]string) (*http.Response, string) {
	t.Helper()
	req, err := http.NewRequest(method, testServer.URL+path, body)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range header {
		req.Header.Set(k, v)
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp, string(b)
}

func postForm(t *testing.T, c *http.Client, path string, form url.Values, header map[string]string) (*http.Response, string) {
	t.Helper()
	h := map[string]string{"Content-Type": "application/x-www-form-urlencoded"}
	for k, v := range header {
		h[k] = v
	}
	return do(t, c, http.MethodPost, path, strings.NewReader(form.Encode()), h)
}

func postJSON(t *testing.T, c *http.Client, path string, payload any, token string) (*http.Response, string) {
	t.Helper()
	b, _ := json.Marshal(payload)
	h := map[string]string{"Content-Type": "application/json"}
	if token != "" {
		h["Authorization"] = "Bearer " + token
	}
	return do(t, c, http.MethodPost, path, bytes.NewReader(b), h)
}

func teacherClient(t *testing.T) *http.Client {
	t.Helper()
	c := newClient(t)
	resp, _ := postForm(t, c, "/login", url.Values{"username": {"teacher"}, "password": {"teacher"}}, nil)
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("teacher login: status %d, want 303", resp.StatusCode)
	}
	return c
}

func gameLogin(t *testing.T, username string) map[string]any {
	t.Helper()
	_, body := postJSON(t, newClient(t), "/game/login", map[string]string{"username": username, "password": "pw"}, "")
	var out map[string]any
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		t.Fatalf("game login response %q: %v", body, err)
	}
	return out
}

func idOf(t *testing.T, query string, args ...any) int {
	t.Helper()
	return count(t, query, args...)
}

func count(t *testing.T, query string, args ...any) int {
	t.Helper()
	var n int
	if err := testDB.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestIntegrationTeacherAuthAndOwnership(t *testing.T) {
	requireDB(t)
	resp, _ := do(t, newClient(t), http.MethodGet, "/home", nil, nil)
	if resp.StatusCode != http.StatusSeeOther || !strings.HasPrefix(resp.Header.Get("Location"), "/login") {
		t.Errorf("anonymous /home: %d %q, want 303 to /login", resp.StatusCode, resp.Header.Get("Location"))
	}

	c := teacherClient(t)
	other := idOf(t, "SELECT classroom_id FROM classrooms WHERE classroom_name = 'IT other class'")
	for path, want := range map[string]int{
		"/home":                     http.StatusOK,
		"/classroom?classroom_id=1": http.StatusOK,
		fmt.Sprintf("/classroom?classroom_id=%d", other): http.StatusForbidden, // another teacher's classroom
		"/classroom?classroom_id=999":                    http.StatusNotFound,  // no such classroom
		"/classroom/students?classroom_id=x":             http.StatusBadRequest,
		"/student/score?userID=abc&classroomID=1":        http.StatusBadRequest,
	} {
		if resp, _ := do(t, c, http.MethodGet, path, nil, nil); resp.StatusCode != want {
			t.Errorf("GET %s: %d, want %d", path, resp.StatusCode, want)
		}
	}

	// A copy of the session cookie taken before logout (another device, or a stolen cookie).
	copied := newClient(t)
	u, _ := url.Parse(testServer.URL)
	copied.Jar.SetCookies(u, c.Jar.Cookies(u))
	if resp, _ := do(t, copied, http.MethodGet, "/home", nil, nil); resp.StatusCode != http.StatusOK {
		t.Fatalf("copied cookie before logout: %d, want 200", resp.StatusCode)
	}

	// Logout deletes the cookie, so the same client is logged out afterwards...
	postForm(t, c, "/logout", nil, nil)
	if resp, _ := do(t, c, http.MethodGet, "/home", nil, nil); resp.StatusCode != http.StatusSeeOther {
		t.Errorf("/home after logout: %d, want 303", resp.StatusCode)
	}
	// ...and the server invalidates the session, so the copy stops working too.
	if resp, _ := do(t, copied, http.MethodGet, "/home", nil, nil); resp.StatusCode != http.StatusSeeOther {
		t.Errorf("copied cookie after logout: %d, want 303 (session should be invalidated server-side)", resp.StatusCode)
	}
	// Logging in again works.
	teacherClient(t)
}

func TestIntegrationCSRF(t *testing.T) {
	requireDB(t)
	c := teacherClient(t)
	form := url.Values{"classname": {"IT CSRF class"}, "section": {"x"}, "description": {""}}

	resp, _ := postForm(t, c, "/createclassroom", form, map[string]string{"Sec-Fetch-Site": "cross-site", "Origin": "https://evil.example"})
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("cross-site POST: %d, want 403", resp.StatusCode)
	}
	if n := count(t, "SELECT COUNT(*) FROM classrooms WHERE classroom_name = 'IT CSRF class'"); n != 0 {
		t.Fatalf("cross-site POST created %d classroom(s)", n)
	}

	postForm(t, c, "/createclassroom", form, map[string]string{"Sec-Fetch-Site": "same-origin"})
	if n := count(t, "SELECT COUNT(*) FROM classrooms WHERE classroom_name = 'IT CSRF class'"); n != 1 {
		t.Errorf("same-origin POST created %d classroom(s), want 1", n)
	}
}

func TestIntegrationAddStudentsIsAtomic(t *testing.T) {
	requireDB(t)
	postJSON(t, newClient(t), "/game/register", map[string]string{
		"firstname": "Atomic", "lastname": "Test", "username": "it_atomic", "section": "x", "classnumber": "1", "password": "pw",
	}, "")
	var id int
	if err := testDB.QueryRow("SELECT user_id FROM users WHERE username = 'it_atomic'").Scan(&id); err != nil {
		t.Fatal(err)
	}

	c := teacherClient(t)
	resp, _ := postForm(t, c, "/addstudents", url.Values{"classroomID": {"1"}, "userID": {fmt.Sprint(id), "99999"}}, nil)
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("batch with a nonexistent student: %d, want 500", resp.StatusCode)
	}
	if n := count(t, "SELECT COUNT(*) FROM enrollments WHERE student_id = ?", id); n != 0 {
		t.Fatalf("failed batch still enrolled the valid student (%d rows)", n)
	}
	if resp, _ := postForm(t, c, "/addstudents", url.Values{"classroomID": {"1"}, "userID": {"abc"}}, nil); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("malformed student id: %d, want 400", resp.StatusCode)
	}
	postForm(t, c, "/addstudents", url.Values{"classroomID": {"1"}, "userID": {fmt.Sprint(id)}}, nil)
	if n := count(t, "SELECT COUNT(*) FROM enrollments WHERE student_id = ?", id); n != 1 {
		t.Errorf("valid add: %d enrollment rows, want 1", n)
	}
}

func TestIntegrationGameLoginAndSaves(t *testing.T) {
	requireDB(t)
	// A registered student no teacher has enrolled gets a message, not a 500.
	postJSON(t, newClient(t), "/game/register", map[string]string{
		"firstname": "Not", "lastname": "Enrolled", "username": "it_unenrolled", "section": "x", "classnumber": "1", "password": "pw",
	}, "")
	if out := gameLogin(t, "it_unenrolled"); out["success"] != false || !strings.Contains(fmt.Sprint(out["error_text"]), "not in a class") {
		t.Errorf("unenrolled login = %v", out)
	}

	studentID := idOf(t, "SELECT user_id FROM users WHERE username = 'it_student'")
	otherStudentID := idOf(t, "SELECT MIN(user_id) FROM users WHERE usertype = 'student' AND username <> 'it_student'")
	login := gameLogin(t, "it_student")
	token, _ := login["token"].(string)
	if login["success"] != true || token == "" {
		t.Fatalf("it_student login = %v", login)
	}
	c := newClient(t)
	if resp, _ := postJSON(t, c, "/game/getsavedata", map[string]any{}, ""); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("load without token: %d, want 401", resp.StatusCode)
	}

	load := func() map[string]any {
		resp, body := postJSON(t, c, "/game/getsavedata", map[string]any{}, token)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("load: %d %s", resp.StatusCode, body)
		}
		var save map[string]any
		json.Unmarshal([]byte(body), &save)
		return save
	}
	before := load()
	if len(before) != 30 || before["student_id"] != float64(studentID) {
		t.Fatalf("loaded save has %d keys, student_id %v; want 30 and %d", len(before), before["student_id"], studentID)
	}

	// A partial save with a spoofed student_id and a null: merged into it_student's save only.
	resp, _ := postJSON(t, c, "/game/postsavedata", map[string]any{
		"student_id": otherStudentID, "current_quest": "it_quest", "rock_removed": nil, "player_badges": map[string]bool{"sword": true},
	}, token)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("save: %d", resp.StatusCode)
	}
	after := load()
	badges, _ := after["player_badges"].(map[string]any)
	if after["current_quest"] != "it_quest" || badges["sword"] != true || badges["shiny_rock"] != before["player_badges"].(map[string]any)["shiny_rock"] || after["rock_removed"] != before["rock_removed"] {
		t.Errorf("merge result wrong: quest=%v sword=%v shiny_rock=%v rock_removed=%v", after["current_quest"], badges["sword"], badges["shiny_rock"], after["rock_removed"])
	}
	if n := count(t, "SELECT COUNT(*) FROM save_states WHERE student_id = ?", otherStudentID); n != 0 {
		t.Errorf("a body student_id wrote a save for student %d", otherStudentID)
	}

	for score, want := range map[int]int{-3: http.StatusBadRequest, 7: http.StatusOK} {
		resp, _ := postJSON(t, c, "/game/add/statistics/quiz", map[string]int{"ClassroomID": 1, "MinigameID": 11, "Score": score}, token)
		if resp.StatusCode != want {
			t.Errorf("quiz score %d: %d, want %d", score, resp.StatusCode, want)
		}
	}
}

func TestIntegrationStudentWordedStatisticsHaveFractions(t *testing.T) {
	requireDB(t)
	studentID := idOf(t, "SELECT user_id FROM users WHERE username = 'it_student'")
	stats, err := database.GetStudentWordedStatistics(context.Background(), studentID, 3, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(stats) == 0 {
		t.Fatal("classroom 1 has no worded questions for minigame 3")
	}
	for _, st := range stats {
		if st.QuestionText == "" || st.Fraction1_Denominator == 0 || st.Fraction2_Denominator == 0 {
			t.Errorf("question %d: text %q, fractions %d/%d and %d/%d; the student page computes the answer from these",
				st.QuestionID, st.QuestionText, st.Fraction1_Numerator, st.Fraction1_Denominator, st.Fraction2_Numerator, st.Fraction2_Denominator)
		}
	}
}
