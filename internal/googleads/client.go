// Package googleads is a thin wrapper around the community-maintained Google
// Ads protobuf clients (github.com/shenzhencenter/google-ads-pb). It hides the
// gRPC plumbing so the Terraform resource layer can operate on plain Go
// structs and small CRUD methods.
package googleads

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	pbclients "github.com/shenzhencenter/google-ads-pb/clients"
)

// Config holds the credentials needed to talk to the Google Ads API.
//
// developer_token is retained for compatibility with Google Ads API versions
// that still accept it, but Google sunset developer tokens on 2026-09-09 and
// newer API versions authenticate API access through the Google Cloud project.
// login_customer_id is only needed when access to the target account is routed
// through a manager account. For authentication, supply one of:
//
//	(a) Service account: ServiceAccountJSONPath + ImpersonateEmail.
//	    Recommended for CI/server-to-server use. Requires the service
//	    account to have Google Workspace domain-wide delegation
//	    authorising the adwords scope, and ImpersonateEmail to be a
//	    user with access to the target Ads accounts.
//	(b) Refresh token: ClientID + ClientSecret + RefreshToken.
//	    Recommended for local development; minted once via the OAuth
//	    desktop flow.
//	(c) None of the above — falls back to Application Default
//	    Credentials. Useful with `gcloud auth application-default
//	    login --scopes=https://www.googleapis.com/auth/adwords`.
//
// If both (a) and (b) are configured, (a) wins.
type Config struct {
	DeveloperToken         string
	LoginCustomerID        string
	ServiceAccountJSONPath string
	ImpersonateEmail       string
	ClientID               string
	ClientSecret           string
	RefreshToken           string

	// Parallelism caps the number of concurrent Ads API requests this
	// client will have in-flight. 0 (the default) means no limit, which
	// matches historical behavior. Useful when a single Terraform apply
	// fans out across many customers under one MCC and you'd otherwise
	// trip the per-account RPC quota.
	Parallelism int

	// AllowDestructiveReplace opts into letting Terraform recreate
	// resources when an immutable field changes that would destroy
	// Google-Ads-side accumulated state (smart bidding learning,
	// historical performance, etc.). False by default — a change to
	// such a field errors at plan time instead of silently replacing.
	AllowDestructiveReplace bool

	// MaxRetries is the maximum number of additional attempts after a
	// failed call on transient gRPC codes (UNAVAILABLE,
	// DEADLINE_EXCEEDED, RESOURCE_EXHAUSTED). 0 (default) means no
	// retries — the call fails fast, preserving historical behavior.
	// Layered on top of any transport-level retry the underlying gRPC
	// client may perform.
	MaxRetries int

	// RetryInitialBackoff seeds the exponential backoff between retry
	// attempts (250ms → 500ms → 1s → …). Defaults to 250ms when zero
	// and MaxRetries > 0. The per-attempt wait caps at 30s.
	RetryInitialBackoff time.Duration
}

// Missing returns the field names of any required credentials that are empty,
// using the same snake_case names the provider schema exposes. Validates the
// auth-mode grouping rules — partial service-account configs or partial
// refresh-token triplets fail closed rather than silently falling through to
// ADC.
func (c Config) Missing() []string {
	var out []string
	saAny := c.ServiceAccountJSONPath != "" || c.ImpersonateEmail != ""
	if saAny {
		if c.ServiceAccountJSONPath == "" {
			out = append(out, "service_account_json_path")
		}
		if c.ImpersonateEmail == "" {
			out = append(out, "impersonate_email")
		}
	}
	rtAny := c.ClientID != "" || c.ClientSecret != "" || c.RefreshToken != ""
	if rtAny {
		if c.ClientID == "" {
			out = append(out, "client_id")
		}
		if c.ClientSecret == "" {
			out = append(out, "client_secret")
		}
		if c.RefreshToken == "" {
			out = append(out, "refresh_token")
		}
	}
	return out
}

// UsesServiceAccount reports whether NewClient will use the service-account
// JWT flow. True when both ServiceAccountJSONPath and ImpersonateEmail are
// set.
func (c Config) UsesServiceAccount() bool {
	return c.ServiceAccountJSONPath != "" && c.ImpersonateEmail != ""
}

