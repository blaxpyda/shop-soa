package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type easySendSMS struct {
	apiKey string
	sender string
	http   *http.Client
}

func NewEasySendSMS(apiKey, sender string) SMSService {
	return &easySendSMS{
		apiKey: apiKey,
		sender: sender,
		http:   &http.Client{Timeout: 10 * time.Second},
	}
}

const sendURL = "https://restapi.easysendsms.app/v1/rest/sms/send"

type sendRequest struct {
	From string `json:"from"`
	To   string `json:"to"`
	Text string `json:"text"`
	Type string `json:"type"`
}

type sendResponse struct {
	Status      string   `json:"status"`
	Scheduled   string   `json:"scheduled"`
	MessageIDs  []string `json:"messageIds"`
	Error       string   `json:"error,omitempty"`
	Description string   `json:"description,omitempty"`
}

func (e *easySendSMS) SendVerificationCode(to string, code string) error {
	text := fmt.Sprintf("Your verification code is: %s. It expires in 10 minutes.", code)
	body, _ := json.Marshal(sendRequest{
		From: e.sender,
		To:   normaliseMSISDN(to),
		Text: text,
		Type: messageType(text),
	})

	req, err := http.NewRequestWithContext(context.Background(), "POST", sendURL, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := e.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var out sendResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return err
	}

	if out.Error != "" {
		return fmt.Errorf("easysend error: %s - %s", out.Error, out.Description)
	}

	if out.Status != "OK" || len(out.MessageIDs) == 0 {
		return fmt.Errorf("unexpected easysend response: %v", out)
	}

	first := out.MessageIDs[0]
	if strings.HasPrefix(first, "ERR") {
		return fmt.Errorf("easysend API error: %s", first)
	}

	return nil
}

func normaliseMSISDN(number string) string {
	phone := strings.TrimSpace(number)
	phone = strings.TrimPrefix(phone, "+")
	phone = strings.TrimPrefix(phone, "00")
	return strings.ReplaceAll(phone, " ", "")
}

func messageType(text string) string {
	for _, r := range text {
		if r > 127 {
			return "1"
		}
	}
	return "0"
}
