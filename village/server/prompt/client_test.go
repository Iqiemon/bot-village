package prompt

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"path/filepath"
	"testing"
	"time"
)

func TestSendMissingSocketIsUnreachable(t *testing.T) {
	tr := NewSocketTransport(filepath.Join(t.TempDir(), "absent.sock"))
	err := tr.Send(context.Background(), "abc", "hello")
	if !errors.Is(err, ErrUnreachable) {
		t.Fatalf("want ErrUnreachable, got %v", err)
	}
}

func TestSendRefusedSocketIsUnreachable(t *testing.T) {
	p := filepath.Join(t.TempDir(), "dead.sock")
	addr, err := net.ResolveUnixAddr("unix", p)
	if err != nil {
		t.Fatal(err)
	}
	l, err := net.ListenUnix("unix", addr)
	if err != nil {
		t.Fatal(err)
	}
	l.SetUnlinkOnClose(false) // keep the dead socket file, as a crashed session leaves it
	l.Close()

	if err := NewSocketTransport(p).Send(context.Background(), "abc", "hello"); !errors.Is(err, ErrUnreachable) {
		t.Fatalf("want ErrUnreachable, got %v", err)
	}
}

func TestSendTimeoutIsUnreachable(t *testing.T) {
	p := filepath.Join(t.TempDir(), "stall.sock")
	l, err := net.Listen("unix", p)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	// accept and never read, so a large write cannot fit in the socket buffer
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			defer c.Close()
		}
	}()

	tr := socketTransport{path: p, timeout: 20 * time.Millisecond}
	err = tr.Send(context.Background(), "abc", string(make([]byte, 8<<20)))
	if !errors.Is(err, ErrUnreachable) {
		t.Fatalf("want ErrUnreachable, got %v", err)
	}
}

func TestSendBlankTextIsRejected(t *testing.T) {
	tr := NewSocketTransport("")
	if err := tr.Send(context.Background(), "abc", "   "); err == nil || errors.Is(err, ErrUnreachable) {
		t.Fatalf("want a non-unreachable rejection, got %v", err)
	}
}

func TestSendWritesOneUserLine(t *testing.T) {
	p := filepath.Join(t.TempDir(), "inbox.sock")
	l, err := net.Listen("unix", p)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	got := make(chan []byte, 1)
	go func() {
		c, err := l.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		buf := make([]byte, 4096)
		n, _ := c.Read(buf)
		got <- buf[:n]
	}()

	if err := NewSocketTransport(p).Send(context.Background(), "abc", "hello"); err != nil {
		t.Fatal(err)
	}
	select {
	case line := <-got:
		var v struct {
			Type    string `json:"type"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
		}
		if err := json.Unmarshal(line, &v); err != nil {
			t.Fatalf("not one JSON line: %q", line)
		}
		if v.Type != "user" || v.Message.Role != "user" || v.Message.Content != "hello" {
			t.Fatalf("wrong payload: %+v", v)
		}
		if line[len(line)-1] != '\n' {
			t.Fatalf("line not terminated: %q", line)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("nothing written to the socket")
	}
}
