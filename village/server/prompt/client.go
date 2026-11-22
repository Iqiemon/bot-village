// Package prompt delivers text to a running Claude Code session through the
// messaging socket that session advertises.
package prompt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// ErrUnreachable reports that the session could not be spoken to: its socket
// is missing, refused the connection, or did not accept the write in time.
var ErrUnreachable = errors.New("session unreachable")

// Transport sends a prompt to one running session.
type Transport interface {
	Send(ctx context.Context, sessionID, text string) error
}

// sendTimeout bounds connecting and writing. It cannot cover a reply: a live
// inbox never answers (see PROBE.md), so waiting would always time out.
const sendTimeout = 2 * time.Second

// NewSocketTransport returns a Transport that speaks to the session's
// messaging socket at socketPath.
func NewSocketTransport(socketPath string) Transport {
	return socketTransport{path: socketPath, timeout: sendTimeout}
}

type socketTransport struct {
	path    string
	timeout time.Duration
}

type userLine struct {
	Type    string `json:"type"`
	Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
}

func (t socketTransport) Send(ctx context.Context, sessionID, text string) error {
	if strings.TrimSpace(text) == "" {
		return errors.New("prompt: blank text")
	}
	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	var d net.Dialer
	conn, err := d.DialContext(ctx, "unix", t.path)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnreachable, err)
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetWriteDeadline(deadline)
	}

	msg := userLine{Type: "user"}
	msg.Message.Role = "user"
	msg.Message.Content = text
	line, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	// The inbox reads one line per message, so the appended newline ends it. It
	// never replies, so an accepted write is all a watcher can verify.
	// shortcut: the line is unsigned; a session whose inbox requires an auth
	// token drops it silently. Read the sibling key file's peerToken when that
	// shows up.
	if _, err := conn.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("%w: %v", ErrUnreachable, err)
	}
	return nil
}
