// Package ingest reads Claude Code sessions and transcripts.
package ingest

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
)

// maxLine caps a transcript line; longer lines are emitted as unknown rather
// than buffered without limit. shortcut: whole line buffered in memory, revisit
// if transcripts ever carry multi-hundred-MB lines.
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
		t.offset = 0 // truncated or rewritten in place
	}
	if fi.Size() == t.offset {
		return 0, nil
	}
	if _, err := t.f.Seek(t.offset, io.SeekStart); err != nil {
		return 0, err
	}
	data, err := io.ReadAll(t.f)
	if err != nil {
		return 0, err
	}
	end := bytes.LastIndexByte(data, '\n')
	if end < 0 {
		return 0, nil
	}
	for _, raw := range bytes.Split(data[:end], []byte{'\n'}) {
		if len(raw) > 0 {
			t.onLine(parseLine(raw))
		}
	}
	t.offset += int64(end + 1)
	return end + 1, nil
}

func parseLine(raw []byte) Line {
	if len(raw) > maxLine {
		return Line{Kind: "unknown"}
	}
	var m struct {
		Type    string `json:"type"`
		Message *struct {
			Role    string `json:"role"`
			Content []struct {
				Type string `json:"type"`
				Name string `json:"name"`
			} `json:"content"`
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
	for _, p := range m.Message.Content {
		if p.Type == "tool_use" && p.Name != "" {
			l.Kind, l.ToolName = "tool", p.Name
			break
		}
	}
	return l
}
