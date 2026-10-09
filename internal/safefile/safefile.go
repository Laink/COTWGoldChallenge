// Package safefile writes the files that the game reads while the program runs: a whole new file
// replaces the old one, so that the game never reads it half written.
package safefile

import (
	"os"
	"time"
)

// Write writes data to path through a temporary file renamed over it. On Windows the rename fails
// while another program has the file open (the game reading it): it is tried again for a moment,
// then the file is written in place.
func Write(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	for i := 0; i < 5; i++ {
		if os.Rename(tmp, path) == nil {
			return nil
		}
		time.Sleep(20 * time.Millisecond)
	}
	os.Remove(tmp)
	return os.WriteFile(path, data, 0o644)
}