// UsesRefreshToken reports whether NewClient will use the OAuth refresh-token
// flow. True when all three OAuth fields are set and the service-account
// path is not (service account always wins when both are configured).
func (c Config) UsesRefreshToken() bool {
	if c.UsesServiceAccount() {
		return false
	}
	return c.ClientID != "" && c.ClientSecret != "" && c.RefreshToken != ""
}

// UsesADC reports whether NewClient will fall back to Application Default
// Credentials for OAuth. True when neither the service-account nor the
// refresh-token flow is fully configured.
func (c Config) UsesADC() bool {
	return !c.UsesServiceAccount() && !c.UsesRefreshToken()
}

// Client bundles a set of lazily-initialised service clients sharing one set
// of credentials. It is safe for concurrent use by multiple Terraform
// resources / data sources.
type Client struct {
	cfg  Config
	opts []option.ClientOption

	// sem caps concurrent in-flight gRPC calls. nil means unlimited.
	// Set by NewClient when cfg.Parallelism > 0 and installed via a
	// unary interceptor.
	sem chan struct{}

	mu                       sync.Mutex
	campaigns                *pbclients.CampaignClient
	budgets                  *pbclients.CampaignBudgetClient
	adGroups                 *pbclients.AdGroupClient
	ads                      *pbclients.AdGroupAdClient
	criteria                 *pbclients.AdGroupCriterionClient
	googleAds                *pbclients.GoogleAdsClient
	customers                *pbclients.CustomerClient
	assets                   *pbclients.AssetClient
	assetGroups              *pbclients.AssetGroupClient
	assetGroupAssets         *pbclients.AssetGroupAssetClient
	conversionActions        *pbclients.ConversionActionClient
	sharedSets               *pbclients.SharedSetClient
	sharedCriteria           *pbclients.SharedCriterionClient
	campaignSharedSets       *pbclients.CampaignSharedSetClient
	customerNegativeCriteria *pbclients.CustomerNegativeCriterionClient
	campaignCriteria         *pbclients.CampaignCriterionClient
	labels                   *pbclients.LabelClient
	campaignLabels           *pbclients.CampaignLabelClient
	adGroupLabels            *pbclients.AdGroupLabelClient
	adGroupAdLabels          *pbclients.AdGroupAdLabelClient
	adGroupCriterionLabels   *pbclients.AdGroupCriterionLabelClient
	campaignAssets           *pbclients.CampaignAssetClient
	customerConversionGoals  *pbclients.CustomerConversionGoalClient
	adGroupAssets            *pbclients.AdGroupAssetClient
	customerAssets           *pbclients.CustomerAssetClient
}

// newClientFromOpts skips credential validation and the OAuth/header
// interceptor wiring, returning a Client that uses the supplied
// option.ClientOptions directly. Intended for unit tests that point
// the client at an in-process bufconn server via option.WithGRPCConn —
// production code must go through NewClient.
func newClientFromOpts(opts []option.ClientOption) *Client {
	return &Client{opts: opts}
}

// NewClient constructs a Client. The OAuth2 token source refreshes
// automatically; gRPC metadata interceptors attach the optional legacy
// developer-token and manager-only login-customer-id headers when configured.
//
// Auth comes from either:
//   - An explicit refresh-token triplet (ClientID, ClientSecret, RefreshToken), or
//   - Application Default Credentials — used when all three OAuth fields are
//     empty. Run `gcloud auth application-default login
//     --scopes=https://www.googleapis.com/auth/adwords,openid,https://www.googleapis.com/auth/userinfo.email`
//     once; the resulting credential is reused on every call.
func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	if len(cfg.Missing()) > 0 {
		return nil, fmt.Errorf("googleads: missing credentials: %v", cfg.Missing())
	}

	tokenSource, err := buildTokenSource(ctx, cfg)
	if err != nil {
		return nil, err
	}

	headers := requestHeaders(cfg)

	var sem chan struct{}
	if cfg.Parallelism > 0 {
		sem = make(chan struct{}, cfg.Parallelism)
	}

	backoff := cfg.RetryInitialBackoff
	if cfg.MaxRetries > 0 && backoff <= 0 {
		backoff = 250 * time.Millisecond
	}

	// Interceptor order: retry wraps everything so a transient
	// UNAVAILABLE doesn't burn a concurrency slot for the whole
	// backoff. Concurrency limit gates each attempt. Header
	// interceptor is innermost — the auth/dev-token headers must
	// land on every retry's request.
	opts := []option.ClientOption{
		option.WithTokenSource(tokenSource),
		option.WithGRPCDialOption(grpc.WithChainUnaryInterceptor(
			retryUnaryInterceptor(cfg.MaxRetries, backoff),
			concurrencyUnaryInterceptor(sem),
			unaryHeaderInterceptor(headers),
		)),
		option.WithGRPCDialOption(grpc.WithStreamInterceptor(streamHeaderInterceptor(headers))),
	}

	return &Client{cfg: cfg, opts: opts, sem: sem}, nil
}

