// Package topics stores original practice prompts and persistent draw history.
package topics

import (
	"crypto/rand"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
	"sync"
)

//go:embed builtin.json
var builtin []byte

type Set struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Language string `json:"language"`
	Builtin  bool   `json:"builtin"`
	Count    int    `json:"count"`
}
type Topic struct {
	ID   int64  `json:"id"`
	Text string `json:"text"`
}
type Store struct {
	mu sync.Mutex
	db *sql.DB
}

func NewStore(db *sql.DB) *Store { return &Store{db: db} }

func (s *Store) Seed() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var sets map[string][]string
	if e := json.Unmarshal(builtin, &sets); e != nil {
		return e
	}
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer func() { _ = tx.Rollback() }()
	for _, language := range []string{"ru", "en"} {
		key := "builtin." + language
		if _, e := tx.Exec("INSERT OR IGNORE INTO topic_sets(set_key,name,language,builtin) VALUES(?,?,?,1)", key, key, language); e != nil {
			return e
		}
		var id int64
		if e := tx.QueryRow("SELECT id FROM topic_sets WHERE set_key=?", key).Scan(&id); e != nil {
			return e
		}
		for _, text := range sets[language] {
			if _, e := tx.Exec("INSERT OR IGNORE INTO topics(set_id,body) VALUES(?,?)", id, text); e != nil {
				return e
			}
		}
	}
	return tx.Commit()
}
func (s *Store) Sets() ([]Set, error) {
	rows, e := s.db.Query("SELECT s.id,s.name,s.language,s.builtin,COUNT(t.id) FROM topic_sets s LEFT JOIN topics t ON t.set_id=s.id GROUP BY s.id ORDER BY s.builtin DESC,s.id")
	if e != nil {
		return nil, e
	}
	defer func() { _ = rows.Close() }()
	items := []Set{}
	for rows.Next() {
		var set Set
		if e := rows.Scan(&set.ID, &set.Name, &set.Language, &set.Builtin, &set.Count); e != nil {
			return nil, e
		}
		items = append(items, set)
	}
	return items, rows.Err()
}
func (s *Store) Import(name string, contents []byte, csv bool) (Set, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	texts, e := Parse(contents, csv)
	if e != nil {
		return Set{}, e
	}
	if len(name) > 1000 {
		return Set{}, errors.New("invalidTopics")
	}
	tx, e := s.db.Begin()
	if e != nil {
		return Set{}, e
	}
	defer func() { _ = tx.Rollback() }()
	r, e := tx.Exec("INSERT INTO topic_sets(name) VALUES(?)", name)
	if e != nil {
		return Set{}, e
	}
	id, e := r.LastInsertId()
	if e != nil {
		return Set{}, e
	}
	for _, text := range texts {
		if _, e := tx.Exec("INSERT INTO topics(set_id,body) VALUES(?,?)", id, text); e != nil {
			return Set{}, e
		}
	}
	if e := tx.Commit(); e != nil {
		return Set{}, e
	}
	return Set{ID: id, Name: name, Count: len(texts)}, nil
}
func (s *Store) Draw(setIDs []int64, avoid int) (Topic, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(setIDs) == 0 || len(setIDs) > 100 || avoid < 0 || avoid > 1000 {
		return Topic{}, errors.New("invalidTopicSelection")
	}
	tx, e := s.db.Begin()
	if e != nil {
		return Topic{}, e
	}
	defer func() { _ = tx.Rollback() }()
	args := make([]any, len(setIDs))
	marks := make([]string, len(setIDs))
	for i, id := range setIDs {
		args[i] = id
		marks[i] = "?"
	}
	rows, e := tx.Query("SELECT MIN(id),body FROM topics WHERE set_id IN ("+strings.Join(marks, ",")+") GROUP BY body ORDER BY MIN(id) LIMIT 10001", args...)
	if e != nil {
		return Topic{}, e
	}
	pool := []Topic{}
	for rows.Next() {
		var t Topic
		if e := rows.Scan(&t.ID, &t.Text); e != nil {
			_ = rows.Close()
			return Topic{}, e
		}
		pool = append(pool, t)
	}
	e = errors.Join(rows.Err(), rows.Close())
	if e != nil {
		return Topic{}, e
	}
	if len(pool) == 0 || len(pool) > 10000 {
		return Topic{}, errors.New("topicsEmpty")
	}
	rows, e = tx.Query("SELECT t.body FROM recent_topics r JOIN topics t ON t.id=r.topic_id ORDER BY r.id DESC LIMIT ?", avoid)
	if e != nil {
		return Topic{}, e
	}
	recent := map[string]int{}
	index := 0
	for rows.Next() {
		var text string
		if e := rows.Scan(&text); e != nil {
			_ = rows.Close()
			return Topic{}, e
		}
		if _, ok := recent[text]; !ok {
			recent[text] = index
		}
		index++
	}
	e = errors.Join(rows.Err(), rows.Close())
	if e != nil {
		return Topic{}, e
	}
	eligible := []Topic{}
	for _, t := range pool {
		if _, ok := recent[t.Text]; !ok {
			eligible = append(eligible, t)
		}
	}
	if len(eligible) == 0 {
		oldest := -1
		for _, t := range pool {
			if recent[t.Text] > oldest {
				oldest = recent[t.Text]
				eligible = []Topic{t}
			}
		}
	}
	n, e := rand.Int(rand.Reader, big.NewInt(int64(len(eligible))))
	if e != nil {
		return Topic{}, e
	}
	selected := eligible[n.Int64()]
	if _, e := tx.Exec("INSERT INTO recent_topics(topic_id) VALUES(?)", selected.ID); e != nil {
		return Topic{}, e
	}
	if _, e := tx.Exec("DELETE FROM recent_topics WHERE id NOT IN (SELECT id FROM recent_topics ORDER BY id DESC LIMIT 1000)"); e != nil {
		return Topic{}, e
	}
	if e := tx.Commit(); e != nil {
		return Topic{}, e
	}
	return selected, nil
}
