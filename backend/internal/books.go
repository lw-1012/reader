package internal

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type importBook struct {
	Title    string           `json:"title"`
	Author   string           `json:"author"`
	Chapters []importChapter  `json:"chapters"`
	Sections []importSection  `json:"sections"` // alternative root: flat sections
	Paragraphs []importPara   `json:"paragraphs"` // alternative root: flat paragraphs (single section)
}

type importChapter struct {
	ChapterNumber int             `json:"chapter_number"`
	Title         string          `json:"title"`
	Sections      []importSection `json:"sections"`
	Paragraphs    []importPara    `json:"paragraphs"`
}

type importSection struct {
	Title      string          `json:"title"`
	Paragraphs []importPara    `json:"paragraphs"`
	Sections   []importSection `json:"sections"`
}

type importPara struct {
	ParagraphID string `json:"paragraph_id"`
	Text        string `json:"text"`
}

func ImportBook(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<20) // 64MB
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	var ib importBook
	if err := json.Unmarshal(raw, &ib); err != nil {
		http.Error(w, "invalid json: "+err.Error(), 400)
		return
	}
	if strings.TrimSpace(ib.Title) == "" {
		http.Error(w, "title is required", 400)
		return
	}
	tx, err := db.Begin()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer tx.Rollback()

	res, err := tx.Exec(`INSERT INTO books(title,author,created_at) VALUES(?,?,?)`,
		ib.Title, ib.Author, time.Now().Unix())
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	bookID, _ := res.LastInsertId()

	globalIdx := 0
	orderCounter := 0
	var walk func(parentID *int64, depth int, secs []importSection, paras []importPara, title string) error
	walk = func(parentID *int64, depth int, secs []importSection, paras []importPara, title string) error {
		// create section row for this level
		var pid any
		if parentID != nil {
			pid = *parentID
		}
		orderCounter++
		secRes, err := tx.Exec(`INSERT INTO sections(book_id,parent_id,order_index,depth,title) VALUES(?,?,?,?,?)`,
			bookID, pid, orderCounter, depth, title)
		if err != nil {
			return err
		}
		sectionID, _ := secRes.LastInsertId()
		for i, p := range paras {
			if strings.TrimSpace(p.Text) == "" {
				continue
			}
			globalIdx++
			_, err := tx.Exec(`INSERT INTO paragraphs(book_id,section_id,order_index,global_index,source_pid,original_text) VALUES(?,?,?,?,?,?)`,
				bookID, sectionID, i+1, globalIdx, p.ParagraphID, p.Text)
			if err != nil {
				return err
			}
		}
		for _, s := range secs {
			if err := walk(&sectionID, depth+1, s.Sections, s.Paragraphs, s.Title); err != nil {
				return err
			}
		}
		return nil
	}

	if len(ib.Chapters) > 0 {
		for _, ch := range ib.Chapters {
			t := ch.Title
			if t == "" {
				t = fmt.Sprintf("Chapter %d", ch.ChapterNumber)
			}
			if err := walk(nil, 0, ch.Sections, ch.Paragraphs, t); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
		}
	} else if len(ib.Sections) > 0 {
		for _, s := range ib.Sections {
			if err := walk(nil, 0, s.Sections, s.Paragraphs, s.Title); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
		}
	} else if len(ib.Paragraphs) > 0 {
		if err := walk(nil, 0, nil, ib.Paragraphs, ib.Title); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	} else {
		http.Error(w, "no content (chapters/sections/paragraphs all empty)", 400)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	WriteJSON(w, 200, map[string]any{"id": bookID, "paragraphs": globalIdx})
}

type bookSummary struct {
	ID         int64  `json:"id"`
	Title      string `json:"title"`
	Author     string `json:"author"`
	Paragraphs int    `json:"paragraphs"`
	Progress   *int   `json:"progress,omitempty"`
}

