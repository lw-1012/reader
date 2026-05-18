package internal

import (
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"time"
)

const sessionCookie = "reader_session"
const sessionTTL = 30 * 24 * time.Hour

type Auth struct {
	DB       *sql.DB
	Password string
}

func NewAuth(db *sql.DB) *Auth {
	pwd := os.Getenv("READER_PASSWORD")
	if pwd == "" {
		pwd = "reader"
	}
	return &Auth{DB: db, Password: pwd}
}

func randToken() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (a *Auth) Login(w http.ResponseWriter, r *http.Request) {
	var body struct{ Password string `json:"password"` }
	_ = json.NewDecoder(r.Body).Decode(&body)
	if subtle.ConstantTimeCompare([]byte(body.Password), []byte(a.Password)) != 1 {
		http.Error(w, "wrong password", http.StatusUnauthorized)
		return
	}
	tok := randToken()
	now := time.Now().Unix()
	exp := time.Now().Add(sessionTTL).Unix()
	_, err := a.DB.Exec(`INSERT INTO sessions(token,created_at,expires_at) VALUES(?,?,?)`, tok, now, exp)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	secure := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    tok,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(exp, 0),
	})
	WriteJSON(w, 200, map[string]any{"ok": true})
}

func (a *Auth) Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		_, _ = a.DB.Exec(`DELETE FROM sessions WHERE token=?`, c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1})
	WriteJSON(w, 200, map[string]any{"ok": true})
}

func (a *Auth) Check(r *http.Request) bool {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return false
	}
	row := a.DB.QueryRow(`SELECT expires_at FROM sessions WHERE token=?`, c.Value)
	var exp int64
	if err := row.Scan(&exp); err != nil {
		return false
	}
	return exp > time.Now().Unix()
}

func (a *Auth) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.Check(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
