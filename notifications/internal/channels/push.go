package channels

import (
	"context"
	"fmt"
	"log"
)

// pushDispatcher is a stub. Replace the Send body with your FCM/APNs implementation.
type pushDispatcher struct{}

func NewPushDispatcher() Dispatcher {
	return &pushDispatcher{}
}

func (p *pushDispatcher) Send(_ context.Context, to, title, body string) error {
	// TODO: integrate FCM — POST to https://fcm.googleapis.com/v1/projects/{id}/messages:send
	// with the device push token in `to`, title in `title`, body in `body`.
	log.Printf("[PUSH STUB] to=%s title=%q body=%q", to, title, body)
	if to == "" {
		return fmt.Errorf("push token is empty")
	}
	return nil
}
