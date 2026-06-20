package templates

import "strings"

// Template holds per-channel content for a notification event.
// Variables use {{key}} placeholders replaced at render time.
type Template struct {
	DefaultChannels []string
	EmailSubject    string
	EmailBody       string
	SMSBody         string
	PushTitle       string
	PushBody        string
	InAppTitle      string
	InAppBody       string
}

func (t *Template) Render(vars map[string]string) *RenderedTemplate {
	pairs := make([]string, 0, len(vars)*2)
	for k, v := range vars {
		pairs = append(pairs, "{{"+k+"}}", v)
	}
	r := strings.NewReplacer(pairs...)
	return &RenderedTemplate{
		DefaultChannels: t.DefaultChannels,
		EmailSubject:    r.Replace(t.EmailSubject),
		EmailBody:       r.Replace(t.EmailBody),
		SMSBody:         r.Replace(t.SMSBody),
		PushTitle:       r.Replace(t.PushTitle),
		PushBody:        r.Replace(t.PushBody),
		InAppTitle:      r.Replace(t.InAppTitle),
		InAppBody:       r.Replace(t.InAppBody),
	}
}

type RenderedTemplate struct {
	DefaultChannels []string
	EmailSubject    string
	EmailBody       string
	SMSBody         string
	PushTitle       string
	PushBody        string
	InAppTitle      string
	InAppBody       string
}

// registry is the built-in template store.
// Add new entries here or replace with a DB-backed store in production.
var registry = map[string]*Template{
	"order_confirmed": {
		DefaultChannels: []string{"CHANNEL_IN_APP", "CHANNEL_EMAIL", "CHANNEL_PUSH"},
		EmailSubject:    "Your order #{{order_id}} is confirmed",
		EmailBody:       "Hi {{name}}, your order #{{order_id}} totalling {{total}} has been confirmed and is being prepared.",
		SMSBody:         "StockStar: Order #{{order_id}} confirmed. Total: {{total}}.",
		PushTitle:       "Order Confirmed",
		PushBody:        "Order #{{order_id}} ({{total}}) is confirmed!",
		InAppTitle:      "Order Confirmed",
		InAppBody:       "Your order #{{order_id}} totalling {{total}} has been confirmed.",
	},
	"order_delivered": {
		DefaultChannels: []string{"CHANNEL_IN_APP", "CHANNEL_EMAIL", "CHANNEL_PUSH", "CHANNEL_SMS"},
		EmailSubject:    "Your order #{{order_id}} has been delivered",
		EmailBody:       "Hi {{name}}, your order #{{order_id}} has been delivered. Enjoy!",
		SMSBody:         "StockStar: Order #{{order_id}} delivered.",
		PushTitle:       "Order Delivered",
		PushBody:        "Order #{{order_id}} has been delivered!",
		InAppTitle:      "Order Delivered",
		InAppBody:       "Your order #{{order_id}} has been delivered.",
	},
	"payment_received": {
		DefaultChannels: []string{"CHANNEL_IN_APP", "CHANNEL_EMAIL"},
		EmailSubject:    "Payment of {{amount}} received",
		EmailBody:       "Hi {{name}}, we received your payment of {{amount}} for order #{{order_id}}.",
		SMSBody:         "StockStar: Payment of {{amount}} received for order #{{order_id}}.",
		PushTitle:       "Payment Received",
		PushBody:        "Payment of {{amount}} confirmed.",
		InAppTitle:      "Payment Received",
		InAppBody:       "Payment of {{amount}} received for order #{{order_id}}.",
	},
	"account_verified": {
		DefaultChannels: []string{"CHANNEL_IN_APP", "CHANNEL_EMAIL"},
		EmailSubject:    "Your StockStar account is verified",
		EmailBody:       "Hi {{name}}, your account has been verified. Welcome aboard!",
		SMSBody:         "StockStar: Your account is now verified.",
		PushTitle:       "Account Verified",
		PushBody:        "Your account is now verified!",
		InAppTitle:      "Account Verified",
		InAppBody:       "Welcome! Your account has been successfully verified.",
	},
	"promo_offer": {
		DefaultChannels: []string{"CHANNEL_IN_APP", "CHANNEL_PUSH"},
		EmailSubject:    "{{title}} — exclusive offer for you",
		EmailBody:       "Hi {{name}}, {{body}}",
		SMSBody:         "StockStar: {{body}}",
		PushTitle:       "{{title}}",
		PushBody:        "{{body}}",
		InAppTitle:      "{{title}}",
		InAppBody:       "{{body}}",
	},
}

// Get returns the template for the given ID, or nil if not found.
func Get(id string) *Template {
	return registry[id]
}
