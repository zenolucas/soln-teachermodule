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
	to := r.URL.Query().Get("to")
	registered := r.URL.Query().Get("registered") == "1"
	return render(w, r, auth.Login(to, registered))
}

func HandleLoginCreate(w http.ResponseWriter, r *http.Request) error {
	// Never echo the submitted password back into the re-rendered form (see FE-24) -
	// credentials.Password is intentionally left unset; it's only read from the
	// request below, for the authentication check itself.
	credentials := auth.LoginParams{
		Username: r.FormValue("username"),
	}
	to := r.FormValue("to")

	// authenticate the user
	if err := database.AuthenticateWebUser(r.Context(), credentials.Username, r.FormValue("password")); err != nil {
		// if an error occurs
		return render(w, r, auth.LoginForm(credentials, auth.LoginErrors{
			InvalidCredentials: "Invalid username or password",
		}, to, false))
	}

	if err := setAuthCookie(w, r); err != nil {
		return err
	}

	// Send the teacher back to the page WithAuth redirected them away from, e.g. a
	// statistics page whose session had expired - but only if it's a same-site path
	// (see isLocalRedirect), so a crafted `to` can't turn this into an open redirect
	// (see FE-07).
	target := "/home"
	if isLocalRedirect(to) {
		target = to
	}
	hxRedirect(w, r, target)
	return nil
}

func setAuthCookie(w http.ResponseWriter, r *http.Request) error {
	// Resolve the teacher before touching the session: an authenticated session without a
	// teacherID would pass WithAuth but fail every page that needs it.
	teacherID, err := database.GetTeacherID(r.Context(), r.FormValue("username"))
	if err != nil {
		return err
	}
	sessionVersion, err := database.GetSessionVersion(r.Context(), teacherID)
	if err != nil {
		return err
	}

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
	session.Values["teacherID"] = teacherID
	session.Values["sessionVersion"] = sessionVersion
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
			// Redirect to the login page instead of rendering it in place (see FE-06) -
			// swapping in a blank LoginForm left the URL at /register with nothing
			// telling the teacher the account was actually created, and a refresh
			// re-showed the register form as if nothing had happened.
			hxRedirect(w, r, "/login?registered=1")
			return nil
		}
	} else {
		return render(w, r, auth.RegisterForm(credentials, auth.RegisterErrors{
			RegisterErrors: "Passwords do not match, please try again.",
		}))
	}

}

func HandleLogoutCreate(w http.ResponseWriter, r *http.Request) error {
	session, _ := store.Get(r, sessionUserKey)
	// Invalidate the session server-side: bumping the version makes every copy of this teacher's
	// cookie (other devices, a copied cookie) fail WithAuth. Then delete the browser's copy too.
	if teacherID, ok := session.Values["teacherID"].(int); ok {
		if err := database.BumpSessionVersion(r.Context(), teacherID); err != nil {
			return err
		}
	}
	session.Values = map[interface{}]interface{}{}
	session.Options.MaxAge = -1
	if err := session.Save(r, w); err != nil {
		return err
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
	return nil
}
