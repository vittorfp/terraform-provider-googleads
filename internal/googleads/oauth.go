package googleads

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// cachedCreds is the persisted shape of a successful interactive login.
type cachedCreds struct {
	RefreshToken string `json:"refresh_token"`
}

// DefaultCachePath returns the platform-appropriate location for the cached
// refresh token: $XDG_CONFIG_HOME/googleads-tfgen/credentials.json on Linux,
// ~/Library/Application Support/googleads-tfgen/credentials.json on macOS,
// %AppData%\googleads-tfgen\credentials.json on Windows.
func DefaultCachePath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "googleads-tfgen", "credentials.json"), nil
}

// LoadCachedRefreshToken reads a refresh token from path. Returns ("", nil)
// if the file does not exist — that's a normal first-run state, not an error.
func LoadCachedRefreshToken(path string) (string, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	var c cachedCreds
	if err := json.Unmarshal(data, &c); err != nil {
		return "", fmt.Errorf("parse %s: %w", path, err)
	}
	return c.RefreshToken, nil
}

// SaveRefreshToken writes the token to path with 0o600 permissions, creating
// parent directories as needed.
func SaveRefreshToken(path, token string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cachedCreds{RefreshToken: token}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// LoginOptions configures a single InteractiveLogin run.
type LoginOptions struct {
	// NoBrowser suppresses the auto-open of the system browser. The URL is
	// still printed to stderr so the user can open it manually (handy on
	// headless boxes, over SSH, or when the default browser is wrong).
	NoBrowser bool
}

// InteractiveLogin runs the OAuth 2.0 installed-application flow against the
// Google identity endpoints with the `adwords` scope. It binds a local
// listener on a random port, opens the user's browser (unless
// opts.NoBrowser), waits for the redirect, exchanges the resulting code for
// a refresh token, and returns it. Blocks up to 5 minutes for the user to
// complete consent.
//
// The caller must provide a Desktop-type OAuth client (client_id +
// client_secret) from Google Cloud Console. The user signing in must have
// access to the target Ads accounts.
func InteractiveLogin(ctx context.Context, clientID, clientSecret string, opts LoginOptions) (string, error) {
	if clientID == "" || clientSecret == "" {
		return "", errors.New("googleads: client_id and client_secret required for interactive OAuth flow")
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("bind callback listener: %w", err)
	}
	defer listener.Close()

	redirectURL := fmt.Sprintf("http://%s/callback", listener.Addr().String())
	cfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{adwordsScope},
		RedirectURL:  redirectURL,
	}
	state := randomState()
	// AccessTypeOffline + ApprovalForce ensures Google returns a refresh
	// token even if the user has consented to this client before.
	authURL := cfg.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)

	type result struct {
		code string
		err  error
	}
	results := make(chan result, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("state") != state {
			http.Error(w, "state mismatch", http.StatusBadRequest)
			results <- result{err: errors.New("oauth state mismatch — possible CSRF")}
			return
		}
		if e := q.Get("error"); e != "" {
			http.Error(w, e, http.StatusBadRequest)
			results <- result{err: fmt.Errorf("oauth error: %s", e)}
			return
		}
		code := q.Get("code")
		if code == "" {
			http.Error(w, "missing code", http.StatusBadRequest)
			results <- result{err: errors.New("oauth callback missing code")}
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, successHTML)
		results <- result{code: code}
	})

	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go srv.Serve(listener) //nolint:errcheck
	defer srv.Shutdown(context.Background())

	if opts.NoBrowser {
		fmt.Fprintln(os.Stderr, "→ Open this URL in your browser to sign in:")
		fmt.Fprintf(os.Stderr, "\n  %s\n\n", authURL)
	} else {
		fmt.Fprintln(os.Stderr, "→ Opening browser for Google sign-in...")
		fmt.Fprintf(os.Stderr, "  If nothing happens, open this URL manually:\n  %s\n\n", authURL)
		_ = openBrowser(authURL)
	}

	select {
	case res := <-results:
		if res.err != nil {
			return "", res.err
		}
		token, err := cfg.Exchange(ctx, res.code)
		if err != nil {
			return "", fmt.Errorf("exchange auth code: %w", err)
		}
		if token.RefreshToken == "" {
			return "", errors.New("Google returned no refresh token — revoke this app at https://myaccount.google.com/permissions and retry")
		}
		return token.RefreshToken, nil
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(5 * time.Minute):
		return "", errors.New("timed out waiting for browser callback after 5 minutes")
	}
}

const successHTML = `<!doctype html>
<html><head><title>Signed in</title>
<style>body{font:16px -apple-system,system-ui,sans-serif;text-align:center;padding:4em;color:#333}h2{color:#0a7d3e}</style>
</head><body><h2>✓ Signed in</h2><p>You may close this tab and return to your terminal.</p></body></html>`

func openBrowser(url string) error {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler"}
	default:
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

func randomState() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
