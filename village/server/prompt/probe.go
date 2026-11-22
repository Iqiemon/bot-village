package prompt

import "context"

// Probe speaks one message to a session socket and reports what happened. Run
// it against a real messagingSocketPath to re-check this machine's behaviour; a
// nil error means the socket took the write, not that a session read it.
func Probe(socketPath string) error {
	return NewSocketTransport(socketPath).Send(context.Background(), "", "bot-village probe")
}
