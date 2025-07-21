package main

import (
	"context"
	"encoding/json"
	"log"
	"regexp"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

const omittedSubject string = "**Omitted**"

// maskEmail masks an email address according to the specified format.
func maskEmail(email string) string {
	emailRegex := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)

	// Find all email addresses in the input string
	emails := emailRegex.FindAllString(email, -1)
	if len(emails) == 0 {
		return email // Return original string if no email is found
	}

	var maskedEmails []string
	for _, e := range emails {
		parts := strings.Split(e, "@")
		localPart := parts[0]
		domainPart := parts[1]

		// Mask the local part, keeping the first character and replacing the rest with asterisks
		maskedLocal := ""
		if len(localPart) > 1 {
			maskedLocal = string(localPart[0]) + strings.Repeat("*", len(localPart)-1)
		} else {
			maskedLocal = localPart
		}

		// Mask the domain part, keeping the first and last characters and replacing the rest with asterisks
		maskedDomain := ""
		if len(domainPart) > 2 {
			maskedDomain = string(domainPart[0]) + strings.Repeat("*", len(domainPart)-2) + string(domainPart[len(domainPart)-1])
		} else {
			maskedDomain = domainPart
		}
		maskedEmails = append(maskedEmails, maskedLocal+"@"+maskedDomain)
	}

	return strings.Join(maskedEmails, ",")
}

func processDelivery(delivery map[string]any) {
	// Mask email addresses in the recipients array
	if recipients, ok := delivery["recipients"].([]any); ok {
		for i, addr := range recipients {
			if email, ok := addr.(string); ok {
				recipients[i] = maskEmail(email)
			}
		}
	}
}

func processMail(mail map[string]any) {
	// Mask email addresses in the destination array
	if destination, ok := mail["destination"].([]any); ok {
		for i, addr := range destination {
			if email, ok := addr.(string); ok {
				destination[i] = maskEmail(email)
			}
		}
	}

	// Mask email addresses and subject in commonHeaders
	if commonHeaders, ok := mail["commonHeaders"].(map[string]any); ok {
		if to, ok := commonHeaders["to"].([]any); ok {
			for i, addr := range to {
				if email, ok := addr.(string); ok {
					to[i] = maskEmail(email)
				}
			}
		}
		commonHeaders["subject"] = omittedSubject
	}

	// Mask "To" and "Subject" in the headers array
	if headers, ok := mail["headers"].([]any); ok {
		for _, headerItem := range headers {
			if header, ok := headerItem.(map[string]any); ok {
				if name, ok := header["name"].(string); ok {
					switch name {
					case "To":
						if value, ok := header["value"].(string); ok {
							header["value"] = maskEmail(value)
						}
					case "Subject":
						header["value"] = omittedSubject
					}
				}
			}
		}
	}
}

func processBounce(bounce map[string]any) {
	// Get the "bouncedRecipients" array
	if recipients, ok := bounce["bouncedRecipients"].([]any); ok {
		for _, recipientItem := range recipients {
			if recipient, ok := recipientItem.(map[string]any); ok {
				// Get and mask the value of the "emailAddress" field
				if email, ok := recipient["emailAddress"].(string); ok {
					recipient["emailAddress"] = maskEmail(email)
				}
			}
		}
	}
}

func processComplaint(complaint map[string]any) {
	// Get the "complainedRecipients" array
	if recipients, ok := complaint["complainedRecipients"].([]any); ok {
		for _, recipientItem := range recipients {
			if recipient, ok := recipientItem.(map[string]any); ok {
				// Get and mask the value of the "emailAddress" field
				if email, ok := recipient["emailAddress"].(string); ok {
					recipient["emailAddress"] = maskEmail(email)
				}
			}
		}
	}
}

func processDeliveryDelay(deliveryDelay map[string]any) {
	// Get the "delayedRecipients" array
	if recipients, ok := deliveryDelay["delayedRecipients"].([]any); ok {
		for _, recipientItem := range recipients {
			if recipient, ok := recipientItem.(map[string]any); ok {
				// Get and mask the value of the "emailAddress" field
				if email, ok := recipient["emailAddress"].(string); ok {
					recipient["emailAddress"] = maskEmail(email)
				}
			}
		}
	}
}

// processRecord processes a single Kinesis Firehose record.
func processRecord(record events.KinesisFirehoseEventRecord) (events.KinesisFirehoseResponseRecord, error) {
	var sesEvent map[string]any
	if err := json.Unmarshal(record.Data, &sesEvent); err != nil {
		log.Printf("Failed to unmarshal record data for record %s: %v", record.RecordID, err)
		return events.KinesisFirehoseResponseRecord{
			RecordID: record.RecordID,
			Result:   events.KinesisFirehoseTransformedStateProcessingFailed,
		}, err
	}

	// Check if "delivery" key exists and is a map
	if delivery, ok := sesEvent["delivery"].(map[string]any); ok {
		processDelivery(delivery)
	}

	// Check if "mail" key exists and is a map
	if mail, ok := sesEvent["mail"].(map[string]any); ok {
		processMail(mail)
	}

	// Check if "bounce" key exists and is a map
	if bounce, ok := sesEvent["bounce"].(map[string]any); ok {
		processBounce(bounce)
	}

	// Check if "complaint" key exists and is a map
	if complaint, ok := sesEvent["complaint"].(map[string]any); ok {
		processComplaint(complaint)
	}

	// Check if "deliveryDelay" key exists and is a map
	if deliveryDelay, ok := sesEvent["deliveryDelay"].(map[string]any); ok {
		processDeliveryDelay(deliveryDelay)
	}

	modifiedData, err := json.Marshal(sesEvent)
	if err != nil {
		log.Printf("Failed to marshal record data for record %s: %v", record.RecordID, err)
		return events.KinesisFirehoseResponseRecord{
			RecordID: record.RecordID,
			Result:   events.KinesisFirehoseTransformedStateProcessingFailed,
		}, err
	}

	return events.KinesisFirehoseResponseRecord{
		RecordID: record.RecordID,
		Result:   events.KinesisFirehoseTransformedStateOk,
		Data:     modifiedData,
	}, nil
}

// handler is the main Lambda function handler.
func handler(ctx context.Context, kinesisEvent events.KinesisFirehoseEvent) (events.KinesisFirehoseResponse, error) {
	var response events.KinesisFirehoseResponse

	for _, record := range kinesisEvent.Records {
		processedRecord, err := processRecord(record)
		if err != nil {
			processedRecord = events.KinesisFirehoseResponseRecord{
				RecordID: record.RecordID,
				Result:   events.KinesisFirehoseTransformedStateProcessingFailed,
			}
		}
		response.Records = append(response.Records, processedRecord)
	}

	return response, nil
}

func main() {
	lambda.Start(handler)
}
