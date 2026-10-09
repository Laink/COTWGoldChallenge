package timer

import "testing"

func TestTimer(t *testing.T) {
	dir := t.TempDir()
	tm, _ := Load(dir)
	for s := int64(1000); s <= 1010; s++ {
		tm.Tick(s, true)
	}
	tm.Tick(1011, false) // game not in front
	for s := int64(2000); s <= 2005; s++ {
		tm.Tick(s, true)
	}
	if e := tm.Elapsed(); e != 15 {
		t.Fatalf("elapsed %v, want 15", e)
	}
	if a := tm.At(1005); a != 5 {
		t.Errorf("at 1005: %v", a)
	}
	tm.TogglePause()
	tm.Tick(2006, true)
	if tm.Elapsed() != 15 {
		t.Error("paused timer counted")
	}
	tm.TogglePause()
	// the time played before: 1 hour
	tm.Set(3600)
	if e := tm.Elapsed(); e != 3600 {
		t.Errorf("set: %v", e)
	}
	tm.Flush()
	tm, _ = Load(dir)
	if e, n := tm.Elapsed(), len(tm.Sessions()); e != 3600 || n != 2 {
		t.Errorf("read again: %v, %d sessions", e, n)
	}
	if tm.At(1500) != 3600-5 {
		t.Errorf("at 1500 after set: %v", tm.At(1500))
	}
}

func TestEstimate(t *testing.T) {
	tm, _ := Load(t.TempDir())
	tm.Tick(100000, true)
	tm.Tick(100001, true)
	// two evenings of harvests before the timer, and one before the challenge start
	harvests := []int64{10000, 11000, 12000, 50000, 51000, 99990, 500}
	if err := tm.Estimate(harvests, 5000); err != nil {
		t.Fatal(err)
	}
	s := tm.Sessions()
	if len(s) != 4 || !s[0].Estimated || s[3].Estimated {
		t.Fatalf("sessions %+v", s)
	}
	if s[0].Start != 10000-1200 || s[0].End != 12000+600 {
		t.Errorf("first session %+v", s[0])
	}
	if s[2].End > 100000 {
		t.Errorf("estimated session over the recorded one: %+v", s[2])
	}
}

// TestSetBack: a test played off the challenge is taken off the end, the times before stay.
func TestSetBack(t *testing.T) {
	tm, _ := Load(t.TempDir())
	tm.Set(3600) // played before the timer
	for s := int64(1000); s <= 1100; s++ {
		tm.Tick(s, true) // the challenge: 100 s
	}
	for s := int64(5000); s <= 5300; s++ {
		tm.Tick(s, true) // a test: 300 s
	}
	tm.Set(3700)
	if e, a := tm.Elapsed(), tm.At(1050); e != 3700 || a != 3650 {
		t.Errorf("elapsed %v, at 1050 %v", e, a)
	}
	if n := len(tm.Sessions()); n != 1 {
		t.Errorf("%d sessions, the test should be gone", n)
	}
}
