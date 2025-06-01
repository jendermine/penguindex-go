// File: penguindex-go/internal/gdrive/gdrive.go
package gdrive

import (
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/schollz/progressbar/v3" // Progress bar
	"google.golang.org/api/drive/v3"
	"github.com/fatih/color"
)

// ProgressTrackingFileReader wraps an os.File to track read progress for uploads.
type ProgressTrackingFileReader struct {
	File     *os.File
	Size     int64
	Bar      *progressbar.ProgressBar
	Reader   io.Reader // This will be io.TeeReader if progress bar is used
	FileName string
}

// NewProgressTrackingFileReader creates a new reader with a progress bar.
func NewProgressTrackingFileReader(filePath string) (*ProgressTrackingFileReader, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	fileInfo, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to get file info for %s: %w", filePath, err)
	}

	fileName := filepath.Base(filePath)
	bar := progressbar.NewOptions64(
		fileInfo.Size(),
		progressbar.OptionSetWriter(os.Stdout), // Use os.Stdout or os.Stderr
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(30),
		progressbar.OptionSetDescription(color.CyanString(fmt.Sprintf("Uploading %s...", fileName))),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        color.GreenString("="),
			SaucerHead:    color.GreenString(">"),
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionThrottle(100*time.Millisecond), // Update progress bar less frequently
	)

	return &ProgressTrackingFileReader{
		File:     file,
		Size:     fileInfo.Size(),
		Bar:      bar,
		Reader:   io.TeeReader(file, bar), // Reads from file, writes to bar
		FileName: fileName,
	}, nil
}

// Read implements io.Reader.
func (p *ProgressTrackingFileReader) Read(b []byte) (int, error) {
	return p.Reader.Read(b)
}

// Close implements io.Closer.
func (p *ProgressTrackingFileReader) Close() error {
	if p.Bar != nil {
		// On completion, ensure the bar shows 100% if it hasn't already.
		_ = p.Bar.Finish() // We don't really care about error on finish for the bar
	}
	return p.File.Close()
}

// UploadFile uploads a file to Google Drive with progress.
func UploadFile(svc *drive.Service, filePath string, targetFolderID string) (*drive.File, error) {
	progressReader, err := NewProgressTrackingFileReader(filePath)
	if err != nil {
		return nil, err // Error already contains file path
	}
	defer progressReader.Close()

	mimeType := mime.TypeByExtension(filepath.Ext(progressReader.FileName))
	if mimeType == "" {
		mimeType = "application/octet-stream" // Default MIME type
	}

	driveFile := &drive.File{
		Name:     progressReader.FileName,
		MimeType: mimeType,
	}
	if targetFolderID != "" {
		driveFile.Parents = []string{targetFolderID}
	}

	// The Go client library handles resumable uploads automatically for larger files
	// when Media() is provided with an io.Reader.
	createdFile, err := svc.Files.Create(driveFile).Media(progressReader).Fields("id", "name", "mimeType", "size", "createdTime", "webViewLink", "webContentLink", "parents").Do()
	if err != nil {
		// Ensure progress bar is cleared or marked as failed on error
		if progressReader.Bar != nil {
			progressReader.Bar.Clear()
		}
		return nil, fmt.Errorf("failed to upload file '%s' to Google Drive: %w", progressReader.FileName, err)
	}
	// Ensure progress bar is explicitly finished on success (if not already by TeeReader)
	if progressReader.Bar != nil && progressReader.Bar.IsFinished() == false {
		_ = progressReader.Bar.Finish()
	}
	return createdFile, nil
}

// DeleteDriveFile deletes a file from Google Drive by its ID.
func DeleteDriveFile(svc *drive.Service, fileID string) error {
	err := svc.Files.Delete(fileID).Do()
	if err != nil {
		return fmt.Errorf("failed to delete file '%s' from Google Drive: %w", fileID, err)
	}
	return nil
}

// driveIdRegex for extracting file ID from various GDrive link formats.
var driveIdRegex = regexp.MustCompile(`(?:(?:https?:\/\/drive\.google\.com\/(?:file\/d\/|open\?id=|drive\/folders\/|folderview\?id=))|(?:\b))([a-zA-Z0-9_-]{25,})(?:\b|\?|$)`)

// ExtractFileID extracts the Google Drive file ID from a string (which can be an ID or a link).
func ExtractFileID(idOrLink string) (string, error) {
	matches := driveIdRegex.FindStringSubmatch(idOrLink)
	if len(matches) > 1 && matches[1] != "" {
		return matches[1], nil // The first capturing group is the ID
	}
	// If no regex match, assume the input itself might be an ID.
	// Basic validation for typical GDrive ID characters and length.
	// This regex is simpler than the above, just for validating a potential raw ID.
	rawIdRegex := regexp.MustCompile(`^[a-zA-Z0-9_-]{25,}$`)
	if rawIdRegex.MatchString(idOrLink) {
		return idOrLink, nil
	}
	return "", fmt.Errorf("invalid or unextractable Google Drive ID/link format: %s", idOrLink)
}
