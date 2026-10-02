package main

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"
)

const releasesURL = "https://github.com/Laink/COTWGoldChallenge/releases"

// latest receives the tag of a release newer than this program, "" otherwise.
var latest = make(chan string, 1)

// checkRelease asks GitHub for the latest release, in the background.
func checkRelease() {
	go func() {
		latest <- newerRelease()
	}()
}

func newerRelease() string {
	cur, ok := semver(version)
	if !ok {
		return "" // development build
	}
	c := http.Client{Timeout: 4 * time.Second}
	req, err := http.NewRequest("GET", "https://api.github.com/repos/Laink/COTWGoldChallenge/releases/latest", nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", "COTWGoldChallenge/"+version)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := c.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	var rel struct {
		Tag string `json:"tag_name"`
	}
	if resp.StatusCode != http.StatusOK || json.NewDecoder(resp.Body).Decode(&rel) != nil {
		return ""
	}
	if v, ok := semver(rel.Tag); ok && newer(v, cur) {
		return rel.Tag
	}
	return ""
}

var release struct {
	sync.Mutex
	tag string
}

// newRelease returns the newer release, "" when there is none or no answer yet. It never waits:
// a late answer shows at the next call.
func newRelease() string {
	release.Lock()
	defer release.Unlock()
	select {
	case release.tag = <-latest:
	default:
	}
	return release.tag
}

var semverRe = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)`)

// semver reads the version numbers of "v2.2.0" or "v2.2.0-12-gd6dc6a9" (a build after a tag).
func semver(s string) ([3]int, bool) {
	var v [3]int
	m := semverRe.FindStringSubmatch(s)
	if m == nil {
		return v, false
	}
	for i := range v {
		v[i], _ = strconv.Atoi(m[i+1])
	}
	return v, true
}

func newer(a, b [3]int) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] > b[i]
		}
	}
	return false
}
