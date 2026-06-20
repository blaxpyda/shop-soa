package channels

import "context"

// Dispatcher is the common interface every channel must satisfy.
type Dispatcher interface {
	// Send delivers a notification. Returns an error only on hard failure;
	// transient failures should be logged and retried by the caller.
	Send(ctx context.Context, to, subject, body string) error
}
