// File: penguindex-go/internal/auth/auth.go
package auth

import (
	"context"
	"fmt"
	"net/http"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// GetAuthenticatedClient creates an HTTP client authenticated with Google Cloud
// using the provided service account JSON string.
func GetAuthenticatedClient(serviceAccountJSONString string) (*http.Client, error) {
	ctx := context.Background()
	creds, err := google.CredentialsFromJSON(ctx, []byte(serviceAccountJSONString), drive.DriveScope)
	if err != nil {
		return nil, fmt.Errorf("failed to create credentials from service account JSON: %w", err)
	}

	client := oauth2.NewClient(ctx, creds.TokenSource)
	return client, nil
}

// NewDriveService creates a new Google Drive service client using an authenticated HTTP client.
func NewDriveService(client *http.Client) (*drive.Service, error) {
	ctx := context.Background()
	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create Drive service: %w", err)
	}
	return srv, nil
}
