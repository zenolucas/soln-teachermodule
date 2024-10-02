package handler

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"soln-teachermodule/database"
	"soln-teachermodule/view/auth"
	"soln-teachermodule/view/home"

	"github.com/gorilla/sessions"
)

const (
	sessionUserKey        = "teacher"
	sessionAccessTokenKey = "access_token"
)

func HandleLoginIndex(w http.ResponseWriter, r *http.Request) error {
	// user := GetAuthenticatedUser(r)
	// fmt.Print("is htis executed?")
	// fmt.Printf("%v+\n", user.LoggedIn)
	return render(w, r, auth.Login())
}

func HandleLoginCreate(w http.ResponseWriter, r *http.Request) error {
	credentials := auth.LoginParams{
		Username: r.FormValue("username"),
		Password: r.FormValue("password"),
	}

	// authenticate the user
	if err := database.AuthenticateWebUser(credentials.Username, credentials.Password); err != nil {
		// if an error occurs
		return render(w, r, auth.LoginForm(credentials, auth.LoginErrors{
			InvalidCredentials: "Invalid username or password",
		}))
	}
	if err := setAuthCookie(w, r); err != nil {
		return err
	}

	return render(w, r, home.Index())
}

func HandleLoginGame(w http.ResponseWriter, r *http.Request) error {
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
		Success bool `json:"success"`
	}

	log.Printf("Received data: %+v", data.Password)
	// authenticate student
	if database.AuthenticateSolnUser(data.Username, data.Password) {
		response := LoginResponse{Success: true}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	} else {
		response := LoginResponse{Success: false}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}

	return nil
}

type RegisterParams struct {
	Username        string
	Password        string
	ConfirmPassword string
}

func HandleRegisterIndex(w http.ResponseWriter, r *http.Request) error {
	return render(w, r, auth.Register())
}

func HandleRegisterCreate(w http.ResponseWriter, r *http.Request) error {
	credentials := auth.RegisterParams{
		Username:        r.FormValue("username"),
		Password:        r.FormValue("password"),
		ConfirmPassword: r.FormValue("confirmPassword"),
	}

	// TODO: username is already taken error

	if credentials.Password == credentials.ConfirmPassword {
		if err := database.RegisterAccount(w, r); err != nil {
			return err
		} else {
			fmt.Print("Account registered successfully!")
			return render(w, r, auth.LoginForm(auth.LoginParams{}, auth.LoginErrors{}))
		}
	} else {
		return render(w, r, auth.RegisterForm(credentials, auth.RegisterErrors{
			RegisterErrors: "Passwords do not match, please try again.",
		}))
	}

}

func generateSessionToken() (string, error) {
	// Create a random 32-byte token
	token := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, token)
	if err != nil {
		return "", err
	}
	// Encode the token to a base64 string
	return base64.URLEncoding.EncodeToString(token), nil
}

func setAuthCookie(w http.ResponseWriter, r *http.Request) error {
	store := sessions.NewCookieStore([]byte(os.Getenv("SESSION_SECRET")))
	session, _ := store.Get(r, sessionUserKey)
	session.Values["authenticated"] = true
	session.Values["teacherID"], _ = database.GetTeacherID(w, r)
	return session.Save(r, w)
}

func HandleLogoutCreate(w http.ResponseWriter, r *http.Request) error {
	store := sessions.NewCookieStore([]byte(os.Getenv("SESSION_SECRET")))
	session, _ := store.Get(r, sessionUserKey)
	fmt.Print(session.Values[sessionAccessTokenKey])
	session.Values["authenticated"] = false
	session.Save(r, w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
	return nil
}
