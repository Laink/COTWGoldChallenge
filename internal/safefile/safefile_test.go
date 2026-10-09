package safefile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWrite(t *testing.T) {
	p := filepath.Join(t.TempDir(), "f.txt")
	for _, s := range []string{"first", "second, longer"} {
		if err := Write(p, []byte(s)); err != nil {
			t.Fatal(err)
		}
		if b, _ := os.ReadFile(p); string(b) != s {
			t.Fatalf("read %q, want %q", b, s)
		}
	}
	if _, err := os.Stat(p + ".tmp"); !os.IsNotExist(err) {
		t.Error("temporary file left")
	}
}
