// Package timer keeps the challenge timer: the play sessions counted (while the game is in front,
// not paused), and the time counted before them, entered by the player or estimated from the
// harvests. The program writes it in dropzone/ui/cotwgc_timer_data.txt.
package timer

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/Laink/COTWGoldChallenge/internal/safefile"
)

// DataFile is the state of the timer, relative to dropzone/ui.
const DataFile = "cotwgc_timer_data.txt"

// Session is a stretch of time counted, in Unix seconds. Estimated ones come from the harvests.
type Session struct {
	Start, End int64
	Estimated  bool
}

// Timer is the challenge timer, safe for concurrent use.
type Timer struct {
	mu       sync.Mutex
	path     string
	offset   float64 // seconds counted before the sessions
	sessions []Session
	paused   bool
	hidden   bool
	dirty    bool
}

// Load reads the timer of a dropzone/ui folder; a missing file is a timer at zero.
func Load(uiDir string) (*Timer, error) {
	t := &Timer{path: filepath.Join(uiDir, DataFile)}
	b, err := os.ReadFile(t.path)
	if errors.Is(err, os.ErrNotExist) {
		return t, nil
	}
	if err != nil {
		return nil, err
	}
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		num := func(i int) int64 { v, _ := strconv.ParseInt(f[i], 10, 64); return v }
		switch {
		case len(f) == 2 && f[0] == "o":
			t.offset, _ = strconv.ParseFloat(f[1], 64)
		case len(f) == 2 && f[0] == "p":
			t.paused = f[1] == "1"
		case len(f) == 2 && f[0] == "h":
			t.hidden = f[1] == "1"
		case len(f) == 3 && (f[0] == "s" || f[0] == "e") && num(2) >= num(1):
			t.sessions = append(t.sessions, Session{num(1), num(2), f[0] == "e"})
		}
	}
	sort.Slice(t.sessions, func(i, j int) bool { return t.sessions[i].Start < t.sessions[j].Start })
	return t, sc.Err()
}

// Tick counts the time up to now when counting is set: the last session goes on when it ended
// a moment ago, else a new one starts.
func (t *Timer) Tick(now int64, counting bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !counting || t.paused {
		return
	}
	if n := len(t.sessions); n > 0 && !t.sessions[n-1].Estimated && now-t.sessions[n-1].End <= 2 && now >= t.sessions[n-1].End {
		t.sessions[n-1].End = now
	} else {
		t.sessions = append(t.sessions, Session{Start: now, End: now})
	}
	t.dirty = true
}

// Elapsed returns the time counted, in seconds.
func (t *Timer) Elapsed() float64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.at(1 << 62)
}

// At returns the time counted up to a Unix time, in seconds.
func (t *Timer) At(unix int64) float64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.at(unix)
}

func (t *Timer) at(unix int64) float64 {
	v := t.offset
	for _, s := range t.sessions {
		if s.Start < unix {
			v += float64(min(s.End, unix) - s.Start)
		}
	}
	return max(v, 0)
}

// Sessions returns a copy of the sessions.
func (t *Timer) Sessions() []Session {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]Session(nil), t.sessions...)
}

// Paused tells whether the timer is paused; Hidden whether the player hid it in the game.
func (t *Timer) Paused() bool { t.mu.Lock(); defer t.mu.Unlock(); return t.paused }
func (t *Timer) Hidden() bool { t.mu.Lock(); defer t.mu.Unlock(); return t.hidden }

// TogglePause pauses or resumes the timer; ToggleHidden hides or shows it in the game.
func (t *Timer) TogglePause() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.paused = !t.paused
	return t.save()
}

func (t *Timer) ToggleHidden() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.hidden = !t.hidden
	return t.save()
}

// Set makes the time counted equal to seconds. More time is time played before the timer (the time
// already played of a challenge started without it); less time is taken off the last time counted
// (a test played off the challenge), so that the times of the harvests stay as they were.
func (t *Timer) Set(seconds float64) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	cut := t.at(1<<62) - seconds
	if cut <= 0 {
		t.offset -= cut
		return t.save()
	}
	for len(t.sessions) > 0 && cut > 0 {
		last := &t.sessions[len(t.sessions)-1]
		if d := float64(last.End - last.Start); d > cut {
			last.End -= int64(cut)
			cut = 0
		} else {
			t.sessions = t.sessions[:len(t.sessions)-1]
			cut -= d
		}
	}
	t.offset = max(t.offset-cut, 0)
	return t.save()
}

// Reset puts the timer back to zero.
func (t *Timer) Reset() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.offset, t.sessions, t.paused = 0, nil, false
	return t.save()
}

// Estimate replaces the time before the recorded sessions by sessions estimated from the times of
// the harvests (Unix seconds) since the challenge start: harvests less than an hour and a half
// apart are one session, which begins 20 minutes before its first harvest and ends 10 minutes
// after its last one.
func (t *Timer) Estimate(harvests []int64, since int64) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	var real []Session
	for _, s := range t.sessions {
		if !s.Estimated {
			real = append(real, s)
		}
	}
	first := int64(1 << 62)
	if len(real) > 0 {
		first = real[0].Start
	}
	var times []int64
	for _, h := range harvests {
		if h >= since && h < first {
			times = append(times, h)
		}
	}
	sort.Slice(times, func(i, j int) bool { return times[i] < times[j] })
	var est []Session
	for _, h := range times {
		if n := len(est); n > 0 && h-(est[n-1].End-600) <= 5400 {
			est[n-1].End = h + 600
			continue
		}
		est = append(est, Session{Start: max(h-1200, since), End: h + 600, Estimated: true})
	}
	// the estimated sessions stop where the recorded ones begin
	for i := range est {
		est[i].End = min(est[i].End, first)
		if i+1 < len(est) {
			est[i].End = min(est[i].End, est[i+1].Start)
		}
	}
	t.offset = 0
	t.sessions = append(est, real...)
	return t.save()
}

// Flush saves the timer when it changed since the last save.
func (t *Timer) Flush() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.dirty {
		return nil
	}
	return t.save()
}

func (t *Timer) save() error {
	t.dirty = true
	var b strings.Builder
	b.WriteString("# COTWGoldChallenge - challenge timer: o seconds counted before the sessions, p paused, h hidden,\n")
	b.WriteString("# s play session (start end, Unix seconds), e session estimated from the harvests\n")
	fmt.Fprintf(&b, "o %s\n", strconv.FormatFloat(t.offset, 'f', 0, 64))
	fmt.Fprintf(&b, "p %d\nh %d\n", bit(t.paused), bit(t.hidden))
	for _, s := range t.sessions {
		k := "s"
		if s.Estimated {
			k = "e"
		}
		fmt.Fprintf(&b, "%s %d %d\n", k, s.Start, s.End)
	}
	if err := safefile.Write(t.path, []byte(b.String())); err != nil {
		return err
	}
	t.dirty = false
	return nil
}

func bit(b bool) int {
	if b {
		return 1
	}
	return 0
}
