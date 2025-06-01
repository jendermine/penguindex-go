// File: penguindex-go/internal/commands/delete.go
package commands

import (
	"fmt"

	"github.com/jendermine/penguindex-go/internal/config"
	"github.com/jendermine/penguindex-go/internal/gdrive"
	"google.golang.org/api/drive/v3"
	"github.com/fatih/color"
)

// HandleDelete orchestrates the file deletion process.
func HandleDelete(driveSvc *drive.Service, _ *config.AppConfig, fileIDOrLink string) error {
	infoColor := color.New(color.FgCyan).SprintfFunc()
	successColor := color.New(color.FgGreen).SprintfFunc()

	fmt.Println(infoColor("Attempting to extract File ID from: %s", fileIDOrLink))
	actualFileID, err := gdrive.ExtractFileID(fileIDOrLink)
	if err != nil {
		return fmt.Errorf("invalid file ID or link: %w", err)
	}
	fmt.Println(infoColor("Extracted File ID: %s", actualFileID))

	fmt.Println(infoColor("Attempting to delete file with ID: %s", actualFileID))
	err = gdrive.DeleteDriveFile(driveSvc, actualFileID)
	if err != nil {
		// Check if the error is a "file not found" type to provide a better message
		// This is a bit simplistic; Google API errors have more structure.
		// if gErr, ok := err.(*googleapi.Error); ok && gErr.Code == 404 {
		// 	return fmt.Errorf("delete failed: File with ID '%s' not found on Google Drive", actualFileID)
		// }
		return fmt.Errorf("delete failed for ID '%s': %w", actualFileID, err)
	}

	fmt.Println(successColor("Successfully deleted file with ID: %s", actualFileID))
	// Optionally, send a Telegram notification about the deletion here if desired.
	return nil
}
