package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	daemon bool
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
		creds, err := os.ReadFile(c.config.ServiceAccount)
		if err != nil {
			return fmt.Errorf("unable to read service account credentials file: %w", err)
		}
		srv, err := drive.NewService(ctx, option.WithCredentialsJSON(creds))
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
			if c.daemon {
				return fmt.Errorf("google_drive OAuth token missing (%s) and interactive browser auth is disabled in --daemon mode", tokenFile)
			}
			tok, err = getTokenFromWeb(oauthConfig)
			if err != nil {
				return fmt.Errorf("google_drive OAuth authorization failed: %w", err)
			}
			if err := saveToken(tokenFile, tok); err != nil {
				// Non-fatal: token caching failure shouldn't prevent the session from working.
				log.Printf("[google_drive] Warning: failed to cache OAuth token to %s: %v", tokenFile, err)
			}
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

func (c *GoogleDriveConnector) Push(ctx context.Context, concepts []*bundle.Concept) error {
	if c.config == nil || c.srv == nil {
		return nil // Not configured
	}

	for _, concept := range concepts {
		extID := c.state.GetExternalID(concept.ID, c.Name())
		content, err := SerializeConceptEnvelope(concept)
		if err != nil {
			log.Printf("[google_drive] Failed to serialize %s: %v", concept.ID, err)
			continue
		}
		hash, _ := HashConcept(concept)
		title := concept.Frontmatter.Title
		if title == "" {
			title = concept.ID
		}

		var appProps map[string]string
		if len("okf_concept_id")+len(concept.ID) <= 124 {
			appProps = map[string]string{"okf_concept_id": concept.ID}
		}

		if extID != "" {
			log.Printf("[google_drive] Updating concept %s (ID: %s) on Google Drive...", concept.ID, extID)
			f := &drive.File{
				Name:          title,
				AppProperties: appProps,
			}
			res, err := c.srv.Files.Update(extID, f).Media(strings.NewReader(content)).Fields("id, modifiedTime").Context(ctx).Do()
			if err != nil {
				log.Printf("[google_drive] failed to update file %s: %v", extID, err)
				continue
			}
			if hash != "" {
				c.state.SetContentHash(concept.ID, c.Name(), hash)
			}
			if res.ModifiedTime != "" {
				c.state.SetRemoteRevision(concept.ID, c.Name(), res.ModifiedTime)
			}
			continue
		}

		log.Printf("[google_drive] Pushing new concept %s (%s)...", concept.ID, title)
		f := &drive.File{
			Name:          title,
			MimeType:      "application/vnd.google-apps.document",
			Parents:       []string{c.config.FolderID},
			AppProperties: appProps,
		}

		res, err := c.srv.Files.Create(f).Media(strings.NewReader(content)).Fields("id, modifiedTime").Context(ctx).Do()
		if err != nil {
			log.Printf("[google_drive] failed to create file for %s: %v", concept.ID, err)
			continue
		}

		c.state.SetExternalID(concept.ID, c.Name(), res.Id)
		if hash != "" {
			c.state.SetContentHash(concept.ID, c.Name(), hash)
		}
		if res.ModifiedTime != "" {
			c.state.SetRemoteRevision(concept.ID, c.Name(), res.ModifiedTime)
		}
	}
	return nil
}

func (c *GoogleDriveConnector) Pull(ctx context.Context) ([]*bundle.Concept, error) {
	if c.config == nil || c.srv == nil {
		return nil, nil // Not configured
	}
	log.Printf("[google_drive] Pulling updates...")

	lastSync := c.state.GetLastSync()
	lastSyncStr := lastSync.Format(time.RFC3339)
	if lastSync.IsZero() {
		lastSyncStr = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
	}

	query := fmt.Sprintf("'%s' in parents and modifiedTime > '%s' and trashed = false",
		c.config.FolderID, lastSyncStr)

	var pulled []*bundle.Concept
	pageToken := ""
	for {
		call := c.srv.Files.List().Q(query).Fields("nextPageToken, files(id, name, modifiedTime, appProperties)").Context(ctx)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}
		r, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("unable to retrieve files: %v", err)
		}

		for _, f := range r.Files {
			conceptID := ""
			if f.AppProperties != nil {
				conceptID = f.AppProperties["okf_concept_id"]
			}
			if conceptID == "" {
				conceptID = c.state.FindConceptByExternalID(c.Name(), f.Id)
			}
			if conceptID == "" {
				conceptID = f.Name
			}
			if f.ModifiedTime != "" && c.state.GetRemoteRevision(conceptID, c.Name()) == f.ModifiedTime {
				continue
			}

			resp, err := c.srv.Files.Export(f.Id, "text/plain").Context(ctx).Download()
			if err != nil {
				log.Printf("[google_drive] Failed to download %s (%s): %v", f.Name, f.Id, err)
				continue
			}
			rawBytes, readErr := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
			resp.Body.Close()
			if readErr != nil {
				log.Printf("[google_drive] Failed to read %s (%s): %v", f.Name, f.Id, readErr)
				continue
			}

			concept, _, parseErr := ParseConceptEnvelope(conceptID, "", string(rawBytes))
			if parseErr != nil {
				log.Printf("[google_drive] Quarantine %s (%s): malformed okf-frontmatter YAML: %v", conceptID, f.Id, parseErr)
				continue
			}
			c.state.SetExternalID(conceptID, c.Name(), f.Id)
			if f.ModifiedTime != "" {
				c.state.SetRemoteRevision(conceptID, c.Name(), f.ModifiedTime)
			}
			pulled = append(pulled, concept)
		}

		if r.NextPageToken == "" {
			break
		}
		pageToken = r.NextPageToken
	}

	return pulled, nil
}

// getTokenFromWeb requests a token from the web using a local OAuth2 callback server.
// Returns the token or an error — never calls log.Fatal.
func getTokenFromWeb(config *oauth2.Config) (*oauth2.Token, error) {
	config.RedirectURL = "http://localhost:8080/"
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", "select_account"))
	fmt.Printf("\nGo to the following link in your browser to authorize okf:\n\n%v\n\n", authURL)

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
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
			errCh <- fmt.Errorf("failed to start local OAuth callback server on :8080: %w", err)
		}
	}()

	fmt.Println("Waiting for authorization code on http://localhost:8080/...")

	var authCode string
	select {
	case authCode = <-codeCh:
	case err := <-errCh:
		return nil, err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)

	if authCode == "" {
		return nil, fmt.Errorf("OAuth authorization failed: no code received from browser")
	}

	tok, err := config.Exchange(context.Background(), authCode)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve token from web: %w", err)
	}
	return tok, nil
}

// tokenFromFile retrieves a token from a local file.
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

// saveToken saves a token to a file path. Returns an error instead of calling log.Fatal.
func saveToken(path string, token *oauth2.Token) error {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("unable to cache oauth token: %w", err)
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(token)
}
