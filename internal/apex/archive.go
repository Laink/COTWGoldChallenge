package apex

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Entry is a file stored in a .arc archive, indexed by a .tab file.
type Entry struct {
	Hash   uint32
	Offset uint32
	Size   uint32
	Arc    string
}

// Archives is the content of archives_win64.
type Archives struct {
	Dir     string
	Entries []Entry
	files   map[string]*os.File
}

// Open indexes every .tab file of an archives_win64 directory.
func Open(dir string) (*Archives, error) {
	tabs, err := filepath.Glob(filepath.Join(dir, "*.tab"))
	if err != nil {
		return nil, err
	}
	if len(tabs) == 0 {
		return nil, fmt.Errorf("no .tab file in %s", dir)
	}
	sort.Strings(tabs)
	a := &Archives{Dir: dir, files: map[string]*os.File{}}
	for _, t := range tabs {
		b, err := os.ReadFile(t)
		if err != nil {
			return nil, err
		}
		arc := strings.TrimSuffix(filepath.Base(t), ".tab") + ".arc"
		for p := 12; p+12 <= len(b); p += 12 {
			a.Entries = append(a.Entries, Entry{
				Hash:   binary.LittleEndian.Uint32(b[p:]),
				Offset: binary.LittleEndian.Uint32(b[p+4:]),
				Size:   binary.LittleEndian.Uint32(b[p+8:]),
				Arc:    arc,
			})
		}
	}
	return a, nil
}

// Close releases the open archive files.
func (a *Archives) Close() {
	for _, f := range a.files {
		f.Close()
	}
}

// ReadAt reads n bytes of an entry, starting at off.
func (a *Archives) ReadAt(e Entry, off, n int) ([]byte, error) {
	f := a.files[e.Arc]
	if f == nil {
		var err error
		if f, err = os.Open(filepath.Join(a.Dir, e.Arc)); err != nil {
			return nil, err
		}
		a.files[e.Arc] = f
	}
	if off+n > int(e.Size) {
		n = int(e.Size) - off
	}
	if n <= 0 {
		return nil, nil
	}
	buf := make([]byte, n)
	_, err := f.ReadAt(buf, int64(e.Offset)+int64(off))
	return buf, err
}

// Read returns the whole entry.
func (a *Archives) Read(e Entry) ([]byte, error) { return a.ReadAt(e, 0, int(e.Size)) }

// Find returns the entries stored under a path.
func (a *Archives) Find(path string) []Entry {
	h := HashString(path)
	var out []Entry
	for _, e := range a.Entries {
		if e.Hash == h {
			out = append(out, e)
		}
	}
	return out
}

// Inflate decompresses an AAF container ("AAF\0" + deflate blocks). If first is set,
// only the first block is decompressed.
func Inflate(b []byte, first bool) ([]byte, error) {
	if len(b) < 0x30 || !bytes.Equal(b[:4], []byte("AAF\x00")) {
		return nil, errors.New("not an AAF container")
	}
	n := int(binary.LittleEndian.Uint32(b[0x2c:]))
	pos := 0x30
	var out bytes.Buffer
	for i := 0; i < n; i++ {
		if pos+16 > len(b) {
			return nil, errors.New("truncated AAF")
		}
		comp := int(binary.LittleEndian.Uint32(b[pos:]))
		next := int(binary.LittleEndian.Uint32(b[pos+8:]))
		if string(b[pos+12:pos+16]) != "EWAM" || pos+16+comp > len(b) {
			return nil, errors.New("invalid AAF block")
		}
		r := flate.NewReader(bytes.NewReader(b[pos+16 : pos+16+comp]))
		_, err := io.Copy(&out, r)
		r.Close()
		if err != nil && err != io.ErrUnexpectedEOF {
			return nil, err
		}
		if first {
			break
		}
		pos += next
	}
	return out.Bytes(), nil
}

// SarcFile is a file inside a SARC container.
type SarcFile struct {
	Name         string
	Offset, Size int
}

// SarcList lists the files of a SARC container.
func SarcList(b []byte) ([]SarcFile, error) {
	if len(b) < 16 || string(b[4:8]) != "SARC" {
		return nil, errors.New("not a SARC container")
	}
	end := 16 + int(binary.LittleEndian.Uint32(b[12:]))
	if end > len(b) {
		return nil, errors.New("truncated SARC index")
	}
	var out []SarcFile
	for p := 16; p < end-8; {
		nl := int(binary.LittleEndian.Uint32(b[p:]))
		p += 4
		if p+nl+8 > len(b) {
			break
		}
		name := strings.TrimRight(string(b[p:p+nl]), "\x00")
		p += nl
		out = append(out, SarcFile{name, int(binary.LittleEndian.Uint32(b[p:])), int(binary.LittleEndian.Uint32(b[p+4:]))})
		p += 8
	}
	return out, nil
}

// FindInContainers looks for a file packed inside an AAF/SARC container.
func (a *Archives) FindInContainers(name string, progress func(done, total int)) ([]byte, error) {
	for i, e := range a.Entries {
		if progress != nil && i%2000 == 0 {
			progress(i, len(a.Entries))
		}
		if e.Size < 0x40 {
			continue
		}
		head, err := a.ReadAt(e, 0, 0x40)
		if err != nil || !bytes.Equal(head[:4], []byte("AAF\x00")) {
			continue
		}
		comp := int(binary.LittleEndian.Uint32(head[0x30:]))
		files, err := a.sarcIndex(e, comp)
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.Name != name {
				continue
			}
			all, err := a.Read(e)
			if err != nil {
				return nil, err
			}
			u, err := Inflate(all, false)
			if err != nil {
				return nil, err
			}
			if f.Offset+f.Size > len(u) {
				return nil, errors.New("truncated container")
			}
			return u[f.Offset : f.Offset+f.Size], nil
		}
	}
	return nil, fmt.Errorf("%s not found", name)
}

// sarcIndex decompresses only the SARC index at the start of an AAF container.
func (a *Archives) sarcIndex(e Entry, comp int) ([]SarcFile, error) {
	for _, n := range []int{64 << 10, comp} {
		blk, err := a.ReadAt(e, 0x40, min(n, comp))
		if err != nil {
			return nil, err
		}
		r := flate.NewReader(bytes.NewReader(blk))
		head := make([]byte, 16)
		if _, err := io.ReadFull(r, head); err != nil || string(head[4:8]) != "SARC" {
			r.Close()
			if err == nil || n >= comp {
				return nil, errors.New("not a SARC container")
			}
			continue
		}
		size := int(binary.LittleEndian.Uint32(head[12:]))
		if size > 64<<20 {
			r.Close()
			return nil, errors.New("invalid SARC index")
		}
		idx := make([]byte, size)
		_, err = io.ReadFull(r, idx)
		r.Close()
		if err != nil {
			if n >= comp {
				return nil, err
			}
			continue
		}
		return SarcList(append(head, idx...))
	}
	return nil, errors.New("not a SARC container")
}
