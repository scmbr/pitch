package topics

import (
	"bytes"
	"encoding/csv"
	"errors"
	"io"
	"strings"
	"unicode/utf8"
)

// TXT is one topic per line. CSV takes the first column, with an optional
// topic/тема header; quoted commas/newlines and Excel's semicolon are supported.
func Parse(data []byte, isCSV bool) ([]string, error) {
	if len(data) > 2*1024*1024 || !utf8.Valid(data) {
		return nil, errors.New("invalidTopics")
	}
	data = bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})
	var candidates []string
	if isCSV {
		r := csv.NewReader(bytes.NewReader(data))
		r.FieldsPerRecord = -1
		commas, semicolons := separators(data)
		if semicolons > commas {
			r.Comma = ';'
		}
		for {
			row, e := r.Read()
			if errors.Is(e, io.EOF) {
				break
			}
			if e != nil {
				return nil, errors.New("invalidTopics")
			}
			if len(row) > 0 {
				candidates = append(candidates, row[0])
			}
		}
		if len(candidates) > 0 {
			header := strings.ToLower(strings.TrimSpace(candidates[0]))
			if header == "topic" || header == "тема" {
				candidates = candidates[1:]
			}
		}
	} else {
		candidates = strings.Split(string(data), "\n")
	}
	result := []string{}
	seen := map[string]bool{}
	for _, text := range candidates {
		text = strings.TrimSpace(text)
		if text == "" || seen[text] {
			continue
		}
		if utf8.RuneCountInString(text) > 2000 || strings.ContainsRune(text, 0) {
			return nil, errors.New("invalidTopics")
		}
		seen[text] = true
		result = append(result, text)
		if len(result) > 10000 {
			return nil, errors.New("invalidTopics")
		}
	}
	if len(result) == 0 {
		return nil, errors.New("topicsEmpty")
	}
	return result, nil
}

func separators(data []byte) (commas, semicolons int) {
	quoted := false
	for i := 0; i < len(data); i++ {
		c := data[i]
		if c == '"' {
			if quoted && i+1 < len(data) && data[i+1] == '"' {
				i++
				continue
			}
			quoted = !quoted
			continue
		}
		if quoted {
			continue
		}
		if c == '\n' {
			break
		}
		if c == ',' {
			commas++
		}
		if c == ';' {
			semicolons++
		}
	}
	return commas, semicolons
}
