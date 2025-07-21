package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Helper function to load a test event from a file
func loadTestEvent(t *testing.T, fileName string) map[string]any {
	t.Helper()
	path := filepath.Join("testevents", fileName)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read test event file %s: %v", fileName, err)
	}

	var event map[string]any
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("Failed to unmarshal test event file %s: %v", fileName, err)
	}
	return event
}

func TestProcessDelivery(t *testing.T) {
	delivery := loadTestEvent(t, "delivery.json")

	processDelivery(delivery)

	recipients := delivery["recipients"].([]any)
	assert.Equal(t, "r********@e*********m", recipients[0].(string))
}

func TestProcessMail(t *testing.T) {
	mail := loadTestEvent(t, "mail.json")

	processMail(mail)

	// Check destination
	destination := mail["destination"].([]any)
	assert.Equal(t, "r********@e*********m", destination[0].(string))

	// Check commonHeaders
	commonHeaders := mail["commonHeaders"].(map[string]any)
	to := commonHeaders["to"].([]any)
	assert.Equal(t, "r********@e*********m", to[0].(string))
	assert.Equal(t, omittedSubject, commonHeaders["subject"].(string))

	// Check headers
	headers := mail["headers"].([]any)
	for _, headerItem := range headers {
		header := headerItem.(map[string]any)
		name := header["name"].(string)
		if name == "To" {
			assert.Equal(t, "r********@e*********m", header["value"].(string))
		}
		if name == "Subject" {
			assert.Equal(t, omittedSubject, header["value"].(string))
		}
	}
}

func TestProcessBounce(t *testing.T) {
	bounce := loadTestEvent(t, "bounce.json")

	processBounce(bounce)

	bouncedRecipients := bounce["bouncedRecipients"].([]any)
	recipient := bouncedRecipients[0].(map[string]any)
	assert.Equal(t, "b*****@e*********m", recipient["emailAddress"].(string))
}

func TestProcessComplaint(t *testing.T) {
	complaint := loadTestEvent(t, "complaint.json")

	processComplaint(complaint)

	complainedRecipients := complaint["complainedRecipients"].([]any)
	recipient := complainedRecipients[0].(map[string]any)
	assert.Equal(t, "c********@e*********m", recipient["emailAddress"].(string))
}

func TestProcessDeliveryDelay(t *testing.T) {
	deliveryDelay := loadTestEvent(t, "deliverydelay.json")

	processDeliveryDelay(deliveryDelay)

	delayedRecipients := deliveryDelay["delayedRecipients"].([]any)
	recipient := delayedRecipients[0].(map[string]any)
	assert.Equal(t, "d****@e*********m", recipient["emailAddress"].(string))
}

func TestMaskEmail(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single email",
			input:    "user1@example.com",
			expected: "u****@e*********m",
		},
		{
			name:     "name and email",
			input:    "User One <user1@example.com>",
			expected: "u****@e*********m",
		},
		{
			name:     "multiple emails",
			input:    "user1@example.com,user2@example.com",
			expected: "u****@e*********m,u****@e*********m",
		},
		{
			name:     "multiple names and emails",
			input:    "User One <user1@example.com>, user2@example.com",
			expected: "u****@e*********m,u****@e*********m",
		},
		{
			name:     "japanese name and email",
			input:    "日本語株式会社 <user1@example.com>",
			expected: "u****@e*********m",
		},
		{
			name:     "short domain",
			input:    "a@bc",
			expected: "a@bc",
		},
		{
			name:     "missing local part",
			input:    "@example.com",
			expected: "@example.com",
		},
		{
			name:     "missing domain",
			input:    "user1@",
			expected: "user1@",
		},
		{
			name:     "no at symbol",
			input:    "user1",
			expected: "user1",
		},
		{
			name:     "short domain no tld",
			input:    "a@b",
			expected: "a@b",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := maskEmail(tc.input)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
