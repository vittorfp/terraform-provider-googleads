// Command googleads-tfgen reads one or more Google Ads accounts via GAQL
// and writes per-customer Terraform HCL (resource + import blocks) into an
// output directory.
//
// Auth on first run:
//   - Set GOOGLE_ADS_CLIENT_ID and GOOGLE_ADS_CLIENT_SECRET in the env (or a
//     .env file you source). Set GOOGLE_ADS_LOGIN_CUSTOMER_ID only when
//     accessing customers through a manager account.
//   - The tool opens your browser for Google sign-in and caches the
//     resulting refresh token at $XDG_CONFIG_HOME/googleads-tfgen/credentials.json.
//
// Later runs reuse the cached token silently. Use -relogin to force a fresh
// browser flow (e.g. to switch Google accounts).
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/vittorfp/terraform-provider-googleads/internal/googleads"
)

type stringSliceFlag []string

func (s *stringSliceFlag) String() string     { return strings.Join(*s, ",") }
func (s *stringSliceFlag) Set(v string) error { *s = append(*s, v); return nil }

func main() {
	var customers stringSliceFlag
	var mccs stringSliceFlag
	flag.Var(&customers, "customer", "Customer ID to generate (repeatable, digits only)")
	flag.Var(&mccs, "mcc", "MCC ID — enumerate all non-manager, ENABLED, non-hidden children and generate for each (repeatable)")
	outDir := flag.String("out", "generated", "Output directory")
	relogin := flag.Bool("relogin", false, "Discard the cached refresh token and re-run the browser sign-in flow")
	cachePath := flag.String("cache", "", "Override credential cache path (defaults to $XDG_CONFIG_HOME/googleads-tfgen/credentials.json)")
	noBrowser := flag.Bool("no-browser", false, "Do not auto-open the browser; just print the sign-in URL")
	foreachThreshold := flag.Int("foreach-threshold", 10, "Emit ad-group-criterion keywords as a single for_each block when an ad group has at least this many keywords. 0 disables, always per-block.")
	flag.Parse()

	if len(customers) == 0 && len(mccs) == 0 {
		log.Fatal("supply at least one -customer or -mcc")
	}

	ctx := context.Background()

	cfg, err := resolveCredentials(ctx, *cachePath, *relogin, *noBrowser)
	if err != nil {
		log.Fatal(err)
	}

	// Plain -customer batch: one client. When set, the env's
	// login_customer_id header applies to every account; direct-access users
	// can omit it.
	if len(customers) > 0 {
		client, err := googleads.NewClient(ctx, cfg)
		if err != nil {
			log.Fatalf("connect: %v", err)
		}
		runForCustomers(ctx, client, customers, *outDir, *foreachThreshold)
		_ = client.Close()
	}

	// -mcc batch: one client per MCC (each needs login_customer_id =
	// MCC ID so the header is correct for every child query). Iterates
	// the MCC's customer_client tree once to enumerate children, then
	// fans the generator over them.
	for _, mccID := range mccs {
		mccCfg := cfg
		mccCfg.LoginCustomerID = mccID
		client, err := googleads.NewClient(ctx, mccCfg)
		if err != nil {
			log.Fatalf("connect to MCC %s: %v", mccID, err)
		}
		children, err := client.ListLeafCustomerClients(ctx, mccID)
		if err != nil {
			log.Fatalf("enumerate MCC %s: %v", mccID, err)
		}
		fmt.Fprintf(os.Stderr, "↳ MCC %s has %d non-manager child accounts\n", mccID, len(children))
		childIDs := make([]string, 0, len(children))
		for _, ch := range children {
			childIDs = append(childIDs, fmt.Sprintf("%d", ch.ID))
		}
		runForCustomers(ctx, client, childIDs, *outDir, *foreachThreshold)
		_ = client.Close()
	}
}

// runForCustomers iterates a flat customer-ID list and emits a folder
// of HCL per account. Shared by the -customer and -mcc batch flows.
func runForCustomers(ctx context.Context, client *googleads.Client, cids []string, outDir string, foreachThreshold int) {
	for _, cid := range cids {
		dir := filepath.Join(outDir, cid)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.Fatalf("mkdir %s: %v", dir, err)
		}
		s, err := generateForCustomer(ctx, client, cid, dir, foreachThreshold)
		if err != nil {
			log.Fatalf("customer %s: %v", cid, err)
		}
		fmt.Printf("✓ %s → %s (%s)\n", cid, dir, s)
	}
}

// resolveCredentials assembles a googleads.Config from env vars and the
// local credential cache, prompting the user through the browser OAuth flow
// when there is no refresh token available yet.
func resolveCredentials(ctx context.Context, cacheOverride string, forceRelogin, noBrowser bool) (googleads.Config, error) {
	cfg := googleads.Config{
		DeveloperToken:  os.Getenv("GOOGLE_ADS_DEVELOPER_TOKEN"),
		LoginCustomerID: os.Getenv("GOOGLE_ADS_LOGIN_CUSTOMER_ID"),
		ClientID:        os.Getenv("GOOGLE_ADS_CLIENT_ID"),
		ClientSecret:    os.Getenv("GOOGLE_ADS_CLIENT_SECRET"),
		RefreshToken:    os.Getenv("GOOGLE_ADS_REFRESH_TOKEN"),
	}

	// DeveloperToken is a legacy compatibility option. LoginCustomerID is
	// inferred for -mcc and optional for direct-access -customer calls.

	// If the user set a refresh token explicitly, respect it and skip the
	// cache / browser flow entirely.
	if cfg.RefreshToken != "" {
		return cfg, nil
	}

	// If neither the OAuth client nor any cached creds exist, fall through
	// and let NewClient try ADC. (UsesADC() will be true.)
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return cfg, nil
	}

	cachePath := cacheOverride
	if cachePath == "" {
		p, err := googleads.DefaultCachePath()
		if err != nil {
			return cfg, fmt.Errorf("resolve cache path: %w", err)
		}
		cachePath = p
	}

	if forceRelogin {
		_ = os.Remove(cachePath)
	} else {
		cached, err := googleads.LoadCachedRefreshToken(cachePath)
		if err != nil {
			return cfg, fmt.Errorf("read credential cache %s: %w", cachePath, err)
		}
		if cached != "" {
			cfg.RefreshToken = cached
			return cfg, nil
		}
	}

	fmt.Fprintln(os.Stderr, "No cached Google Ads credential found. Starting sign-in...")
	refreshToken, err := googleads.InteractiveLogin(ctx, cfg.ClientID, cfg.ClientSecret, googleads.LoginOptions{NoBrowser: noBrowser})
	if err != nil {
		return cfg, fmt.Errorf("interactive login: %w", err)
	}
	if err := googleads.SaveRefreshToken(cachePath, refreshToken); err != nil {
		return cfg, fmt.Errorf("save refresh token: %w", err)
	}
	fmt.Fprintf(os.Stderr, "✓ Saved refresh token to %s\n\n", cachePath)
	cfg.RefreshToken = refreshToken
	return cfg, nil
}
