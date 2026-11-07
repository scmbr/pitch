package topics

import (
	"context"
	"github.com/scmbr/pitch/internal/storage"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func fixture(t *testing.T) *Store {
	t.Helper()
	db, e := storage.Open(context.Background(), filepath.Join(t.TempDir(), "topics.sqlite"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = db.Close() })
	return NewStore(db)
}
func TestBuiltinSeedIsIdempotentAndBilingual(t *testing.T) {
	s := fixture(t)
	if e := s.Seed(); e != nil {
		t.Fatal(e)
	}
	if e := s.Seed(); e != nil {
		t.Fatal(e)
	}
	sets, e := s.Sets()
	if e != nil || len(sets) != 2 {
		t.Fatalf("sets: %+v %v", sets, e)
	}
	for _, set := range sets {
		if !set.Builtin || set.Count != 30 || (set.Language != "ru" && set.Language != "en") {
			t.Fatalf("builtin: %+v", set)
		}
	}
}
func TestDrawWindowPersistsAndRelaxesOnlyWhenNecessary(t *testing.T) {
	s := fixture(t)
	set, e := s.Import("test", []byte("one\ntwo\nthree\n"), false)
	if e != nil {
		t.Fatal(e)
	}
	var history []string
	for i := 0; i < 12; i++ {
		// New service instances must still see the same persisted draw history.
		draw, e := NewStore(s.db).Draw([]int64{set.ID}, 2)
		if e != nil {
			t.Fatal(e)
		}
		for _, previous := range history[max(0, len(history)-2):] {
			if draw.Text == previous {
				t.Fatal("repeated within window")
			}
		}
		history = append(history, draw.Text)
	}
	last := history[len(history)-1]
	for i := 0; i < 6; i++ {
		draw, e := s.Draw([]int64{set.ID}, 1000)
		if e != nil {
			t.Fatal(e)
		}
		if draw.Text == last {
			t.Fatal("small pool immediately repeated a topic")
		}
		last = draw.Text
	}
	if _, e := s.Draw(nil, 2); e == nil {
		t.Fatal("accepted empty selection")
	}
	if _, e := s.Draw([]int64{set.ID}, -1); e == nil {
		t.Fatal("accepted invalid window")
	}
}
func TestSameTextAcrossSetsSharesRepeatHistory(t *testing.T) {
	s := fixture(t)
	a, e := s.Import("a", []byte("same"), false)
	if e != nil {
		t.Fatal(e)
	}
	b, e := s.Import("b", []byte("same\nother"), false)
	if e != nil {
		t.Fatal(e)
	}
	if _, e := s.Draw([]int64{a.ID}, 1); e != nil {
		t.Fatal(e)
	}
	draw, e := s.Draw([]int64{b.ID}, 1)
	if e != nil || draw.Text != "other" {
		t.Fatalf("repeat across sets: %+v %v", draw, e)
	}
}
func TestParseTXTAndCSV(t *testing.T) {
	for _, tt := range []struct {
		text string
		csv  bool
		want []string
	}{
		{"\ufeff  First \r\n\nSecond\nFirst\n", false, []string{"First", "Second"}},
		{"topic,tag\n\"A, B\",x\n\"C\nD\",y", true, []string{"A, B", "C\nD"}},
		{"Тема;тег\nОдин;а\nДва;б", true, []string{"Один", "Два"}},
		{"\"A; B; C\",tag\nNext,tag", true, []string{"A; B; C", "Next"}},
	} {
		got, e := Parse([]byte(tt.text), tt.csv)
		if e != nil || !reflect.DeepEqual(got, tt.want) {
			t.Fatalf("parse %q: %#v %v", tt.text, got, e)
		}
	}
	for _, text := range []string{"", string([]byte{255}), strings.Repeat("x", 2001), "zero\x00byte"} {
		if _, e := Parse([]byte(text), false); e == nil {
			t.Fatal("accepted invalid text")
		}
	}
	if _, e := Parse([]byte("\"unclosed"), true); e == nil {
		t.Fatal("accepted broken CSV")
	}
}