func ListBooks(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT b.id, b.title, COALESCE(b.author,''),
		       (SELECT COUNT(*) FROM paragraphs p WHERE p.book_id=b.id),
		       (SELECT pr.last_paragraph_id FROM progress pr WHERE pr.book_id=b.id)
		FROM books b ORDER BY b.created_at DESC`)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	list := []bookSummary{}
	for rows.Next() {
		var s bookSummary
		var last sql.NullInt64
		if err := rows.Scan(&s.ID, &s.Title, &s.Author, &s.Paragraphs, &last); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		if last.Valid {
			row := db.QueryRow(`SELECT global_index FROM paragraphs WHERE id=?`, last.Int64)
			var gi int
			if err := row.Scan(&gi); err == nil {
				s.Progress = &gi
			}
		}
		list = append(list, s)
	}
	WriteJSON(w, 200, list)
}

func DeleteBook(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", 400)
		return
	}
	_, err = db.Exec(`DELETE FROM books WHERE id=?`, id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	WriteJSON(w, 200, map[string]any{"ok": true})
}

type sectionNode struct {
	ID       int64          `json:"id"`
	Title    string         `json:"title"`
	Depth    int            `json:"depth"`
	Children []*sectionNode `json:"children,omitempty"`
	ParaFrom int            `json:"para_from,omitempty"`
	ParaTo   int            `json:"para_to,omitempty"`
}

func GetBook(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", 400)
		return
	}
	row := db.QueryRow(`SELECT title, COALESCE(author,'') FROM books WHERE id=?`, id)
	var title, author string
	if err := row.Scan(&title, &author); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "not found", 404)
			return
		}
		http.Error(w, err.Error(), 500)
		return
	}
	total := 0
	_ = db.QueryRow(`SELECT COUNT(*) FROM paragraphs WHERE book_id=?`, id).Scan(&total)

	srows, err := db.Query(`SELECT id, COALESCE(parent_id,0), order_index, depth, title FROM sections WHERE book_id=? ORDER BY order_index`, id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer srows.Close()
	nodes := map[int64]*sectionNode{}
	var roots []*sectionNode
	// build & link in a single pass; query is already ORDER BY order_index so
	// appending preserves the source structure.
	for srows.Next() {
		var sid, parent int64
		var order, depth int
		var stitle string
		if err := srows.Scan(&sid, &parent, &order, &depth, &stitle); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		n := &sectionNode{ID: sid, Title: stitle, Depth: depth}
		nodes[sid] = n
		if parent == 0 {
			roots = append(roots, n)
		} else if p, ok := nodes[parent]; ok {
			p.Children = append(p.Children, n)
		}
	}
	// para_from/to per section (descendants)
	prows, err := db.Query(`SELECT section_id, MIN(global_index), MAX(global_index) FROM paragraphs WHERE book_id=? GROUP BY section_id`, id)
	if err == nil {
		for prows.Next() {
			var sid int64
			var lo, hi int
			_ = prows.Scan(&sid, &lo, &hi)
			if n, ok := nodes[sid]; ok {
				n.ParaFrom = lo
				n.ParaTo = hi
			}
		}
		prows.Close()
	}
	// propagate child ranges upward
	var fillRange func(n *sectionNode)
	fillRange = func(n *sectionNode) {
		for _, c := range n.Children {
			fillRange(c)
			if c.ParaFrom > 0 && (n.ParaFrom == 0 || c.ParaFrom < n.ParaFrom) {
				n.ParaFrom = c.ParaFrom
			}
			if c.ParaTo > n.ParaTo {
				n.ParaTo = c.ParaTo
			}
		}
	}
	for _, r := range roots {
		fillRange(r)
	}
	WriteJSON(w, 200, map[string]any{
		"id":         id,
		"title":      title,
		"author":     author,
		"paragraphs": total,
		"sections":   roots,
	})
}

type paragraphDTO struct {
	ID            int64  `json:"id"`
	GlobalIndex   int    `json:"global_index"`
	SectionID     int64  `json:"section_id"`
	SectionTitle  string `json:"section_title"`
	OriginalText  string `json:"original_text"`
}

func GetParagraphs(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	bookID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", 400)
		return
	}
	from, _ := strconv.Atoi(r.URL.Query().Get("from"))
	to, _ := strconv.Atoi(r.URL.Query().Get("to"))
	if from <= 0 {
		from = 1
	}
	if to <= 0 || to-from > 50 {
		to = from + 19
	}
	rows, err := db.Query(`
	  SELECT p.id, p.global_index, p.section_id, s.title, p.original_text
	  FROM paragraphs p JOIN sections s ON s.id=p.section_id
	  WHERE p.book_id=? AND p.global_index BETWEEN ? AND ?
	  ORDER BY p.global_index`, bookID, from, to)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	out := []paragraphDTO{}
	for rows.Next() {
		var p paragraphDTO
		if err := rows.Scan(&p.ID, &p.GlobalIndex, &p.SectionID, &p.SectionTitle, &p.OriginalText); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		out = append(out, p)
	}
	WriteJSON(w, 200, out)
}
