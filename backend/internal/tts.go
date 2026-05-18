package internal

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type TTSHandler struct {
	DB       *sql.DB
	OR       *ORClient
	CacheDir string
}

func (h *TTSHandler) Serve(w http.ResponseWriter, r *http.Request) {
	text := strings.TrimSpace(r.URL.Query().Get("text"))
	if text == "" {
		http.Error(w, "text required", 400)
		return
	}
	if len(text) > 4000 {
		http.Error(w, "text too long", 400)
		return
	}
	settings, err := LoadSettings(h.DB)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	voice := r.URL.Query().Get("voice")
	if voice == "" {
		voice = settings.Voice
	}
	model := settings.TTSModel
	key := Sha256Hex(model + "|" + voice + "|" + settings.TTSInstruction + "|" + text)

	row := h.DB.QueryRow(`SELECT file_path FROM tts_cache WHERE hash=?`, key)
	var path string
	if err := row.Scan(&path); err == nil {
		if _, err := os.Stat(path); err == nil {
			serveAudio(w, r, path)
			return
		}
		_, _ = h.DB.Exec(`DELETE FROM tts_cache WHERE hash=?`, key)
	}

	settingsOverride := settings
	settingsOverride.Voice = voice

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	data, ct, err := h.OR.TTS(ctx, settingsOverride, text)
	if err != nil {
		http.Error(w, err.Error(), 502)
		return
	}
	_ = os.MkdirAll(h.CacheDir, 0o755)
	fp := filepath.Join(h.CacheDir, key+".mp3")
	if err := os.WriteFile(fp, data, 0o644); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	_, _ = h.DB.Exec(`INSERT INTO tts_cache(hash,text,voice,model,file_path,bytes,created_at)
	    VALUES(?,?,?,?,?,?,?)
	    ON CONFLICT(hash) DO UPDATE SET file_path=excluded.file_path, bytes=excluded.bytes`,
		key, text, voice, model, fp, len(data), time.Now().Unix())
	_ = ct
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Write(data)
}

func serveAudio(w http.ResponseWriter, r *http.Request, path string) {
	f, err := os.Open(path)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer f.Close()
	info, _ := f.Stat()
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	http.ServeContent(w, r, filepath.Base(path), info.ModTime(), f)
}
