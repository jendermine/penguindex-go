// File: penguindex-go/internal/telegram/telegram.go
package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// TelegramSendMessagePayload defines the structure for the message payload.
type TelegramSendMessagePayload struct {
	ChatID                string                `json:"chat_id"`
	Text                  string                `json:"text"`
	ParseMode             string                `json:"parse_mode"` // MarkdownV2
	DisableWebPagePreview bool                  `json:"disable_web_page_preview"`
	ReplyMarkup           *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

// InlineKeyboardMarkup for buttons.
type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

// InlineKeyboardButton for a single button.
type InlineKeyboardButton struct {
	Text string `json:"text"`
	URL  string `json:"url"`
}

var markdownV2Escaper = strings.NewReplacer(
	"_", "\\_", "*", "\\*", "[", "\\[", "]", "\\]", "(",
	"\\(", ")", "\\)", "~", "\\~", "`", "\\`", ">",
	"\\>", "#", "\\#", "+", "\\+", "-", "\\-", "=",
	"\\=", "|", "\\|", "{", "\\{", "}", "\\}", ".",
	"\\.", "!", "\\!",
)

func escapeMarkdownV2(text string) string {
	return markdownV2Escaper.Replace(text)
}

// SendNotification sends a message to a Telegram chat.
func SendNotification(botToken, chatID, fileName, folderName, size, mimeType, createdTime, gdriveLink, ddlLink string) error {
	messageText := fmt.Sprintf(
		"*File Uploaded* ✅\n\n"+
			"*File Name*: `%s`\n"+
			"*Folder*: `%s`\n"+
			"*Size*: `%s`\n"+
			"*Type*: `%s`\n"+
			"*Created*: `%s`",
		escapeMarkdownV2(fileName),
		escapeMarkdownV2(folderName),
		escapeMarkdownV2(size),
		escapeMarkdownV2(mimeType),
		escapeMarkdownV2(createdTime),
	)

	payload := TelegramSendMessagePayload{
		ChatID:                chatID,
		Text:                  messageText,
		ParseMode:             "MarkdownV2",
		DisableWebPagePreview: false, // Set to true if you don't want link previews for GDrive/DDL
		ReplyMarkup: &InlineKeyboardMarkup{
			InlineKeyboard: [][]InlineKeyboardButton{
				{
					{Text: "☁️ GDrive Link", URL: gdriveLink},
					{Text: "🔗 Direct Link", URL: ddlLink},
				},
			},
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Telegram payload: %w", err)
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)

	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to send Telegram message request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Telegram API error: %s - %s", resp.Status, string(bodyBytes))
	}

	return nil
}
