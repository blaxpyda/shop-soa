package utils

// SMSService defines the interface for sending SMS messages.
// Currently implemented with Twilio, but can be swapped for any provider
// (Africa's Talking, Vonage, etc.) by implementing this interface.
type SMSService interface {
	SendVerificationCode(phone string, code string) error
}

// type twilioSMSService struct {
// 	accountSID string
// 	authToken  string
// 	fromNumber string
// }

// func NewSMSService() SMSService {
// 	return &twilioSMSService{
// 		accountSID: os.Getenv("TWILIO_ACCOUNT_SID"),
// 		authToken:  os.Getenv("TWILIO_AUTH_TOKEN"),
// 		fromNumber: os.Getenv("TWILIO_FROM_NUMBER"),
// 	}
// }

// func (t *twilioSMSService) SendVerificationCode(to string, code string) error {
// 	if t.accountSID == "" || t.authToken == "" || t.fromNumber == "" {
// 		return fmt.Errorf("twilio credentials not configured: set TWILIO_ACCOUNT_SID, TWILIO_AUTH_TOKEN, TWILIO_FROM_NUMBER")
// 	}

// 	apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", t.accountSID)

// 	body := fmt.Sprintf("Your StockStar verification code is: %s. It expires in 10 minutes.", code)

// 	data := url.Values{}
// 	data.Set("To", to)
// 	data.Set("From", t.fromNumber)
// 	data.Set("Body", body)

// 	req, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
// 	if err != nil {
// 		return fmt.Errorf("failed to create SMS request: %v", err)
// 	}

// 	req.SetBasicAuth(t.accountSID, t.authToken)
// 	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

// 	client := &http.Client{}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return fmt.Errorf("failed to send SMS: %v", err)
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
// 		var twilioErr map[string]interface{}
// 		json.NewDecoder(resp.Body).Decode(&twilioErr)
// 		return fmt.Errorf("twilio API error (status %d): %v", resp.StatusCode, twilioErr["message"])
// 	}

// 	return nil
// }
