package worker

import (
	"fmt"
	"log"
	"os"
	"voltdesk/internal/models"

	"github.com/resend/resend-go/v2"
)

func CheckIdleConversations(queries *models.Queries) {
	idleConvs, err := queries.GetIdleConversations()
	if err != nil {
		log.Printf("Error checking idle conversations: %v", err)
		return
	}

	if len(idleConvs) == 0 {
		return
	}

	apiKey := os.Getenv("RESEND_API_KEY")
	if apiKey == "" {
		log.Println("RESEND_API_KEY not set, skipping idle notifications")
		return
	}
	client := resend.NewClient(apiKey)

	for _, ic := range idleConvs {
		params := &resend.SendEmailRequest{
			From:    "VoltDesk Support <support@voltdesk.com>",
			To:      []string{"agent@voltdesk.com"}, // In a real app, this would be dynamic agent email
			Subject: fmt.Sprintf("Action Required: Idle Chat %s", ic.ConversationID),
			Html:    fmt.Sprintf("<p>Customer %s has been waiting for more than 2 minutes. Please respond.</p>", ic.CustomerEmail),
		}

		_, err := client.Emails.Send(params)
		if err != nil {
			log.Printf("Failed to send idle alert for conversation %s: %v", ic.ConversationID, err)
		} else {
			log.Printf("Sent idle alert for conversation %s", ic.ConversationID)
		}
	}
}
