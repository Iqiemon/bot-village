// Package ingest reads Claude Code sessions and transcripts.
package ingest

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
)

// maxLine is the read buffer, so memory per line stays bounded; a longer line
// is dropped and emitted as unknown.
const maxLine = 1 << 20

// Line is one parsed transcript line.
type Line struct {
	Kind     string // "message", "tool" or "unknown"
	ToolName string
	Role     string
}

// Tailer reads a JSONL transcript incrementally from its current end. It emits
// only newline-terminated lines, so a partial trailing line is never emitted
// and never advances the offset.
type Tailer struct {
	f      *os.File
	offset int64
	onLine func(Line)
}

// StartOffset returns the current size of path so tailing begins at EOF.
func StartOffset(path string) (int64, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return fi.Size(), nil
}

// NewTailer opens path for tailing from EOF, calling onLine for each new line.
func NewTailer(path string, onLine func(Line)) (*Tailer, error) {
	off, err := StartOffset(path)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return &Tailer{f: f, offset: off, onLine: onLine}, nil
}

// Close releases the held file.
func (t *Tailer) Close() error { return t.f.Close() }

// Read emits every complete line appended since the last call and returns the
// number of bytes consumed. A partial trailing line is left for the next call.
func (t *Tailer) Read() (int, error) {
	fi, err := t.f.Stat()
	if err != nil {
		return 0, err
	}
	if fi.Size() < t.offset {
		t.offset = 0
	}
	if fi.Size() == t.offset {
		return 0, nil
	}
	if _, err := t.f.Seek(t.offset, io.SeekStart); err != nil {
		return 0, err
	}
	buf := make([]byte, 0, maxLine)
	consumed, long := 0, false
	chunk := make([]byte, 64<<10)
	for {
		n, err := t.f.Read(chunk)
		data := chunk[:n]
		for len(data) > 0 {
			i := bytes.IndexByte(data, '\n')
			if i < 0 {
				buf = append(buf, data...)
				if len(buf) > maxLine {
					buf, long = buf[:0], true
				}
				break
			}
			buf = append(buf, data[:i]...)
			if long || len(buf) > maxLine {
				t.onLine(Line{Kind: "unknown"})
			} else {
				t.onLine(parseLine(buf))
			}
			buf, long = buf[:0], false
			data = data[i+1:]
			consumed += i + 1
		}
		if err != nil {
			break
		}
	}
	t.offset += int64(consumed)
	return consumed, nil
}

func parseLine(raw []byte) Line {
	var m struct {
		Type    string `json:"type"`
		Message *struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
		} `json:"message"`
	}
	if json.Unmarshal(raw, &m) != nil || (m.Type != "user" && m.Type != "assistant") {
		return Line{Kind: "unknown"}
	}
	l := Line{Kind: "message", Role: m.Type}
	if m.Message == nil {
		return l
	}
	if m.Message.Role != "" {
		l.Role = m.Message.Role
	}
	var parts []struct {
		Type string `json:"type"`
		Name string `json:"name"`
	}
	if json.Unmarshal(m.Message.Content, &parts) != nil {
		return l // user turns carry content as a plain string
	}
	for _, p := range parts {
		if p.Type == "tool_use" && p.Name != "" {
			l.Kind, l.ToolName = "tool", p.Name
			break
		}
	}
	return l
}
