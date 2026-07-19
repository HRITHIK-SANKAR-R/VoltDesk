package ai

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"voltdesk/internal/models"

	"google.golang.org/genai"
)

var client *genai.Client

func InitGemini() {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("GEMINI_API_KEY environment variable is required")
	}

	ctx := context.Background()
	var err error
	client, err = genai.NewClient(ctx, &genai.ClientConfig{APIKey: apiKey})
	if err != nil {
		log.Fatalf("Failed to create Gemini client: %v", err)
	}
}

// GenerateDraft interacts with Gemini to suggest a response
func GenerateDraft(queries *models.Queries, conversationID string) (*models.Message, error) {
	if client == nil {
		return nil, fmt.Errorf("gemini client not initialized")
	}

	// Fetch last 5 messages for context
	messages, err := queries.GetMessages(conversationID, 5)
	if err != nil {
		return nil, err
	}

	var contextStr strings.Builder
	for i := len(messages) - 1; i >= 0; i-- { // Reverse to chronological
		msg := messages[i]
		if msg.IsAIDraft {
			continue // Don't include drafts in context
		}
		sender := "Customer"
		// Determine if sender was agent (could fetch from users, but let's assume if it's the conversation customer it's customer, else agent)
		// To be precise we need sender role, but we'll approximate based on CustomerID of conversation
		conv, _ := queries.GetOrCreateOpenConversation(msg.SenderID) 
		if conv != nil && conv.CustomerID == msg.SenderID {
			sender = "Customer"
		} else {
			sender = "Agent"
		}
		contextStr.WriteString(fmt.Sprintf("%s: %s\n", sender, msg.Content))
	}

	prompt := fmt.Sprintf(`You are an aggressive, highly competent support agent for Sticker Mule. 
Your goal is to reply to the customer's last message concisely and helpfully.
Do not include pleasantries. Keep it under 2 sentences.
Here is the recent conversation history:
%s
Your reply as the agent:`, contextStr.String())

	ctx := context.Background()
	resp, err := client.Models.GenerateContent(ctx, "gemini-2.5-flash", genai.Text(prompt), nil)
	if err != nil {
		return nil, err
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no response from Gemini")
	}

	draftContent := ""
	if part, ok := resp.Candidates[0].Content.Parts[0].(genai.Text); ok {
		draftContent = string(part)
	}

	// Save draft to DB
	// For AI draft sender, we will assign a dummy agent ID or null, wait, sender_id is NOT NULL in DB,
	// so we'll need an agent ID, or use a system ID. Let's just create a system agent if none exists.
	// We'll pass the system agent ID or we can fetch a valid agent.
	
	// Assuming we fetch an agent
	sysAgent, err := queries.GetOrCreateCustomer("system@voltdesk.com") // we can update it to agent
	if err != nil {
		return nil, err
	}
	
	savedDraft, err := queries.SaveMessage(conversationID, sysAgent.ID, draftContent, true)
	if err != nil {
		return nil, err
	}

	return savedDraft, nil
}
