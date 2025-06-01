// File: penguindex-go/internal/commands/upload.go
package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/jendermine/penguindex-go/internal/config"
	"github.com/jendermine/penguindex-go/internal/gdrive"
	"github.com/jendermine/penguindex-go/internal/telegram"
	"github.com/jendermine/penguindex-go/internal/utils"
	"google.golang.org/api/drive/v3"
	"github.com/fatih/color"
)

// HandleUpload orchestrates the file upload process.
func HandleUpload(driveSvc *drive.Service, appCfg *config.AppConfig, filePath, folderID string) error {
	infoColor := color.New(color.FgCyan).SprintfFunc()
	successColor := color.New(color.FgGreen).SprintfFunc()

	if folderID == "" {
		folderID = appCfg.DefaultFolderID
		fmt.Println(infoColor("No folder ID provided, using default: %s", folderID))
	}

	fmt.Println(infoColor("Starting upload for: %s to folder ID: %s", filePath, folderID))
	uploadedFile, err := gdrive.UploadFile(driveSvc, filePath, folderID)
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}
	fmt.Println(successColor("\n--- Upload Successful ---")) // Newline to ensure it's after progress bar

	// --- Process and display results ---
	gdriveLink := uploadedFile.WebViewLink
	if gdriveLink == "" { // Fallback if WebViewLink is not populated for some reason
		gdriveLink = fmt.Sprintf("https://drive.google.com/file/d/%s/view?usp=sharing", uploadedFile.Id)
	}
	// webContentLink is often the direct download link for files stored natively.
	// For Google Docs, Sheets, etc., it might be an export link.
	// Your Rust DDL: https://drive.google.com/uc?export=download&id={FILE_ID}
	ddlLink := fmt.Sprintf("https://drive.google.com/uc?export=download&id=%s", uploadedFile.Id)

	var createdTime time.Time
	if uploadedFile.CreatedTime != "" {
		createdTime, err = time.Parse(time.RFC3339, uploadedFile.CreatedTime)
		if err != nil {
			fmt.Printf("Warning: Could not parse file creation time '%s': %v\n", uploadedFile.CreatedTime, err)
		}
	}

	fileSizeStr := utils.HumanReadableSize(uint64(uploadedFile.Size))

	fmt.Printf("File Name: %s\n", successColor(uploadedFile.Name))
	fmt.Printf("Size: %s\n", successColor(fileSizeStr))
	fmt.Printf("MIME Type: %s\n", successColor(uploadedFile.MimeType))
	if !createdTime.IsZero() {
		fmt.Printf("Created: %s\n", successColor(createdTime.Format("2006-01-02 15:04:05 MST")))
	}
	fmt.Printf("Gdrive Link: %s\n", successColor(gdriveLink))
	fmt.Printf("DDL Link: %s\n", successColor(ddlLink))


	// Fetch folder name if one parent exists
	folderName := "N/A"
	if len(uploadedFile.Parents) > 0 {
		parentFolderID := uploadedFile.Parents[0]
		parentFolder, err := driveSvc.Files.Get(parentFolderID).Fields("name").Do()
		if err == nil {
			folderName = parentFolder.Name
		} else {
			fmt.Printf("Warning: Could not fetch parent folder name for ID %s: %v\n", parentFolderID, err)
		}
	}
	fmt.Printf("Folder Name: %s\n", successColor(folderName))


	// Send Telegram Notification
	if appCfg.TelegramBotToken != "" && appCfg.TelegramChatID != "" {
		fmt.Println(infoColor("Sending Telegram notification..."))
		var createdTimeStr string
		if !createdTime.IsZero() {
			createdTimeStr = createdTime.Format("02 Jan 06 15:04 MST")
		} else {
			createdTimeStr = "N/A"
		}

		err = telegram.SendNotification(
			appCfg.TelegramBotToken,
			appCfg.TelegramChatID,
			uploadedFile.Name,
			folderName,
			fileSizeStr,
			uploadedFile.MimeType,
			createdTimeStr,
			gdriveLink,
			ddlLink,
		)
		if err != nil {
			fmt.Printf(color.YellowString("Warning: Failed to send Telegram notification: %v\n"), err)
		} else {
			fmt.Println(successColor("Telegram notification sent successfully."))
		}
	} else {
		fmt.Println(color.YellowString("Telegram bot token or chat ID not configured. Skipping notification."))
	}

	return nil
}
