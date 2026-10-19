package tunnel

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// signalContext ends on SIGINT or SIGTERM, so run stops cloudflared and
// removes the credential file before it exits.
func signalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}
