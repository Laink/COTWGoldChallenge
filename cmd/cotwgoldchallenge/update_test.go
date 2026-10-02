package main

import "testing"

func TestNewer(t *testing.T) {
	for _, c := range []struct {
		tag, cur string
		want     bool
	}{
		{"v2.3.0", "v2.2.0", true},
		{"v2.10.0", "v2.9.1", true},
		{"v2.2.0", "v2.2.0", false},
		{"v2.2.0", "v2.2.0-12-gd6dc6a9", false},
		{"v2.1.0", "v2.2.0", false},
		{"v3.0.0", "2.2.0", true},
	} {
		a, ok1 := semver(c.tag)
		b, ok2 := semver(c.cur)
		if !ok1 || !ok2 || newer(a, b) != c.want {
			t.Errorf("%s vs %s: want %v", c.tag, c.cur, c.want)
		}
	}
	if _, ok := semver("dev"); ok {
		t.Error("dev is a version")
	}
}
