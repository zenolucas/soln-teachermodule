package handler

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"soln-teachermodule/database"
	"soln-teachermodule/util"
	"soln-teachermodule/view/auth"
	"github.com/go-sql-driver/mysql"
	"github.com/gorilla/sessions"
)

const (
	sessionUserKey = "teacher"
)

// store is configured by InitSessionStore, which must run once at startup after any
// .env file has been loaded (a package-level initializer here would run before that
// load happens, and would read an empty SESSION_SECRET). It is nil until then; every
// route that uses it sits behind main() calling InitSessionStore first.
var store *sessions.CookieStore

// InitSessionStore reads SESSION_SECRET from the environment and configures the
// session store. The old hardcoded key this replaces is committed in git history, so
// it must not be reused even as a fallback - every deployment needs its own secret.
func InitSessionStore() error {
	secret := os.Getenv("SESSION_SECRET")
	if len(secret) < 32 {
		return fmt.Errorf("SESSION_SECRET must be set to at least 32 characters (got %d)", len(secret))
	}
	store = sessions.NewCookieStore([]byte(secret))
	return nil
}

func HandleLoginIndex(w http.ResponseWriter, r *http.Request) error {
	return render(w, r, auth.Login())
}

func HandleLoginCreate(w http.ResponseWriter, r *http.Request) error {
	credentials := auth.LoginParams{
		Username: r.FormValue("username"),
		Password: r.FormValue("password"),
	}

	// authenticate the user
	if err := database.AuthenticateWebUser(r.Context(), credentials.Username, credentials.Password); err != nil {
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
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   3600 * 8,
		HttpOnly: true,
		// Only send the cookie over TLS in production. Defaults to false so local dev
		// (typically plain HTTP) keeps working; set COOKIE_SECURE=true once the app is
		// actually served over HTTPS.
		Secure:   os.Getenv("COOKIE_SECURE") == "true",
		SameSite: http.SameSiteLaxMode,
	}

	session, _ := store.Get(r, sessionUserKey)

	session.Values["authenticated"] = true
	session.Values["teacherID"], _ = database.GetTeacherID(w, r)
	return session.Save(r, w)
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

	fmt.Println("username is : ", credentials.Username)

	errorMessage, validCredentials := util.ValidateUsername(credentials.Username)
	if validCredentials {
	errorMessage, validCredentials = util.ValidatePassword(credentials.Password)
		if !validCredentials {
			return render(w, r, auth.RegisterForm(credentials, auth.RegisterErrors{
				RegisterErrors: errorMessage,
			}))
		}
	} else {
		return render(w, r, auth.RegisterForm(credentials, auth.RegisterErrors{
			RegisterErrors: errorMessage,
		}))
	}

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
	// store = sessions.NewCookieStore([]byte(os.Getenv("SESSION_SECRET")))
	session, _ := store.Get(r, sessionUserKey)
	session.Values["authenticated"] = false
	session.Save(r, w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
	return nil
}
