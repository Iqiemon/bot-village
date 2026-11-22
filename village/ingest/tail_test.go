package ingest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTailer(t *testing.T, lines *[]Line) (*Tailer, string) {
	t.Helper()
	p := filepath.Join(t.TempDir(), "s.jsonl")
	if err := os.WriteFile(p, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	tr, err := NewTailer(p, func(l Line) { *lines = append(*lines, l) })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { tr.Close() })
	return tr, p
}

func appendFile(t *testing.T, p, body string) {
	t.Helper()
	f, err := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(body); err != nil {
		t.Fatal(err)
	}
	f.Close()
}

func TestStartsAtEOF(t *testing.T) {
	p := filepath.Join(t.TempDir(), "s.jsonl")
	os.WriteFile(p, []byte(`{"type":"user"}`+"\n"), 0o644)
	var lines []Line
	tr, err := NewTailer(p, func(l Line) { lines = append(lines, l) })
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Close()
	if n, err := tr.Read(); err != nil || n != 0 || len(lines) != 0 {
		t.Fatalf("n=%d err=%v lines=%d", n, err, len(lines))
	}
}

func TestPartialTrailingLineIsNotEmitted(t *testing.T) {
	var lines []Line
	tr, p := newTailer(t, &lines)
	appendFile(t, p, `{"type":"user"}`+"\n"+`{"type":"assis`)
	n, err := tr.Read()
	if err != nil {
		t.Fatal(err)
	}
	if n != len(`{"type":"user"}`+"\n") {
		t.Fatalf("partial line advanced the offset: n=%d", n)
	}
	if len(lines) != 1 {
		t.Fatalf("got %d lines", len(lines))
	}
}

func TestCompletingTheLineEmitsItOnce(t *testing.T) {
	var lines []Line
	tr, p := newTailer(t, &lines)
	appendFile(t, p, `{"type":"user"}`+"\n"+`{"type":"assis`)
	tr.Read()
	appendFile(t, p, `tant"}`+"\n")
	if _, err := tr.Read(); err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 || lines[1].Kind != "message" {
		t.Fatalf("%+v", lines)
	}
}

func TestTruncationResetsOffset(t *testing.T) {
	var lines []Line
	tr, p := newTailer(t, &lines)
	appendFile(t, p, `{"type":"user"}`+"\n"+`{"type":"assistant"}`+"\n"+`{"type":"user"}`+"\n")
	if _, err := tr.Read(); err != nil {
		t.Fatal(err)
	}
	if len(lines) != 3 {
		t.Fatalf("got %d lines", len(lines))
	}
	if err := os.Truncate(p, 0); err != nil {
		t.Fatal(err)
	}
	appendFile(t, p, `{"type":"assistant"}`+"\n")
	if _, err := tr.Read(); err != nil {
		t.Fatal(err)
	}
	if len(lines) != 4 || lines[3].Kind != "message" {
		t.Fatalf("offset did not reset: %+v", lines)
	}
}

func TestReplacingTranscriptWithLargerFileResetsOffset(t *testing.T) {
	var lines []Line
	tr, p := newTailer(t, &lines)
	appendFile(t, p, `{"type":"user"}`+"\n"+`{"type":"assistant"}`+"\n")
	if _, err := tr.Read(); err != nil {
		t.Fatal(err)
	}
	body := `{"type":"assistant"}` + "\n" + `{"type":"user"}` + "\n" + `{"type":"assistant"}` + "\n"
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(tmp, p); err != nil {
		t.Fatal(err)
	}
	n, err := tr.Read()
	if err != nil {
		t.Fatal(err)
	}
	if n != len(body) || len(lines) != 5 {
		t.Fatalf("replaced transcript skipped: n=%d lines=%+v", n, lines)
	}
}

func TestUnknownJSONIsKindUnknown(t *testing.T) {
	var lines []Line
	tr, p := newTailer(t, &lines)
	appendFile(t, p, `{"nope":1}`+"\n")
	if _, err := tr.Read(); err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 || lines[0].Kind != "unknown" {
		t.Fatalf("%+v", lines)
	}
}

func TestLongLineIsTruncatedNotFatal(t *testing.T) {
	var lines []Line
	tr, p := newTailer(t, &lines)
	body := `{"type":"assistant","message":{"role":"assistant","content":"` + strings.Repeat("a", 2<<20) + `"}}` + "\n"
	appendFile(t, p, body)
	n, err := tr.Read()
	if err != nil {
		t.Fatal(err)
	}
	if n != len(body) || len(lines) != 1 || lines[0].Kind != "unknown" {
		t.Fatalf("n=%d lines=%+v", n, lines)
	}
	if n, err := tr.Read(); err != nil || n != 0 || len(lines) != 1 {
		t.Fatalf("second Read n=%d err=%v lines=%+v", n, err, lines)
	}
}

func TestValidLineSpanningChunksIsConsumedOnce(t *testing.T) {
	var lines []Line
	tr, p := newTailer(t, &lines)
	body := `{"type":"assistant","message":{"role":"assistant","content":"` + strings.Repeat("a", 200<<10) + `"}}` + "\n"
	appendFile(t, p, body)
	n, err := tr.Read()
	if err != nil {
		t.Fatal(err)
	}
	if n != len(body) || len(lines) != 1 || lines[0].Kind != "message" {
		t.Fatalf("n=%d lines=%+v", n, lines)
	}
	if n, err := tr.Read(); err != nil || n != 0 || len(lines) != 1 {
		t.Fatalf("second Read n=%d err=%v lines=%+v", n, err, lines)
	}
}

func TestStringContentUserLineIsMessage(t *testing.T) {
	var lines []Line
	tr, p := newTailer(t, &lines)
	appendFile(t, p, `{"type":"user","message":{"role":"user","content":"hello"}}`+"\n")
	if _, err := tr.Read(); err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 || lines[0].Kind != "message" || lines[0].Role != "user" {
		t.Fatalf("%+v", lines)
	}
}

func TestToolLineCarriesToolAndRole(t *testing.T) {
	var lines []Line
	tr, p := newTailer(t, &lines)
	appendFile(t, p, `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"hi"},{"type":"tool_use","name":"Bash"}]}}`+"\n")
	if _, err := tr.Read(); err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 || lines[0].Kind != "tool" || lines[0].ToolName != "Bash" || lines[0].Role != "assistant" {
		t.Fatalf("%+v", lines)
	}
}
