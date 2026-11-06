package attempts

import (
	"errors"
	"math"
)

type SlideTiming struct {
	SlideIndex int     `json:"slideIndex"`
	Start      float64 `json:"start"`
	End        float64 `json:"end"`
}
type Draft struct {
	Duration  float64       `json:"duration"`
	Thumbnail string        `json:"thumbnail"`
	ProjectID int64         `json:"projectId"`
	VersionID int64         `json:"versionId"`
	Topic     string        `json:"topic"`
	Timings   []SlideTiming `json:"timings"`
}

func (d Draft) validate() error {
	if math.IsNaN(d.Duration) || math.IsInf(d.Duration, 0) || d.Duration < 0 || len(d.Topic) > 10000 || len(d.Timings) > 10000 {
		return errors.New("invalidAttempt")
	}
	previous := 0.0
	for _, t := range d.Timings {
		if t.SlideIndex < 0 || math.IsNaN(t.Start) || math.IsNaN(t.End) || math.IsInf(t.Start, 0) || math.IsInf(t.End, 0) || t.Start < previous || t.End < t.Start || t.End > d.Duration+.001 {
			return errors.New("invalidTimeline")
		}
		previous = t.End
	}
	if d.ProjectID == 0 && (d.VersionID != 0 || len(d.Timings) > 0) {
		return errors.New("invalidVersion")
	}
	return nil
}

func (s *Store) Timings(id int64) ([]SlideTiming, error) {
	rows, e := s.db.Query("SELECT slide_index,start_sec,end_sec FROM slide_timings WHERE attempt_id=? ORDER BY start_sec", id)
	if e != nil {
		return nil, e
	}
	defer func() { _ = rows.Close() }()
	items := []SlideTiming{}
	for rows.Next() {
		var t SlideTiming
		if e := rows.Scan(&t.SlideIndex, &t.Start, &t.End); e != nil {
			return nil, e
		}
		items = append(items, t)
	}
	return items, rows.Err()
}
