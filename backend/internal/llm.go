package internal

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type SimplifyResult struct {
	Simplified string   `json:"simplified"`
	Sentences  []string `json:"sentences"`
}

type AnalyzeResult struct {
	Translation string `json:"translation"`
	Vocab       []struct {
		Word    string `json:"word"`
		Pos     string `json:"pos"`
		Meaning string `json:"meaning"`
	} `json:"vocab"`
	Grammar string `json:"grammar"`
	Notes   string `json:"notes"`
}

func Simplify(db *sql.DB, or *ORClient, w http.ResponseWriter, r *http.Request) {
	pid, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", 400)
		return
	}
	level := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("level")))
	if level == "" {
		level = "B1"
	}
	force := r.URL.Query().Get("force") == "1"

	settings, err := LoadSettings(db)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	promptHash := Sha256Hex(settings.SimplifyPrompt)

	if !force {
		row := db.QueryRow(`SELECT simplified_text, sentences_json FROM simplifications WHERE paragraph_id=? AND level=? AND prompt_hash=?`, pid, level, promptHash)
		var sim, sents string
		if err := row.Scan(&sim, &sents); err == nil {
			var arr []string
			_ = json.Unmarshal([]byte(sents), &arr)
			WriteJSON(w, 200, map[string]any{"simplified": sim, "sentences": arr, "cached": true})
			return
		}
	}

	row := db.QueryRow(`SELECT original_text FROM paragraphs WHERE id=?`, pid)
	var orig string
	if err := row.Scan(&orig); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "paragraph not found", 404)
			return
		}
		http.Error(w, err.Error(), 500)
		return
	}
	prompt := ApplyPrompt(settings.SimplifyPrompt, level, orig) + "\n\nParagraph:\n" + orig
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	content, err := or.Chat(ctx, settings, settings.SimplifyModel, prompt, settings.SimplifyReasoning)
	if err != nil {
		http.Error(w, err.Error(), 502)
		return
	}
	var res SimplifyResult
	if err := json.Unmarshal([]byte(StripFences(content)), &res); err != nil {
		http.Error(w, "bad model output: "+err.Error()+"; raw="+content, 502)
		return
	}
	if res.Simplified == "" {
		http.Error(w, "empty simplification", 502)
		return
	}
	if len(res.Sentences) == 0 {
		res.Sentences = splitSentences(res.Simplified)
	}
	sentsJSON, _ := json.Marshal(res.Sentences)
	_, _ = db.Exec(`INSERT INTO simplifications(paragraph_id,level,simplified_text,sentences_json,model,prompt_hash,created_at)
	    VALUES(?,?,?,?,?,?,?)
	    ON CONFLICT(paragraph_id,level,prompt_hash) DO UPDATE SET simplified_text=excluded.simplified_text, sentences_json=excluded.sentences_json, created_at=excluded.created_at`,
		pid, level, res.Simplified, string(sentsJSON), settings.SimplifyModel, promptHash, time.Now().Unix())
	WriteJSON(w, 200, map[string]any{"simplified": res.Simplified, "sentences": res.Sentences, "cached": false})
}

func Analyze(db *sql.DB, or *ORClient, w http.ResponseWriter, r *http.Request) {
	var body struct {
		Text  string `json:"text"`
		Level string `json:"level"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad body", 400)
		return
	}
	body.Text = strings.TrimSpace(body.Text)
	if body.Text == "" {
		http.Error(w, "text required", 400)
		return
	}
	level := strings.ToUpper(strings.TrimSpace(body.Level))
	if level == "" {
		level = "B1"
	}
	settings, err := LoadSettings(db)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	promptHash := Sha256Hex(settings.AnalyzePrompt)
	sentHash := Sha256Hex(body.Text)

	row := db.QueryRow(`SELECT analysis_json FROM analyses WHERE sentence_hash=? AND level=? AND prompt_hash=?`, sentHash, level, promptHash)
	var blob string
	if err := row.Scan(&blob); err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(blob))
		return
	}

	prompt := ApplyPrompt(settings.AnalyzePrompt, level, body.Text)
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	content, err := or.Chat(ctx, settings, settings.AnalyzeModel, prompt, settings.AnalyzeReasoning)
	if err != nil {
		http.Error(w, err.Error(), 502)
		return
	}
	clean := StripFences(content)
	var res AnalyzeResult
	if err := json.Unmarshal([]byte(clean), &res); err != nil {
		http.Error(w, "bad model output: "+err.Error()+"; raw="+content, 502)
		return
	}
	_, _ = db.Exec(`INSERT INTO analyses(sentence_hash,level,text,analysis_json,model,prompt_hash,created_at)
	    VALUES(?,?,?,?,?,?,?)
	    ON CONFLICT(sentence_hash,level,prompt_hash) DO UPDATE SET analysis_json=excluded.analysis_json, created_at=excluded.created_at`,
		sentHash, level, body.Text, clean, settings.AnalyzeModel, promptHash, time.Now().Unix())
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(clean))
}

func splitSentences(s string) []string {
	out := []string{}
	cur := strings.Builder{}
	runes := []rune(s)
	for i, r := range runes {
		cur.WriteRune(r)
		if r == '.' || r == '!' || r == '?' {
			next := byte(' ')
			if i+1 < len(runes) {
				next = byte(runes[i+1])
			}
			if next == ' ' || next == '\n' || i == len(runes)-1 {
				txt := strings.TrimSpace(cur.String())
				if txt != "" {
					out = append(out, txt)
				}
				cur.Reset()
			}
		}
	}
	if rem := strings.TrimSpace(cur.String()); rem != "" {
		out = append(out, rem)
	}
	return out
}
