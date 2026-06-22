package utils

// SMSService defines the interface for sending SMS messages.
// Currently implemented with Twilio, but can be swapped for any provider
// (Africa's Talking, Vonage, etc.) by implementing this interface.
type SMSService interface {
	SendVerificationCode(phone string, code string) error
}
