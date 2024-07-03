package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/go-pg/pg/v10"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
)




func (pm *PostgresMemory) InsertConversation(ctx context.Context, title string) (*Conversation, error) {
    conversation := &Conversation{
        Title: title,
    }
    _, err := pm.db.WithContext(ctx).Model(conversation).Insert()
    return conversation, err
}

func (pm *PostgresMemory) InsertMessage(ctx context.Context, conversationID int64, role, content string) (*Message, error) {
    message := &Message{
        ConversationID: conversationID,
        Role:           role,
        Content:        content,
    }
    _, err := pm.db.WithContext(ctx).Model(message).Insert()
    return message, err
}

func (pm *PostgresMemory) GetConversationByID(ctx context.Context, id int64) (*Conversation, error) {
    conversation := new(Conversation)
    err := pm.db.WithContext(ctx).Model(conversation).Where("id = ?", id).Select()
    return conversation, err
}

func (pm *PostgresMemory) GetMessagesByConversationID(ctx context.Context, conversationID int64) ([]Message, error) {
    var messages []Message
    err := pm.db.WithContext(ctx).Model(&messages).
        Where("conversation_id = ?", conversationID).
        Order("created_at ASC").
        Select()
    return messages, err
}


func (pm *PostgresMemory) SaveContext(ctx context.Context, inputs map[string]interface{}, outputs map[string]interface{}) error {
    conversationID, ok := inputs["conversation_id"].(int64)
    if !ok {
        conversation, err := pm.InsertConversation(ctx, "New Conversation")
        if err != nil {
            return err
        }
        conversationID = conversation.ID
    }

    if input, ok := inputs["input"].(string); ok {
        _, err := pm.InsertMessage(ctx, conversationID, "human", input)
        if err != nil {
            return err
        }
    }

    if output, ok := outputs["output"].(string); ok {
        _, err := pm.InsertMessage(ctx, conversationID, "ai", output)
        if err != nil {
            return err
        }
    }

    return pm.checkAndSummarize(ctx, conversationID)
}

func (pm *PostgresMemory) checkAndSummarize(ctx context.Context, conversationID int64) error {
    messageCount, err := pm.db.Model((*Message)(nil)).
        Where("conversation_id = ? AND is_summary = FALSE", conversationID).
        Count()
    if err != nil {
        return err
    }

    if messageCount >= 15 {
        return pm.SummarizeConversation(ctx, conversationID, 15)
    }

    return nil
}

func (pm *PostgresMemory) LoadMemoryVariables(ctx context.Context, inputs map[string]interface{}) (map[string]interface{}, error) {
    conversationID, ok := inputs["conversation_id"].(int64)
    if !ok {
        return nil, nil
    }

    messages, err := pm.GetMessagesByConversationID(ctx, conversationID)
    if err != nil {
        return nil, err
    }

    var history []map[string]string
    for _, msg := range messages {
        entry := map[string]string{
            "role":    msg.Role,
            "content": msg.Content,
        }
        if msg.IsSummary {
            entry["type"] = "summary"
        }
        history = append(history, entry)
    }

    return map[string]interface{}{
        "history": history,
    }, nil
}

func (pm *PostgresMemory) SummarizeConversation(ctx context.Context, conversationID int64, messageLimit int) error {
    messages, err := pm.GetMessagesByConversationID(ctx, conversationID)
    if err != nil {
        return err
    }

    if len(messages) <= messageLimit {
        return nil
    }

    var conversationText strings.Builder
    for _, msg := range messages {
        conversationText.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
    }

    summary, err := callLLMForSummary(conversationText.String())
    if err != nil {
        return err
    }

    tx, err := pm.db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    summaryMsg := &Message{
        ConversationID: conversationID,
        Role:           "system",
        Content:        summary,
        IsSummary:      true,
    }
    _, err = tx.Model(summaryMsg).Insert()
    if err != nil {
        return err
    }

    _, err = tx.Model((*Message)(nil)).
        Where("conversation_id = ? AND is_summary = FALSE", conversationID).
        Delete()
    if err != nil {
        return err
    }

    return tx.Commit()
}

func callLLMForSummary(conversationText string) (string, error) {
	model, err := ollama.New(ollama.WithModel(os.Getenv("DEFAULT_LLM")))
	if err != nil {
		return "", err
	}

	prompt := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, "You are a summarizer of a conversation; Only Return a summary of the provided conversation"),
		llms.TextParts(llms.ChatMessageTypeHuman, conversationText),
	}

	//TODO: Honestly no idea what this does but it's in the documentation
	ctx := context.Background()

	completion, err := model.GenerateContent(ctx, prompt, llms.WithMaxTokens(500))
	if err != nil {
		return "", err
	}

	// Extract the summary from the completion
	summary := completion.Choices[0].Content

	return summary, nil
}

func (pm *PostgresMemory) GetMessages(ctx context.Context) ([]llms.MessageContent, error) {
    messages, err := pm.GetMessagesByConversationID(ctx, pm.currentConversationID)
    if err != nil {
        return nil, err
    }
    var contentMessages []llms.MessageContent
    for _, msg := range messages {
        var role llms.ChatMessageType
        switch msg.Role {
        case "system":
            role = llms.ChatMessageTypeSystem
        case "human":
            role = llms.ChatMessageTypeHuman
        default:
            // Handle other types if necessary
            role = llms.ChatMessageTypeSystem
        }
        contentMessages = append(contentMessages, llms.MessageContent{
            Role:  role,
            Parts: []llms.ContentPart{msg.Content},
        })
    }
    return contentMessages, nil
}

func (pm *PostgresMemory) AddMessage(ctx context.Context, message llms.MessageContent) error {
    var role string
    switch message.Role {
    case llms.ChatMessageTypeSystem:
        role = "system"
    case llms.ChatMessageTypeHuman:
        role = "human"
    default:
        // Handle other types if necessary
        role = "system"
    }
    
    // Assuming we're dealing with text content in the first part
    content := string(message.Parts)
    _, err := pm.InsertMessage(ctx, pm.currentConversationID, role, content)
    return err
}

func (pm *PostgresMemory) Clear(ctx context.Context) error {
    // Create a new conversation
    conversation, err := pm.InsertConversation(ctx, "New Conversation")
    if err != nil {
        return err
    }
    pm.currentConversationID = conversation.ID
    return nil
}

type PostgresMemory struct {
    db                    *pg.DB
    currentConversationID int64
}

func NewPostgresMemory(db *pg.DB) *PostgresMemory {
    pm := &PostgresMemory{db: db}
    conversation, _ := pm.InsertConversation(context.Background(), "New Conversation")
    pm.currentConversationID = conversation.ID
    return pm
}