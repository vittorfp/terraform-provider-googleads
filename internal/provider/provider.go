package provider

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vittorfp/terraform-provider-googleads/internal/googleads"
)

var _ provider.Provider = (*googleadsProvider)(nil)

type googleadsProvider struct {
	version string
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &googleadsProvider{version: version}
	}
}

type providerModel struct {
	DeveloperToken          types.String `tfsdk:"developer_token"`
	LoginCustomerID         types.String `tfsdk:"login_customer_id"`
	ServiceAccountJSONPath  types.String `tfsdk:"service_account_json_path"`
	ImpersonateEmail        types.String `tfsdk:"impersonate_email"`
	ClientID                types.String `tfsdk:"client_id"`
	ClientSecret            types.String `tfsdk:"client_secret"`
	RefreshToken            types.String `tfsdk:"refresh_token"`
	Parallelism             types.Int64  `tfsdk:"parallelism"`
	AllowDestructiveReplace types.Bool   `tfsdk:"allow_destructive_replace"`
	MaxRetries              types.Int64  `tfsdk:"max_retries"`
	RetryBackoffMs          types.Int64  `tfsdk:"retry_backoff_ms"`
}

func (p *googleadsProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "googleads"
	resp.Version = p.version
}

func (p *googleadsProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage Google Ads resources. Credentials may be supplied via provider block or the GOOGLE_ADS_* environment variables. Three auth modes, tried in this order: (1) service account — `service_account_json_path` + `impersonate_email` (recommended for CI; requires Workspace domain-wide delegation); (2) refresh token — `client_id` + `client_secret` + `refresh_token` (recommended for local development); (3) Application Default Credentials, if neither of the above is set. API access is controlled by the Google Cloud project that owns the OAuth credentials. `developer_token` is a legacy compatibility option, and `login_customer_id` is only needed when accessing an account through a manager account.",
		Attributes: map[string]schema.Attribute{
			"developer_token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Legacy Google Ads developer token. Google sunset developer tokens on 2026-09-09; omit this for Cloud-project-managed access. When supplied for compatibility with older API versions, it falls back to GOOGLE_ADS_DEVELOPER_TOKEN.",
			},
			"login_customer_id": schema.StringAttribute{
				Optional:    true,
				Description: "Manager (MCC) account ID used as the login-customer-id header. Required only when access to the target account is through a manager account; omit it for direct access. Digits only, no dashes. Falls back to GOOGLE_ADS_LOGIN_CUSTOMER_ID.",
			},
			"service_account_json_path": schema.StringAttribute{
				Optional:    true,
				Description: "Path to a service account JSON key. When set alongside impersonate_email, the provider authenticates via the JWT/domain-wide-delegation flow — the right mode for CI runners. Falls back to GOOGLE_ADS_SERVICE_ACCOUNT_JSON_PATH.",
			},
			"impersonate_email": schema.StringAttribute{
				Optional:    true,
				Description: "Workspace user the service account impersonates. Required when service_account_json_path is set. The user must have access to the target Ads accounts. Falls back to GOOGLE_ADS_IMPERSONATE_EMAIL.",
			},
			"client_id": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "OAuth2 client ID. Falls back to GOOGLE_ADS_CLIENT_ID.",
			},
			"client_secret": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "OAuth2 client secret. Falls back to GOOGLE_ADS_CLIENT_SECRET.",
			},
			"refresh_token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "OAuth2 refresh token. Falls back to GOOGLE_ADS_REFRESH_TOKEN.",
			},
			"parallelism": schema.Int64Attribute{
				Optional:    true,
				Description: "Maximum concurrent in-flight Ads API requests across all resources sharing this provider. 0 (the default) means unlimited and matches historical behavior. Useful when a single Terraform apply fans out across many customers under one MCC and would otherwise trip the per-account RPC quota. Falls back to GOOGLE_ADS_PARALLELISM.",
			},
			"allow_destructive_replace": schema.BoolAttribute{
				Optional:    true,
				Description: "Opt in to allowing Terraform to destroy and recreate resources when an immutable field changes that would otherwise lose Google Ads-side accumulated state (smart bidding learning, performance history, etc.). Default false — changes to those fields error at plan time and require either reverting the change or explicitly setting this to true for the run. Falls back to GOOGLE_ADS_ALLOW_DESTRUCTIVE_REPLACE.",
			},
			"max_retries": schema.Int64Attribute{
				Optional:    true,
				Description: "Maximum number of additional attempts after a failed call on transient gRPC codes (UNAVAILABLE, DEADLINE_EXCEEDED, RESOURCE_EXHAUSTED). 0 (default) disables retries — the call fails fast. Useful when a single Terraform apply fans out across many customers and you'd rather absorb transient API hiccups than partial-fail. Falls back to GOOGLE_ADS_MAX_RETRIES.",
			},
			"retry_backoff_ms": schema.Int64Attribute{
				Optional:    true,
				Description: "Initial wait between retry attempts in milliseconds. Doubles on each retry, capped at 30s. Defaults to 250 when max_retries > 0. Falls back to GOOGLE_ADS_RETRY_BACKOFF_MS.",
			},
		},
	}
}

