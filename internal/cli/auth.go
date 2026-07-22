package cli

import (
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/abcubed3/okf/pkg/config"
)

// RunAuth handles the "okf auth" CLI command.
func RunAuth(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("subcommand required: login")
	}

	subcommand := args[0]
	cmdArgs := args[1:]

	switch subcommand {
	case "login":
		return RunAuthLogin(cmdArgs)
	default:
		return fmt.Errorf("unknown auth subcommand: %s", subcommand)
	}
}

// RunAuthLogin handles the "okf auth login" CLI command.
func RunAuthLogin(args []string) error {
	token := ""
	fs := flag.NewFlagSet("auth login", flag.ContinueOnError)
	fs.StringVar(&token, "token", "", "OKF Hub API Key (if you prefer not to use the interactive prompt)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	// If token is provided via flag, bypass browser flow
	if token != "" {
		return saveToken(token)
	}

	// Browser-based device login flow
	port := 8989
	authURL := fmt.Sprintf("http://localhost:8080/cli-auth?port=%d", port)

	fmt.Println("To authenticate, please visit the following URL in your web browser:")
	fmt.Printf("\n    %s\n\n", authURL)
	fmt.Println("Waiting for authorization...")

	tokenChan := make(chan string)
	errChan := make(chan error)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		t := r.URL.Query().Get("token")
		if t == "" {
			http.Error(w, "Missing token", http.StatusBadRequest)
			errChan <- fmt.Errorf("callback received no token")
			return
		}

		// Success response with auto-close attempt
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`
			<!DOCTYPE html>
			<html>
			<head><title>OKF Auth Successful</title></head>
			<body style="font-family: sans-serif; text-align: center; margin-top: 50px;">
				<h2>Login Successful!</h2>
				<p>You may safely close this window.</p>
				<script>
					// Attempt to auto-close the window
					window.close();
					setTimeout(function() {
						document.body.innerHTML += '<p style="color: gray; font-size: 0.9em;">(If the window did not close automatically, please close it manually.)</p>';
					}, 500);
				</script>
			</body>
			</html>
		`))
		tokenChan <- t
	})

	server := &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: mux}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("local server error: %w", err)
		}
	}()

	select {
	case t := <-tokenChan:
		// Shut down server asynchronously after short delay to allow browser response to finish
		go func() {
			time.Sleep(200 * time.Millisecond)
			_ = server.Close()
		}()
		return saveToken(t)
	case err := <-errChan:
		_ = server.Close()
		return err
	}
}

func saveToken(token string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		// Log the error but don't fail, we can just create a new config
		cfg = &config.Config{}
	}

	cfg.APIKey = token

	if err := config.SaveConfig(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	configPath, _ := config.GetConfigPath()
	fmt.Printf("Successfully logged in. Credentials saved to %s\n", configPath)
	return nil
}