// requestHeaders returns only explicitly configured compatibility/account
// routing headers. Empty values must not be sent: developer-token is obsolete
// for Cloud-project-managed access, and login-customer-id changes how Google
// resolves permissions for manager-account hierarchies.
func requestHeaders(cfg Config) []string {
	var headers []string
	if cfg.DeveloperToken != "" {
		headers = append(headers, "developer-token", cfg.DeveloperToken)
	}
	if cfg.LoginCustomerID != "" {
		headers = append(headers, "login-customer-id", cfg.LoginCustomerID)
	}
	return headers
}

const adwordsScope = "https://www.googleapis.com/auth/adwords"

func buildTokenSource(ctx context.Context, cfg Config) (oauth2.TokenSource, error) {
	// The TokenSource lives for the whole client lifetime and is used for
	// background refreshes; binding it to a request-scoped context (like
	// Terraform's Configure ctx, which is canceled once Configure returns)
	// makes every later refresh fail with "context canceled". Detach from
	// the caller's ctx by using a Background context for the source itself.
	tsCtx := context.Background()

	switch {
	case cfg.UsesServiceAccount():
		data, err := os.ReadFile(cfg.ServiceAccountJSONPath)
		if err != nil {
			return nil, fmt.Errorf("googleads: read service account JSON %q: %w", cfg.ServiceAccountJSONPath, err)
		}
		jwtCfg, err := google.JWTConfigFromJSON(data, adwordsScope)
		if err != nil {
			return nil, fmt.Errorf("googleads: parse service account JSON: %w", err)
		}
		// Domain-wide delegation requires us to act AS a real user with
		// access to the Ads accounts. The Ads API rejects service-account
		// principals directly.
		jwtCfg.Subject = cfg.ImpersonateEmail
		return jwtCfg.TokenSource(tsCtx), nil

	case cfg.UsesRefreshToken():
		oauthCfg := &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			Endpoint:     google.Endpoint,
			Scopes:       []string{adwordsScope},
		}
		return oauthCfg.TokenSource(tsCtx, &oauth2.Token{RefreshToken: cfg.RefreshToken}), nil

	default: // UsesADC
		creds, err := google.FindDefaultCredentials(ctx, adwordsScope)
		if err != nil {
			return nil, fmt.Errorf("googleads: application default credentials not available — supply service_account_json_path + impersonate_email, OR client_id/client_secret/refresh_token, OR run `gcloud auth application-default login --scopes=%s,openid,https://www.googleapis.com/auth/userinfo.email`: %w", adwordsScope, err)
		}
		return creds.TokenSource, nil
	}
}

