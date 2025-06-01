// File: penguindex-go/main.go
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/jendermine/penguindex-go/internal/auth"
	"github.com/jendermine/penguindex-go/internal/commands"
	"github.com/jendermine/penguindex-go/internal/config"
	"github.com/fatih/color" // For colored output
	"golang.org/x/term"      // For PIN input
)

// !!! REPLACE THESE WITH YOUR ACTUAL VALUES !!!
const EMBEDDED_BUNDLE_URL = "https://gist.githubusercontent.com/jendermine/f963de2bcf12c37421277d7702466b2b/raw/ceabd48a9f0f6412a1dd42af44f20b5619d04d6d/log.json"
const TELEGRAM_CHAT_ID_URL = "https://gist.githubusercontent.com/jendermine/66015cce5cf15c0e04ba5987cb3ca342/raw/2e0f17aaee25abbcfa8a254f390bcb214775826b/log2.json"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Color setup (optional)
	errorColor := color.New(color.FgRed).SprintfFunc()
	successColor := color.New(color.FgGreen).SprintfFunc()
	infoColor := color.New(color.FgYellow).SprintfFunc()

	fmt.Println(infoColor("Fetching configuration..."))
	appConfigDetails, err := config.FetchRemoteConfigDetails(EMBEDDED_BUNDLE_URL, TELEGRAM_CHAT_ID_URL)
	if err != nil {
		fmt.Fprintln(os.Stderr, errorColor("Error fetching remote configuration: %v", err))
		os.Exit(1)
	}

	fmt.Print("Enter PIN: ")
	pinBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Fprintln(os.Stderr, errorColor("Error reading PIN: %v", err))
		os.Exit(1)
	}
	pin := string(pinBytes)
	fmt.Println() // Newline after PIN input

	decryptedBundle, err := config.DecryptBundle(appConfigDetails.EncryptedBundleHex, pin)
	if err != nil {
		fmt.Fprintln(os.Stderr, errorColor("Error decrypting bundle (check PIN or bundle URL): %v", err))
		os.Exit(1)
	}
	fmt.Println(successColor("Bundle decrypted successfully."))

	appCfg := &config.AppConfig{
		ServiceAccountJSON: decryptedBundle.ServiceAccountJSONString,
		TelegramBotToken:   decryptedBundle.TelegramBotToken,
		TelegramChatID:     appConfigDetails.TelegramChatID,
		DefaultFolderID:    config.DEFAULT_TEST_FOLDER_ID, // From config package
	}

	fmt.Println(infoColor("Authenticating with Google Drive..."))
	driveHTTPClient, err := auth.GetAuthenticatedClient(appCfg.ServiceAccountJSON)
	if err != nil {
		fmt.Fprintln(os.Stderr, errorColor("Google Drive authentication failed: %v", err))
		os.Exit(1)
	}

	driveService, err := auth.NewDriveService(driveHTTPClient)
	if err != nil {
		fmt.Fprintln(os.Stderr, errorColor("Failed to create Google Drive service: %v", err))
		os.Exit(1)
	}
	// Perform an auth check
	gDriveUser, err := driveService.About.Get().Fields("user").Do()
	if err != nil {
		fmt.Fprintln(os.Stderr, errorColor("Failed to verify Drive service authentication: %v", err))
		os.Exit(1)
	}
	fmt.Println(successColor("Successfully authenticated with Google Drive as: %s", gDriveUser.User.EmailAddress))


	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "upload":
		uploadCmd := flag.NewFlagSet("upload", flag.ExitOnError)
		filePath := uploadCmd.String("file", "", "Path to the file to upload (required)")
		folderID := uploadCmd.String("folder", "", "Google Drive folder ID (optional, uses default if not provided)")

		uploadCmd.Usage = func() {
			fmt.Fprintf(os.Stderr, "Usage: %s upload -file <filepath> [-folder <folderID>]\n", os.Args[0])
			uploadCmd.PrintDefaults()
		}
		if err := uploadCmd.Parse(args); err != nil {
            fmt.Fprintln(os.Stderr, errorColor("Error parsing upload flags: %v", err))
			os.Exit(1)
		}


		if *filePath == "" {
			fmt.Fprintln(os.Stderr, errorColor("Error: --file flag is required for upload."))
			uploadCmd.Usage()
			os.Exit(1)
		}
		actualFolderID := *folderID
		if actualFolderID == "" {
			actualFolderID = appCfg.DefaultFolderID
		}
		err := commands.HandleUpload(driveService, appCfg, *filePath, actualFolderID)
		if err != nil {
			fmt.Fprintln(os.Stderr, errorColor("Upload command failed: %v", err))
			os.Exit(1)
		}
		fmt.Println(successColor("Upload command completed successfully."))

	case "delete":
		deleteCmd := flag.NewFlagSet("delete", flag.ExitOnError)
		fileIDOrLink := deleteCmd.String("id", "", "File ID or Google Drive link to delete (required)")
		
		deleteCmd.Usage = func() {
			fmt.Fprintf(os.Stderr, "Usage: %s delete -id <fileID_or_link>\n", os.Args[0])
			deleteCmd.PrintDefaults()
		}
		if err := deleteCmd.Parse(args); err != nil {
            fmt.Fprintln(os.Stderr, errorColor("Error parsing delete flags: %v", err))
			os.Exit(1)
		}


		if *fileIDOrLink == "" {
			fmt.Fprintln(os.Stderr, errorColor("Error: --id flag is required for delete."))
			deleteCmd.Usage()
			os.Exit(1)
		}
		err := commands.HandleDelete(driveService, appCfg, *fileIDOrLink)
		if err != nil {
			fmt.Fprintln(os.Stderr, errorColor("Delete command failed: %v", err))
			os.Exit(1)
		}
		fmt.Println(successColor("Delete command completed successfully."))

	default:
		fmt.Fprintln(os.Stderr, errorColor("Unknown command: %s", command))
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s <command> [arguments]\n", os.Args[0])
	fmt.Fprintln(os.Stderr, "Available commands: upload, delete")
	fmt.Fprintln(os.Stderr, "Use <command> -help for more information on a specific command.")
}
