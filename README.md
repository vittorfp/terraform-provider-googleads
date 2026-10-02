# Terraform Provider for Google Ads

A community Terraform provider for managing [Google Ads](https://ads.google.com) resources, plus a generator that imports an existing account into HCL.

Built on [terraform-plugin-framework](https://developer.hashicorp.com/terraform/plugin/framework) and the community-maintained [google-ads-pb](https://github.com/shenzhencenter/google-ads-pb) protobuf definitions (Google does not publish an official Go SDK for the Ads API).

> [!IMPORTANT]
> This is an independent community project. It is not affiliated with, endorsed by, or supported by Google. The provider is pre-1.0 and its schema may still change between minor releases. Pin the version and review every Terraform plan before applying it to a production Ads account.

## Status

Public-beta candidate. The provider is used against real accounts and covers the core Search and Performance Max stacks, conversion actions, assets, targeting, shared negative lists, labels, and account import. Coverage is intentionally narrower than the full Google Ads API; see the generated [`docs/`](docs/) and [`BACKLOG.md`](BACKLOG.md) for the exact surface.

The current `google-ads-pb` dependency is v1.25.1 and targets Google Ads API v25. Google sunsets API versions on a rolling schedule, so check the [official sunset table](https://developers.google.com/google-ads/api/docs/sunset-dates) before adopting a release.

## Quick start

```hcl
terraform {
  required_providers {
    googleads = {
      source  = "vittorfp/googleads"
      version = "~> 0.5"
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
git tag v0.6.0
git push origin v0.6.0
# Wait for .github/workflows/release.yml to finish.
```

Every new release must be signed. Configure `GPG_PRIVATE_KEY` and `GPG_PASSPHRASE` as repository secrets before pushing a tag. The release workflow fails closed when signing is unavailable, and released versions must never be replaced because Terraform records their checksums.

## Support and security

This project is maintained on a best-effort basis with no response or resolution SLA. Use GitHub Issues for reproducible bugs and feature requests, and read [`SUPPORT.md`](SUPPORT.md) before posting. New issues receive an automated acknowledgement and enter a maintainer triage queue; that acknowledgement is not a technical diagnosis. For vulnerabilities or accidental credential exposure, follow [`SECURITY.md`](SECURITY.md) and do not open a public issue.

Contributions are welcome; see [`CONTRIBUTING.md`](CONTRIBUTING.md).

## License

MPL-2.0
