# Terraform Provider for Google Ads

[![Terraform Registry](https://img.shields.io/terraform/provider/v/vittorfp/googleads?label=registry)](https://registry.terraform.io/providers/vittorfp/googleads/latest)
[![Downloads](https://img.shields.io/terraform/provider/dt/vittorfp/googleads)](https://registry.terraform.io/providers/vittorfp/googleads/latest)
[![CI](https://github.com/vittorfp/terraform-provider-googleads/actions/workflows/ci.yml/badge.svg)](https://github.com/vittorfp/terraform-provider-googleads/actions/workflows/ci.yml)
[![License: MPL-2.0](https://img.shields.io/badge/license-MPL--2.0-blue.svg)](LICENSE)

Manage Google Ads with Terraform. This independent community provider supports Search campaigns, Performance Max, conversion actions, assets, targeting, shared negative lists, labels, and account-to-HCL import.

Use it to keep Google Ads campaign configuration in version control, review changes through Terraform plans, reproduce account structures, and turn an existing account into maintainable HCL with the included `googleads-tfgen` generator.

Built on [terraform-plugin-framework](https://developer.hashicorp.com/terraform/plugin/framework) and the community-maintained [google-ads-pb](https://github.com/shenzhencenter/google-ads-pb) protobuf definitions (Google does not publish an official Go SDK for the Ads API).

> [!IMPORTANT]
> This is an independent community project. It is not affiliated with, endorsed by, or supported by Google. The provider is pre-1.0 and its schema may still change between minor releases. Pin the version and review every Terraform plan before applying it to a production Ads account.

## Status

Public beta. The provider is used against real accounts and covers the core Search and Performance Max stacks, conversion actions, assets, targeting, shared negative lists, labels, and account import. Coverage is intentionally narrower than the full Google Ads API; see the generated [`docs/`](docs/) and [`BACKLOG.md`](BACKLOG.md) for the exact surface.

The current `google-ads-pb` dependency is v1.25.2 and targets Google Ads API v25. Google sunsets API versions on a rolling schedule, so check the [official sunset table](https://developers.google.com/google-ads/api/docs/sunset-dates) before adopting a release.

## What you can manage

| Area | Included resources and tooling |
| --- | --- |
| Search campaigns | Campaigns, budgets, ad groups, responsive search ads, keywords, audiences, and campaign criteria |
| Performance Max | Asset groups, asset-group assets, text and image assets, and campaign assets |
| Measurement | Conversion actions and customer conversion goals |
| Account controls | Labels, shared negative lists, customer negatives, callouts, sitelinks, and structured snippets |
| Existing accounts | Read-only discovery and account-to-HCL generation with `googleads-tfgen` |

See the generated [`docs/`](docs/) for the complete resource schema and [`BACKLOG.md`](BACKLOG.md) for known coverage gaps.

## Installation

Signed `v0.6.1` binaries are available from the [Terraform Registry](https://registry.terraform.io/providers/vittorfp/googleads/latest) and [GitHub Releases](https://github.com/vittorfp/terraform-provider-googleads/releases/tag/v0.6.1). Use the normal installation flow shown in [Quick start](#quick-start).

To develop against the current source instead, clone and build it locally:

```sh
git clone https://github.com/vittorfp/terraform-provider-googleads.git
cd terraform-provider-googleads
mkdir -p dist/dev
go build -o dist/dev/terraform-provider-googleads .
```

Point Terraform at the absolute build directory with a development override in `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "vittorfp/googleads" = "/absolute/path/to/terraform-provider-googleads/dist/dev"
  }

  direct {}
}
```

With the override active, run `terraform plan` directly from your Terraform configuration. A development override deliberately bypasses version selection and checksums, so use it only for evaluation and remove it after installing the provider through the Registry.

## Quick start

```hcl
terraform {
  required_providers {
    googleads = {
      source  = "vittorfp/googleads"
      version = "~> 0.6"
    }
  }
}

provider "googleads" {
  # All fields read from env vars if unset:
  #   GOOGLE_ADS_LOGIN_CUSTOMER_ID  # only for manager-account access
  #   GOOGLE_ADS_CLIENT_ID
  #   GOOGLE_ADS_CLIENT_SECRET
  #   GOOGLE_ADS_REFRESH_TOKEN
}

resource "googleads_campaign_budget" "default" {
  customer_id   = "1234567890"
  name          = "tf-budget"
  amount_micros = 5000000 # $5/day
}

resource "googleads_campaign" "search" {
  customer_id              = "1234567890"
  name                     = "tf-search"
  advertising_channel_type = "SEARCH"
  campaign_budget_id       = googleads_campaign_budget.default.id
  status                   = "PAUSED"
}
```

## Import an existing account

Use [`googleads-tfgen`](cmd/googleads-tfgen/README.md) to turn an existing
Google Ads account into Terraform resource and import blocks. The generator
makes read-only API calls and writes local files under `generated/`; it does
not change the Ads account.

```sh
export GOOGLE_ADS_CLIENT_ID=...
export GOOGLE_ADS_CLIENT_SECRET=...

go run ./cmd/googleads-tfgen -customer 1234567890
cd generated/1234567890
terraform init
terraform plan
```

The first run opens the Google OAuth flow and stores the refresh token in a
mode-`0600` local credentials file. Set `GOOGLE_ADS_LOGIN_CUSTOMER_ID` only
when the target account is accessed through a manager account. Review the
generated comments and the complete plan before running `terraform apply`;
unsupported Google Ads resource variants are intentionally skipped.

See the [generator guide](cmd/googleads-tfgen/README.md) for MCC discovery,
non-interactive authentication, generated file coverage, and known limits.

## Development

```sh
make build          # compile the provider
make install        # install to ~/.terraform.d/plugins/...
make test           # unit tests
make testacc        # acceptance tests (requires Ads credentials)
make docs           # regenerate docs via tfplugindocs
```

## Authentication

API access is attached to the Google Cloud project that owns your OAuth credentials. Configure access in the [Google Ads API page in Google Cloud Console](https://console.cloud.google.com/google/ads/api/overview). Google sunset developer tokens on September 9, 2026; the optional `developer_token` setting remains temporarily for compatibility with older API versions but is omitted from requests when unset.

`login_customer_id` is only needed when the authenticated identity reaches the target account through a manager account. Omit it for direct account access.

The provider tries three credential modes in this order:

**Service account (recommended for CI / production)** — supply a service-account JSON key plus the email of a Google Workspace user the service account impersonates:

```sh
export GOOGLE_ADS_SERVICE_ACCOUNT_JSON_PATH=/path/to/sa.json
export GOOGLE_ADS_IMPERSONATE_EMAIL=user@example.com
```

Requires Workspace [domain-wide delegation](https://developers.google.com/identity/protocols/oauth2/service-account#delegatingauthority) authorising the `adwords` scope on the SA. The impersonated user must have access to the target Ads accounts.

**OAuth refresh token (recommended for local development)** — supply `GOOGLE_ADS_CLIENT_ID`, `GOOGLE_ADS_CLIENT_SECRET`, and `GOOGLE_ADS_REFRESH_TOKEN`. Generated once via the [OAuth2 desktop flow](https://developers.google.com/google-ads/api/docs/oauth/cloud-project), and the `googleads-tfgen` CLI will mint and cache one for you on first run.

**Application Default Credentials (fallback)** — if neither of the above is fully configured, the provider falls through to ADC. Useful with:

```sh
gcloud auth application-default login \
  --scopes=https://www.googleapis.com/auth/adwords,openid,https://www.googleapis.com/auth/userinfo.email
```

Then set `GOOGLE_ADS_LOGIN_CUSTOMER_ID` only if you access the target through a manager account. Existing integrations may continue setting `GOOGLE_ADS_DEVELOPER_TOKEN` while using an API version that still accepts it.

### Reusing an existing `google-ads.yaml`

If you already have a working `google-ads.yaml` from another Google Ads SDK, its OAuth and account-routing values map straight onto environment variables — no second OAuth flow is needed:

```yaml
# google-ads.yaml
login_customer_id: "1234567890"
client_id: "...apps.googleusercontent.com"
client_secret: "..."
refresh_token: "1//0g..."
```

```sh
# Same shell where you'll run terraform:
export GOOGLE_ADS_LOGIN_CUSTOMER_ID=$(yq '.login_customer_id' ~/.google-ads.yaml)
export GOOGLE_ADS_CLIENT_ID=$(yq '.client_id' ~/.google-ads.yaml)
export GOOGLE_ADS_CLIENT_SECRET=$(yq '.client_secret' ~/.google-ads.yaml)
export GOOGLE_ADS_REFRESH_TOKEN=$(yq '.refresh_token' ~/.google-ads.yaml)
```

If the legacy YAML still contains `developer_token`, you may export it as `GOOGLE_ADS_DEVELOPER_TOKEN`; new Cloud-project-managed setups do not need it.

Use `yq`, `python -c "import yaml,sys; print(yaml.safe_load(open('...'))['client_id'])"`, or just copy the values by hand.

## Drift on auto-managed fields

The Google Ads system rewrites a handful of fields on its own (Smart Bidding adjusting `target_roas`, optimisation score, etc.). Terraform reads those back on every plan and would otherwise show a perpetual diff. Use `lifecycle.ignore_changes` to silence the noise on the fields the system owns:

```hcl
resource "googleads_campaign" "smart_bidding" {
  customer_id              = "1234567890"
  name                     = "tf-pmax"
  advertising_channel_type = "PERFORMANCE_MAX"
  campaign_budget_id       = googleads_campaign_budget.b.id
  bidding_strategy_type    = "TARGET_ROAS"
  target_roas              = 3.5 # initial target; Smart Bidding will drift it

  lifecycle {
    ignore_changes = [target_roas]
  }
}
```

Fields most likely to drift on their own:

- `target_roas` / `target_cpa_micros` under MAXIMIZE_* strategies (Smart Bidding tunes them).
- `cpc_bid_ceiling_micros` / `cpc_bid_floor_micros` if you let the system widen them.
- `status` after a campaign self-pauses on policy / budget exhaustion — surfaces as a plan diff on next refresh.

If a field appears in plan output every time but you didn't touch it, that's the signal to add it to `ignore_changes`.

## Releasing

Releases fire on tag push (`v*`). The workflow uses [goreleaser](https://goreleaser.com) to cross-compile and publish a GitHub release with archives, checksums, and the Terraform Registry manifest.

```sh
# After merging changes to main (replace with the next unused version):
git tag v0.6.1
git push origin v0.6.1
# Wait for .github/workflows/release.yml to finish.
```

Every new release must be signed. Configure `GPG_PRIVATE_KEY` and `GPG_PASSPHRASE` as repository secrets before pushing a tag. The release workflow fails closed when signing is unavailable, and released versions must never be replaced because Terraform records their checksums.

## Support and security

This project is maintained on a best-effort basis with no response or resolution SLA. Use GitHub Issues for reproducible bugs and feature requests, and read [`SUPPORT.md`](SUPPORT.md) before posting. New issues receive an automated acknowledgement and enter a maintainer triage queue; that acknowledgement is not a technical diagnosis. For vulnerabilities or accidental credential exposure, follow [`SECURITY.md`](SECURITY.md) and do not open a public issue.

Contributions are welcome; see [`CONTRIBUTING.md`](CONTRIBUTING.md).

## Frequently asked questions

### Can Terraform manage Google Ads?

Yes. This provider exposes a focused set of Google Ads resources through the standard Terraform workflow: configuration, plan, apply, import, and state refresh. It does not yet cover the entire Google Ads API.

### Is there an official Google Ads Terraform provider?

Google does not publish or support this provider. `vittorfp/googleads` is an independent community project built with the Terraform Plugin Framework and community-maintained Google Ads protobuf definitions.

### Can I import an existing Google Ads account into Terraform?

Yes. The included [`googleads-tfgen`](cmd/googleads-tfgen/README.md) command reads an accessible account and generates Terraform resource and import blocks. It makes read-only discovery calls and does not change the Ads account.

### Does the provider support Performance Max campaigns?

It supports the core Performance Max structure, including campaigns, budgets, asset groups, and asset-group assets. Check the generated documentation and backlog for the exact supported fields and variants.

### Is it ready for production use?

The provider is a pre-1.0 public beta used against real accounts. Pin the provider version, review every plan, test changes on a non-critical account first, and expect some schema changes between minor releases.

## License

MPL-2.0