func (p *googleadsProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clientCfg := googleads.Config{
		DeveloperToken:          pick(cfg.DeveloperToken, "GOOGLE_ADS_DEVELOPER_TOKEN"),
		LoginCustomerID:         pick(cfg.LoginCustomerID, "GOOGLE_ADS_LOGIN_CUSTOMER_ID"),
		ServiceAccountJSONPath:  pick(cfg.ServiceAccountJSONPath, "GOOGLE_ADS_SERVICE_ACCOUNT_JSON_PATH"),
		ImpersonateEmail:        pick(cfg.ImpersonateEmail, "GOOGLE_ADS_IMPERSONATE_EMAIL"),
		ClientID:                pick(cfg.ClientID, "GOOGLE_ADS_CLIENT_ID"),
		ClientSecret:            pick(cfg.ClientSecret, "GOOGLE_ADS_CLIENT_SECRET"),
		RefreshToken:            pick(cfg.RefreshToken, "GOOGLE_ADS_REFRESH_TOKEN"),
		Parallelism:             pickInt64(cfg.Parallelism, "GOOGLE_ADS_PARALLELISM"),
		AllowDestructiveReplace: pickBool(cfg.AllowDestructiveReplace, "GOOGLE_ADS_ALLOW_DESTRUCTIVE_REPLACE"),
		MaxRetries:              pickInt64(cfg.MaxRetries, "GOOGLE_ADS_MAX_RETRIES"),
		RetryInitialBackoff:     time.Duration(pickInt64(cfg.RetryBackoffMs, "GOOGLE_ADS_RETRY_BACKOFF_MS")) * time.Millisecond,
	}

	allowDestructiveReplace.Store(clientCfg.AllowDestructiveReplace)

	missing := clientCfg.Missing()
	if len(missing) > 0 {
		for _, name := range missing {
			resp.Diagnostics.AddAttributeError(
				attrPath(name),
				"Missing Google Ads credential",
				"The provider requires "+name+". Set it in the provider block or via the corresponding GOOGLE_ADS_* environment variable.",
			)
		}
		return
	}

	client, err := googleads.NewClient(ctx, clientCfg)
	if err != nil {
		resp.Diagnostics.AddError("Unable to construct Google Ads client", err.Error())
		return
	}

	resp.ResourceData = client
	resp.DataSourceData = client
}

func (p *googleadsProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewCampaignBudgetResource,
		NewCampaignResource,
		NewAdGroupResource,
		NewAdGroupAdResource,
		NewAdGroupCriterionResource,
		NewTextAssetResource,
		NewImageAssetResource,
		NewAssetGroupResource,
		NewAssetGroupAssetResource,
		NewConversionActionResource,
		NewSharedSetResource,
		NewSharedCriterionResource,
		NewCampaignSharedSetResource,
		NewCustomerNegativeCriterionResource,
		NewCampaignCriterionResource,
		NewAdGroupAudienceCriterionResource,
		NewLabelResource,
		NewCampaignLabelResource,
		NewAdGroupLabelResource,
		NewAdGroupAdLabelResource,
		NewAdGroupCriterionLabelResource,
		NewSitelinkAssetResource,
		NewCalloutAssetResource,
		NewStructuredSnippetAssetResource,
		NewCampaignAssetResource,
		NewAdGroupAssetResource,
		NewCustomerAssetResource,
		NewCustomerConversionGoalResource,
	}
}

func (p *googleadsProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewCustomerDataSource,
	}
}

func pick(v types.String, env string) string {
	if !v.IsNull() && !v.IsUnknown() && v.ValueString() != "" {
		return v.ValueString()
	}
	return os.Getenv(env)
}

// pickInt64 mirrors pick for Int64 attributes with env fallback. A
// non-numeric env value is silently treated as zero so a typo doesn't
// crash provider startup.
func pickInt64(v types.Int64, env string) int {
	if !v.IsNull() && !v.IsUnknown() {
		return int(v.ValueInt64())
	}
	if s := os.Getenv(env); s != "" {
		n, err := strconv.Atoi(s)
		if err == nil {
			return n
		}
	}
	return 0
}

// pickBool mirrors pick for Bool attributes with env fallback. Env
// values are matched loosely — "1", "true", "yes" (case-insensitive)
// mean true; anything else (including empty) means false.
func pickBool(v types.Bool, env string) bool {
	if !v.IsNull() && !v.IsUnknown() {
		return v.ValueBool()
	}
	switch s := os.Getenv(env); s {
	case "1", "true", "TRUE", "True", "yes", "YES", "Yes":
		return true
	}
	return false
}
