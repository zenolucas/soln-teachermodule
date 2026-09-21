package handler

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/securecookie"
)

const (
	gameTokenName   = "game_token"
	gameTokenMaxAge = 24 * 60 * 60 // seconds; matches the 24h expiry the unused SaveSessionToken already used
)

// gameTokenCodec signs (but does not encrypt - a student_id isn't secret, only tamper
// resistance is needed) the bearer tokens issued to the Godot game client at
// /game/login. Configured by InitGameTokenStore, which must run once at startup after
// .env has been loaded, for the same reason handler.store is configured lazily: a
// package-level initializer here would read an empty GAME_TOKEN_SECRET.
//
// This uses a separate secret from the teacher web session (GAME_TOKEN_SECRET, not
// SESSION_SECRET) so a leak of one credential doesn't let an attacker forge the other.
var gameTokenCodec *securecookie.SecureCookie

// InitGameTokenStore reads GAME_TOKEN_SECRET from the environment and configures the
// game token codec. Fails fast if the secret is unset or too short.
func InitGameTokenStore() error {
	secret := os.Getenv("GAME_TOKEN_SECRET")
	if len(secret) < 32 {
		return fmt.Errorf("GAME_TOKEN_SECRET must be set to at least 32 characters (got %d)", len(secret))
	}
	gameTokenCodec = securecookie.New([]byte(secret), nil)
	gameTokenCodec.MaxAge(gameTokenMaxAge)
	return nil
}

type gameTokenClaims struct {
	StudentID int
}

// issueGameToken mints a signed, self-expiring token for the given student, to be
// returned to the game client at login and sent back on every subsequent /game/*
// request that acts on that student's data.
func issueGameToken(studentID int) (string, error) {
	return gameTokenCodec.Encode(gameTokenName, gameTokenClaims{StudentID: studentID})
}

// authenticateGameRequest reads the bearer token from the Authorization header and
// returns the student_id it was issued for. Every /game/* route that reads or writes
// a specific student's data must call this and use the returned student_id instead of
// trusting one posted in the request body - the body's value can't be trusted once an
// attacker can send arbitrary JSON (see SEC-04).
func authenticateGameRequest(r *http.Request) (int, error) {
	authHeader := r.Header.Get("Authorization")
	token, ok := strings.CutPrefix(authHeader, "Bearer ")
	if !ok || token == "" {
		return 0, fmt.Errorf("missing or malformed Authorization header")
	}

	var claims gameTokenClaims
	if err := gameTokenCodec.Decode(gameTokenName, token, &claims); err != nil {
		return 0, fmt.Errorf("invalid or expired token: %w", err)
	}
	return claims.StudentID, nil
}
