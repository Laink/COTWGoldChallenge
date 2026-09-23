// Package swf edits Scaleform (.gfx) movies and their ActionScript 3 bytecode.
package swf

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"io"
)

// Tag is a SWF tag. Header keeps the original encoding when the length is unchanged.
type Tag struct {
	Code   int
	Data   []byte
	Header []byte
}

// Movie is a decoded SWF/GFX file.
type Movie struct {
	Signature [3]byte // "CFX", "GFX", "CWS" or "FWS"
	Version   byte
	Head      []byte // frame size, rate and count
	Tags      []*Tag
}

// Compressed reports whether the body is zlib-compressed.
func (m *Movie) Compressed() bool { return m.Signature[0] == 'C' }

// Load decodes a movie. Bytes after the End tag are dropped.
func Load(b []byte) (*Movie, error) {
	if len(b) < 8 {
		return nil, errors.New("file too short")
	}
	m := &Movie{Version: b[3]}
	copy(m.Signature[:], b[:3])
	declared := int(binary.LittleEndian.Uint32(b[4:]))
	body := b[8:]
	if m.Compressed() {
		r, err := zlib.NewReader(bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		if body, err = io.ReadAll(r); err != nil && err != io.ErrUnexpectedEOF {
			return nil, err
		}
	}
	if declared-8 < len(body) && declared > 8 {
		body = body[:declared-8]
	}
	if len(body) < 1 {
		return nil, errors.New("empty movie")
	}
	nbits := int(body[0] >> 3)
	p := (5+4*nbits+7)/8 + 4
	m.Head = body[:p]
	for p+2 <= len(body) {
		cl := binary.LittleEndian.Uint16(body[p:])
		code, ln, h := int(cl>>6), int(cl&0x3f), 2
		if ln == 0x3f {
			ln, h = int(binary.LittleEndian.Uint32(body[p+2:])), 6
		}
		if p+h+ln > len(body) {
			return nil, errors.New("truncated tag")
		}
		m.Tags = append(m.Tags, &Tag{code, body[p+h : p+h+ln], body[p : p+h]})
		p += h + ln
		if code == 0 {
			break
		}
	}
	return m, nil
}

func (t *Tag) bytes() []byte {
	if t.Header != nil {
		cl := binary.LittleEndian.Uint16(t.Header)
		ln := int(cl & 0x3f)
		if ln == 0x3f {
			ln = int(binary.LittleEndian.Uint32(t.Header[2:]))
		}
		if ln == len(t.Data) {
			return append(append([]byte{}, t.Header...), t.Data...)
		}
	}
	h := make([]byte, 6)
	binary.LittleEndian.PutUint16(h, uint16(t.Code<<6|0x3f))
	binary.LittleEndian.PutUint32(h[2:], uint32(len(t.Data)))
	return append(h, t.Data...)
}

// Body returns the uncompressed movie body.
func (m *Movie) Body() []byte {
	var b bytes.Buffer
	b.Write(m.Head)
	for _, t := range m.Tags {
		b.Write(t.bytes())
	}
	return b.Bytes()
}

// Bytes encodes the movie.
func (m *Movie) Bytes() []byte {
	body := m.Body()
	out := bytes.NewBuffer(nil)
	out.Write(m.Signature[:])
	out.WriteByte(m.Version)
	binary.Write(out, binary.LittleEndian, uint32(8+len(body)))
	if m.Compressed() {
		w, _ := zlib.NewWriterLevel(out, zlib.BestCompression)
		w.Write(body)
		w.Close()
	} else {
		out.Write(body)
	}
	return out.Bytes()
}

// ABCTag returns the single DoABC tag and the offset of its bytecode.
func (m *Movie) ABCTag() (*Tag, int, error) {
	var found *Tag
	for _, t := range m.Tags {
		if t.Code == 82 {
			if found != nil {
				return nil, 0, errors.New("several ActionScript blocks")
			}
			found = t
		}
	}
	if found == nil {
		return nil, 0, errors.New("no ActionScript block")
	}
	p := 4
	for p < len(found.Data) && found.Data[p] != 0 {
		p++
	}
	return found, p + 1, nil
}
