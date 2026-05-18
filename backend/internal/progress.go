package internal

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

func GetProgress(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", 400)
		return
	}
	row := db.QueryRow(`SELECT last_paragraph_id, level FROM progress WHERE book_id=?`, id)
	var pid sql.NullInt64
	var level sql.NullString
	if err := row.Scan(&pid, &level); err != nil {
		WriteJSON(w, 200, map[string]any{"book_id": id, "last_paragraph_id": nil, "level": nil})
		return
	}
	var gi *int
	if pid.Valid {
		var v int
		if err := db.QueryRow(`SELECT global_index FROM paragraphs WHERE id=?`, pid.Int64).Scan(&v); err == nil {
			gi = &v
		}
	}
	WriteJSON(w, 200, map[string]any{
		"book_id":           id,
		"last_paragraph_id": pid.Int64,
		"last_global_index": gi,
		"level":             level.String,
	})
}

func PutProgress(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", 400)
		return
	}
	var body struct {
		ParagraphID int64  `json:"paragraph_id"`
		Level       string `json:"level"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad body", 400)
		return
	}
	_, err = db.Exec(`INSERT INTO progress(book_id,last_paragraph_id,level,updated_at) VALUES(?,?,?,?)
	    ON CONFLICT(book_id) DO UPDATE SET last_paragraph_id=excluded.last_paragraph_id, level=excluded.level, updated_at=excluded.updated_at`,
		id, body.ParagraphID, body.Level, time.Now().Unix())
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	WriteJSON(w, 200, map[string]any{"ok": true})
}
