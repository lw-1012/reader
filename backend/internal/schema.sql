PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS settings (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
  token      TEXT PRIMARY KEY,
  created_at INTEGER NOT NULL,
  expires_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS books (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  title      TEXT NOT NULL,
  author     TEXT,
  meta_json  TEXT,
  created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS sections (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  book_id      INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
  parent_id    INTEGER REFERENCES sections(id) ON DELETE CASCADE,
  order_index  INTEGER NOT NULL,
  depth        INTEGER NOT NULL,
  title        TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sections_book ON sections(book_id);
CREATE INDEX IF NOT EXISTS idx_sections_parent ON sections(parent_id);

CREATE TABLE IF NOT EXISTS paragraphs (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  book_id       INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
  section_id    INTEGER NOT NULL REFERENCES sections(id) ON DELETE CASCADE,
  order_index   INTEGER NOT NULL,
  global_index  INTEGER NOT NULL,
  source_pid    TEXT,
  original_text TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_paragraphs_book_global ON paragraphs(book_id, global_index);
CREATE INDEX IF NOT EXISTS idx_paragraphs_section ON paragraphs(section_id, order_index);

CREATE TABLE IF NOT EXISTS simplifications (
  id              INTEGER PRIMARY KEY AUTOINCREMENT,
  paragraph_id    INTEGER NOT NULL REFERENCES paragraphs(id) ON DELETE CASCADE,
  level           TEXT NOT NULL,
  simplified_text TEXT NOT NULL,
  sentences_json  TEXT NOT NULL,
  model           TEXT,
  prompt_hash     TEXT,
  created_at      INTEGER NOT NULL,
  UNIQUE(paragraph_id, level, prompt_hash)
);
CREATE INDEX IF NOT EXISTS idx_simpl_paragraph ON simplifications(paragraph_id, level);

CREATE TABLE IF NOT EXISTS analyses (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  sentence_hash TEXT NOT NULL,
  level         TEXT NOT NULL,
  text          TEXT NOT NULL,
  analysis_json TEXT NOT NULL,
  model         TEXT,
  prompt_hash   TEXT,
  created_at    INTEGER NOT NULL,
  UNIQUE(sentence_hash, level, prompt_hash)
);

CREATE TABLE IF NOT EXISTS tts_cache (
  hash       TEXT PRIMARY KEY,
  text       TEXT NOT NULL,
  voice      TEXT NOT NULL,
  model      TEXT NOT NULL,
  file_path  TEXT NOT NULL,
  bytes      INTEGER,
  created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS progress (
  book_id           INTEGER PRIMARY KEY REFERENCES books(id) ON DELETE CASCADE,
  last_paragraph_id INTEGER REFERENCES paragraphs(id),
  level             TEXT,
  updated_at        INTEGER NOT NULL
);
