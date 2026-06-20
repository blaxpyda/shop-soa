package channels

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type smsDispatcher struct {
	accountSID string
	authToken  string
	fromNumber string
}

func NewSMSDispatcher() Dispatcher {
	return &smsDispatcher{
		accountSID: os.Getenv("TWILIO_ACCOUNT_SID"),
		authToken:  os.Getenv("TWILIO_AUTH_TOKEN"),
		fromNumber: os.Getenv("TWILIO_FROM_NUMBER"),
	}
}

func (t *smsDispatcher) Send(_ context.Context, to, _, body string) error {
	if t.accountSID == "" || t.authToken == "" || t.fromNumber == "" {
		return fmt.Errorf("twilio credentials not configured")
	}

	apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", t.accountSID)

	data := url.Values{}
	data.Set("To", to)
	data.Set("From", t.fromNumber)
	data.Set("Body", body)

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create SMS request: %w", err)
	}
	req.SetBasicAuth(t.accountSID, t.authToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send SMS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var twilioErr map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&twilioErr)
		return fmt.Errorf("twilio error (status %d): %v", resp.StatusCode, twilioErr["message"])
	}
	return nil
}
