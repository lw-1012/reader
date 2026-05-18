package internal

import (
	"database/sql"
	"encoding/json"
)

type Settings struct {
	APIKey         string `json:"api_key"`
	BaseURL        string `json:"base_url"`
	SimplifyModel  string `json:"simplify_model"`
	AnalyzeModel   string `json:"analyze_model"`
	TTSModel       string `json:"tts_model"`
	Voice          string `json:"voice"`
	Level          string `json:"level"`
	SimplifyPrompt string `json:"simplify_prompt"`
	AnalyzePrompt  string `json:"analyze_prompt"`
	TTSInstruction string `json:"tts_instruction"`
	// Reasoning effort per task: "" = model default (omit field),
	// otherwise one of: none/minimal/low/medium/high/xhigh.
	SimplifyReasoning string `json:"simplify_reasoning"`
	AnalyzeReasoning  string `json:"analyze_reasoning"`
	// ProviderOnly restricts which OpenRouter providers may serve the
	// simplify/analyze chat calls. Comma/space separated provider slugs
	// (e.g. "openai, anthropic"). Empty = no restriction. TTS unaffected.
	ProviderOnly string `json:"provider_only"`
}

const DefaultSimplifyPrompt = `You are an English text simplifier specialized in adapting literature for second-language learners.

Rewrite the input paragraph at CEFR level {LEVEL}. Preserve the meaning, narrative voice, and key ideas; reduce vocabulary complexity and sentence length appropriate to {LEVEL}. Keep roughly the same paragraph length.

Return strict JSON only (no markdown, no commentary):
{
  "simplified": "the rewritten paragraph as a single string",
  "sentences": ["sentence 1.", "sentence 2.", "..."]
}

Each item in "sentences" MUST be a substring of "simplified", and concatenating them with single spaces reconstructs it (just split by sentence boundaries).`

const DefaultAnalyzePrompt = `You are an English tutor for Chinese learners at CEFR level {LEVEL}.

Analyze the given English sentence and return strict JSON only (no markdown, no commentary):
{
  "translation": "中文翻译",
  "vocab": [
    {"word": "原词或词组", "pos": "词性缩写", "meaning": "中文释义"}
  ],
  "grammar": "关键语法点（中文，简明）",
  "notes": "其他需要注意的点（中文，可选，无则空字符串）"
}

Only include vocabulary entries that an English learner at {LEVEL} would benefit from looking up. Limit vocab to at most 6 entries.

Sentence:
{TEXT}`

const DefaultTTSInstruction = `Speak in a clear, natural, native American English accent at a moderate pace, suitable for an English learner.`

func defaultSettings() Settings {
	return Settings{
		BaseURL:        "https://openrouter.ai/api/v1",
		SimplifyModel:  "openai/gpt-4o-mini",
		AnalyzeModel:   "openai/gpt-4o-mini",
		TTSModel:       "openai/gpt-4o-mini-tts",
		Voice:          "alloy",
		Level:          "B1",
		SimplifyPrompt: DefaultSimplifyPrompt,
		AnalyzePrompt:  DefaultAnalyzePrompt,
		TTSInstruction: DefaultTTSInstruction,
		// keep simplification cheap by default; analysis benefits from
		// light reasoning so default it to "low".
		SimplifyReasoning: "",
		AnalyzeReasoning:  "low",
	}
}

func LoadSettings(db *sql.DB) (Settings, error) {
	s := defaultSettings()
	row := db.QueryRow(`SELECT value FROM settings WHERE key = 'app'`)
	var blob string
	err := row.Scan(&blob)
	if err == sql.ErrNoRows {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	_ = json.Unmarshal([]byte(blob), &s)
	if s.BaseURL == "" {
		s.BaseURL = "https://openrouter.ai/api/v1"
	}
	if s.SimplifyPrompt == "" {
		s.SimplifyPrompt = DefaultSimplifyPrompt
	}
	if s.AnalyzePrompt == "" {
		s.AnalyzePrompt = DefaultAnalyzePrompt
	}
	if s.TTSInstruction == "" {
		s.TTSInstruction = DefaultTTSInstruction
	}
	return s, nil
}

func SaveSettings(db *sql.DB, s Settings) error {
	b, _ := json.Marshal(s)
	_, err := db.Exec(`INSERT INTO settings(key,value) VALUES('app',?)
	    ON CONFLICT(key) DO UPDATE SET value=excluded.value`, string(b))
	return err
}

// PublicSettings hides the api key for GET responses.
func (s Settings) Public() map[string]any {
	masked := ""
	if s.APIKey != "" {
		masked = "********"
	}
	return map[string]any{
		"api_key_set":     s.APIKey != "",
		"api_key_masked":  masked,
		"base_url":        s.BaseURL,
		"simplify_model":  s.SimplifyModel,
		"analyze_model":   s.AnalyzeModel,
		"tts_model":       s.TTSModel,
		"voice":           s.Voice,
		"level":           s.Level,
		"simplify_prompt":    s.SimplifyPrompt,
		"analyze_prompt":     s.AnalyzePrompt,
		"tts_instruction":    s.TTSInstruction,
		"simplify_reasoning": s.SimplifyReasoning,
		"analyze_reasoning":  s.AnalyzeReasoning,
		"provider_only":      s.ProviderOnly,
	}
}
