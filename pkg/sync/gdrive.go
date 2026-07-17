package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/abcubed3/okf/pkg/bundle"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

var (
	// Embedded OAuth 2.0 credentials (can be injected via ldflags)
	googleClientID     = ""
	googleClientSecret = ""
)

type GoogleDriveConnector struct {
	state  *StateManager
	config *GoogleDriveConfig
	srv    *drive.Service
}

func NewGoogleDriveConnector(state *StateManager) *GoogleDriveConnector {
	return &GoogleDriveConnector{
		state: state,
	}
}

func (c *GoogleDriveConnector) Name() string {
	return "google_drive"
}

func (c *GoogleDriveConnector) Initialize(ctx context.Context, cfg *Config) error {
	if cfg.Connectors.GoogleDrive == nil {
		return fmt.Errorf("google_drive configuration missing")
	}
	c.config = cfg.Connectors.GoogleDrive

	if c.config.ServiceAccount != "" {
		srv, err := drive.NewService(ctx, option.WithCredentialsFile(c.config.ServiceAccount))
		if err != nil {
			return fmt.Errorf("unable to retrieve Drive client via service account: %v", err)
		}
		c.srv = srv
	} else if c.config.CredentialsFile != "" || googleClientID != "" {
		var oauthConfig *oauth2.Config

		if c.config.CredentialsFile != "" {
			b, err := os.ReadFile(c.config.CredentialsFile)
			if err != nil {
				return fmt.Errorf("unable to read credentials file: %v", err)
			}
			oauthConfig, err = google.ConfigFromJSON(b, drive.DriveFileScope)
			if err != nil {
				return fmt.Errorf("unable to parse client secret file to config: %v", err)
			}
		} else {
			oauthConfig = &oauth2.Config{
				ClientID:     googleClientID,
				ClientSecret: googleClientSecret,
				Endpoint:     google.Endpoint,
				Scopes:       []string{drive.DriveFileScope},
				RedirectURL:  "http://localhost:8080/",
			}
		}

		tokenFile := c.config.TokenFile
		if tokenFile == "" {
			tokenFile = "google_token.json"
		}

		tok, err := tokenFromFile(tokenFile)
		if err != nil {
			tok = getTokenFromWeb(oauthConfig)
			saveToken(tokenFile, tok)
		}

		client := oauthConfig.Client(ctx, tok)
		srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
		if err != nil {
			return fmt.Errorf("unable to retrieve Drive client via oauth: %v", err)
		}
		c.srv = srv
	} else {
		return fmt.Errorf("google_drive requires either service_account or credentials_file in config")
	}

	log.Printf("[google_drive] Initialized with FolderID: %s", c.config.FolderID)
	return nil
}

func (c *GoogleDriveConnector) Push(ctx context.Context, concept *bundle.Concept) error {
	if c.config == nil || c.srv == nil {
		return nil // Not configured
	}

	extID := c.state.GetExternalID(concept.ID, c.Name())
	content := concept.Body
	title := concept.Frontmatter.Title
	if title == "" {
		title = concept.ID
	}

	if extID != "" {
		// Update existing file
		log.Printf("[google_drive] Updating concept %s (ID: %s) on Google Drive...", concept.ID, extID)
		f := &drive.File{
			Name: title,
		}
		_, err := c.srv.Files.Update(extID, f).Media(strings.NewReader(content)).Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("failed to update file in drive: %w", err)
		}
		return nil
	}

	log.Printf("[google_drive] Pushing new concept %s (%s)...", concept.ID, title)
	f := &drive.File{
		Name:     title,
		MimeType: "application/vnd.google-apps.document", // Convert to Google Doc
		Parents:  []string{c.config.FolderID},
	}

	res, err := c.srv.Files.Create(f).Media(strings.NewReader(content)).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to create file in drive: %w", err)
	}

	c.state.SetExternalID(concept.ID, c.Name(), res.Id)
	return nil
}

func (c *GoogleDriveConnector) Pull(ctx context.Context) ([]*bundle.Concept, error) {
	if c.config == nil || c.srv == nil {
		return nil, nil // Not configured
	}
	log.Printf("[google_drive] Pulling updates...")

	// Default to a wide query if no last sync time
	lastSyncStr := c.state.state.LastSync.Format(time.RFC3339)
	if c.state.state.LastSync.IsZero() {
		// Just a fallback in case we want to pull all
		lastSyncStr = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
	}

	query := fmt.Sprintf("'%s' in parents and modifiedTime > '%s' and trashed = false",
		c.config.FolderID, lastSyncStr)

	r, err := c.srv.Files.List().Q(query).Fields("nextPageToken, files(id, name, modifiedTime)").Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve files: %v", err)
	}

	for _, i := range r.Files {
		log.Printf("[google_drive] Found modified file: %s (%s)\n", i.Name, i.Id)
		// Export as plain text
		resp, err := c.srv.Files.Export(i.Id, "text/plain").Context(ctx).Download()
		if err != nil {
			log.Printf("[google_drive] Failed to download %s: %v", i.Id, err)
			continue
		}
		resp.Body.Close()

		// In a real implementation, we would figure out which local concept ID this matches
		// (e.g., by scanning the state mappings backwards), and update the local `.md` file.
		// For this prototype, we'll just log it.
	}

	return nil, nil
}

// Request a token from the web, using a local callback server.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	config.RedirectURL = "http://localhost:8080/"
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", "select_account"))
	fmt.Printf("\nGo to the following link in your browser to authorize okf:\n\n%v\n\n", authURL)

	codeCh := make(chan string)
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code != "" {
			fmt.Fprintf(w, "Authentication successful! You can close this window and return to the okf CLI.")
			codeCh <- code
		} else {
			fmt.Fprintf(w, "Authentication failed! No code provided.")
			codeCh <- ""
		}
	})

	srv := &http.Server{Addr: ":8080", Handler: mux}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start local server: %v", err)
		}
	}()

	fmt.Println("Waiting for authorization code on http://localhost:8080/...")
	authCode := <-codeCh

	srv.Shutdown(context.Background())

	if authCode == "" {
		log.Fatalf("Failed to get authorization code")
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}
