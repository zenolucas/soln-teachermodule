package handler

import (
	"log/slog"
	"net/http"
	"soln-teachermodule/types"

	"github.com/a-h/templ"
)

func Make(h func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			slog.Error("internal server error!!", "err", err, "path: ", r.URL.Path)
		}
	}
}

func GetAuthenticatedUser(r *http.Request) types.AuthenticatedUser {
	user, ok := r.Context().Value(types.UserContextKey).(types.AuthenticatedUser)
	if !ok {
		return types.AuthenticatedUser{}
	}
	return user
} 

func render(w http.ResponseWriter, r *http.Request, component templ.Component) error {
	return component.Render(r.Context(), w)
}