// Close releases all underlying gRPC connections.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	var firstErr error
	closers := []interface{ Close() error }{}
	if c.campaigns != nil {
		closers = append(closers, c.campaigns)
	}
	if c.budgets != nil {
		closers = append(closers, c.budgets)
	}
	if c.adGroups != nil {
		closers = append(closers, c.adGroups)
	}
	if c.ads != nil {
		closers = append(closers, c.ads)
	}
	if c.criteria != nil {
		closers = append(closers, c.criteria)
	}
	if c.googleAds != nil {
		closers = append(closers, c.googleAds)
	}
	for _, x := range closers {
		if err := x.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// LoginCustomerID returns the manager account ID used by this client.
func (c *Client) LoginCustomerID() string { return c.cfg.LoginCustomerID }

func (c *Client) campaignClient(ctx context.Context) (*pbclients.CampaignClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.campaigns != nil {
		return c.campaigns, nil
	}
	cl, err := pbclients.NewCampaignClient(ctx, c.opts...)
	if err != nil {
		return nil, fmt.Errorf("googleads: campaign client: %w", err)
	}
	c.campaigns = cl
	return cl, nil
}

func (c *Client) budgetClient(ctx context.Context) (*pbclients.CampaignBudgetClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.budgets != nil {
		return c.budgets, nil
	}
	cl, err := pbclients.NewCampaignBudgetClient(ctx, c.opts...)
	if err != nil {
		return nil, fmt.Errorf("googleads: budget client: %w", err)
	}
	c.budgets = cl
	return cl, nil
}

func (c *Client) adGroupClient(ctx context.Context) (*pbclients.AdGroupClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.adGroups != nil {
		return c.adGroups, nil
	}
	cl, err := pbclients.NewAdGroupClient(ctx, c.opts...)
	if err != nil {
		return nil, fmt.Errorf("googleads: ad group client: %w", err)
	}
	c.adGroups = cl
	return cl, nil
}

func (c *Client) adGroupAdClient(ctx context.Context) (*pbclients.AdGroupAdClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ads != nil {
		return c.ads, nil
	}
	cl, err := pbclients.NewAdGroupAdClient(ctx, c.opts...)
	if err != nil {
		return nil, fmt.Errorf("googleads: ad group ad client: %w", err)
	}
	c.ads = cl
	return cl, nil
}

func (c *Client) adGroupCriterionClient(ctx context.Context) (*pbclients.AdGroupCriterionClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.criteria != nil {
		return c.criteria, nil
	}
	cl, err := pbclients.NewAdGroupCriterionClient(ctx, c.opts...)
	if err != nil {
		return nil, fmt.Errorf("googleads: criterion client: %w", err)
	}
	c.criteria = cl
	return cl, nil
}

func (c *Client) googleAdsClient(ctx context.Context) (*pbclients.GoogleAdsClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.googleAds != nil {
		return c.googleAds, nil
	}
	cl, err := pbclients.NewGoogleAdsClient(ctx, c.opts...)
	if err != nil {
		return nil, fmt.Errorf("googleads: google ads client: %w", err)
	}
	c.googleAds = cl
	return cl, nil
}

// ErrNotFound is returned when a Search returns zero rows for a lookup by
// resource name.
var ErrNotFound = errors.New("googleads: resource not found")

func unaryHeaderInterceptor(kv []string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx = metadata.AppendToOutgoingContext(ctx, kv...)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// retryUnaryInterceptor retries failed RPCs on transient codes
// (UNAVAILABLE, DEADLINE_EXCEEDED, RESOURCE_EXHAUSTED) with
// exponential backoff capped at 30s per attempt. maxRetries == 0
// makes the interceptor a pass-through (preserves historical
// fail-fast behavior).
func retryUnaryInterceptor(maxRetries int, initialBackoff time.Duration) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if maxRetries <= 0 {
			return invoker(ctx, method, req, reply, cc, opts...)
		}
		backoff := initialBackoff
		var err error
		for attempt := 0; attempt <= maxRetries; attempt++ {
			err = invoker(ctx, method, req, reply, cc, opts...)
			if err == nil || !isRetryable(err) {
				return err
			}
			if attempt == maxRetries {
				return err
			}
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return ctx.Err()
			}
			backoff *= 2
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
		}
		return err
	}
}

// isRetryable reports whether a gRPC error code is worth a retry.
// Conservative on purpose — INVALID_ARGUMENT and the like should
// fail immediately so the user sees the diagnostic at apply time
// instead of after a slow exponential backoff.
func isRetryable(err error) bool {
	st, ok := status.FromError(err)
	if !ok {
		return false
	}
	switch st.Code() {
	case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted:
		return true
	}
	return false
}

// concurrencyUnaryInterceptor gates calls through a buffered channel of
// size N. A nil channel disables limiting — the interceptor becomes a
// pass-through, so it's safe to install unconditionally.
func concurrencyUnaryInterceptor(sem chan struct{}) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if sem != nil {
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func streamHeaderInterceptor(kv []string) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		ctx = metadata.AppendToOutgoingContext(ctx, kv...)
		return streamer(ctx, desc, cc, method, opts...)
	}
}
