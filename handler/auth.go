package handler

import (
	// "crypto/rand"
	// "encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"soln-teachermodule/database"
	"soln-teachermodule/view/auth"

	"github.com/go-sql-driver/mysql"
	"github.com/gorilla/sessions"
)

const (
	sessionUserKey        = "teacher"
	sessionAccessTokenKey = "access_token"
)

var store = sessions.NewCookieStore([]byte(os.Getenv("SESSION_SECRET")))

func HandleLoginIndex(w http.ResponseWriter, r *http.Request) error {
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

	hxRedirect(w, r, "/home")
	return nil
}

func setAuthCookie(w http.ResponseWriter, r *http.Request) error {
	session, _ := store.Get(r, sessionUserKey)
	session.Values["authenticated"] = true
	session.Values["teacherID"], _ = database.GetTeacherID(w, r)
	return session.Save(r, w)
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
			// check for duplicate entry
			var mysqlErr *mysql.MySQLError
			if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
				return render(w, r, auth.RegisterForm(credentials, auth.RegisterErrors{
					RegisterErrors: "That username is already taken.",
				}))
			}
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

func HandleLogoutCreate(w http.ResponseWriter, r *http.Request) error {
	store = sessions.NewCookieStore([]byte(os.Getenv("SESSION_SECRET")))
	session, _ := store.Get(r, sessionUserKey)
	session.Values["authenticated"] = false
	session.Save(r, w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
	return nil
}
