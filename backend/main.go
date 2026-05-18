package main

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"reader/internal"
)

//go:embed all:webui
var webuiFS embed.FS

func main() {
	dataDir := getenv("READER_DATA_DIR", "./data")
	_ = os.MkdirAll(dataDir, 0o755)
	dbPath := filepath.Join(dataDir, "reader.db")
	ttsDir := filepath.Join(dataDir, "tts")

	db, err := internal.OpenDB(dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	or := internal.NewORClient()
	auth := internal.NewAuth(db)
	tts := &internal.TTSHandler{DB: db, OR: or, CacheDir: ttsDir}

	mux := http.NewServeMux()

	// auth
	mux.HandleFunc("POST /api/auth/login", auth.Login)
	mux.HandleFunc("POST /api/auth/logout", auth.Logout)
	mux.HandleFunc("GET /api/auth/check", func(w http.ResponseWriter, r *http.Request) {
		internal.WriteJSON(w, 200, map[string]any{"authed": auth.Check(r)})
	})

	// settings
	mux.Handle("GET /api/settings", auth.Require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s, err := internal.LoadSettings(db)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		internal.WriteJSON(w, 200, s.Public())
	})))
	mux.Handle("PUT /api/settings", auth.Require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cur, _ := internal.LoadSettings(db)
		var patch map[string]any
		if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
			http.Error(w, "bad body", 400)
			return
		}
		// merge
		if v, ok := patch["api_key"].(string); ok && v != "" && v != "********" {
			cur.APIKey = v
		}
		if v, ok := patch["base_url"].(string); ok && v != "" {
			cur.BaseURL = v
		}
		if v, ok := patch["simplify_model"].(string); ok {
			cur.SimplifyModel = v
		}
		if v, ok := patch["analyze_model"].(string); ok {
			cur.AnalyzeModel = v
		}
		if v, ok := patch["tts_model"].(string); ok {
			cur.TTSModel = v
		}
		if v, ok := patch["voice"].(string); ok {
			cur.Voice = v
		}
		if v, ok := patch["level"].(string); ok {
			cur.Level = v
		}
		if v, ok := patch["simplify_prompt"].(string); ok {
			cur.SimplifyPrompt = v
		}
		if v, ok := patch["analyze_prompt"].(string); ok {
			cur.AnalyzePrompt = v
		}
		if v, ok := patch["tts_instruction"].(string); ok {
			cur.TTSInstruction = v
		}
		if v, ok := patch["simplify_reasoning"].(string); ok {
			cur.SimplifyReasoning = v
		}
		if v, ok := patch["analyze_reasoning"].(string); ok {
			cur.AnalyzeReasoning = v
		}
		if err := internal.SaveSettings(db, cur); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		internal.WriteJSON(w, 200, cur.Public())
	})))

	// books
	mux.Handle("POST /api/books/import", auth.Require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		internal.ImportBook(db, w, r)
	})))
	mux.Handle("GET /api/books", auth.Require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		internal.ListBooks(db, w, r)
	})))
	mux.Handle("DELETE /api/books/{id}", auth.Require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		internal.DeleteBook(db, w, r)
	})))
	mux.Handle("GET /api/books/{id}", auth.Require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		internal.GetBook(db, w, r)
	})))
	mux.Handle("GET /api/books/{id}/paragraphs", auth.Require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		internal.GetParagraphs(db, w, r)
	})))
	mux.Handle("GET /api/books/{id}/progress", auth.Require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		internal.GetProgress(db, w, r)
	})))
	mux.Handle("PUT /api/books/{id}/progress", auth.Require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		internal.PutProgress(db, w, r)
	})))

	// LLM
	mux.Handle("POST /api/paragraphs/{id}/simplify", auth.Require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		internal.Simplify(db, or, w, r)
	})))
	mux.Handle("POST /api/analyze", auth.Require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		internal.Analyze(db, or, w, r)
	})))

	// TTS
	mux.Handle("GET /api/tts", auth.Require(http.HandlerFunc(tts.Serve)))

	// static UI
	sub, err := fs.Sub(webuiFS, "webui")
	if err != nil {
		log.Fatalf("webui fs: %v", err)
	}
	fileServer := http.FileServer(http.FS(sub))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// SPA fallback: if path has no extension and isn't an asset, serve index.html
		p := r.URL.Path
		if p != "/" && filepath.Ext(p) == "" && !strings.HasPrefix(p, "/api/") {
			data, err := fs.ReadFile(sub, "index.html")
			if err != nil {
				http.Error(w, "index missing", 500)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(data)
			return
		}
		fileServer.ServeHTTP(w, r)
	})

	addr := getenv("READER_ADDR", ":8080")
	srv := &http.Server{
		Addr:              addr,
		Handler:           withLog(mux),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("reader listening on %s (data=%s)", addr, dataDir)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func withLog(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t := time.Now()
		h.ServeHTTP(w, r)
		if !strings.HasPrefix(r.URL.Path, "/assets/") && !strings.HasSuffix(r.URL.Path, ".js") && !strings.HasSuffix(r.URL.Path, ".css") {
			log.Printf("%s %s (%s)", r.Method, r.URL.Path, time.Since(t))
		}
	})
}
