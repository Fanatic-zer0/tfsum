package tempdir

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// SignalContext returns a context that is cancelled on SIGINT or SIGTERM,
// plus a stop function that must be deferred to release the signal
// handler.
//
// Cancelling the context (rather than letting the default OS signal
// disposition kill the process) lets in-flight work started with
// exec.CommandContext unwind through normal Go defers, so the temp
// directory is still cleaned up instead of being skipped by an abrupt
// process kill.
func SignalContext(parent context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
}
