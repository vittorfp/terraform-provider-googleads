# googleads-tfgen

Pulls existing Google Ads resources out of one or more accounts and writes them as Terraform HCL — resource blocks plus matching `import` blocks — so you can adopt an account into Terraform without writing the configuration by hand.

## Usage

Two modes:

- **`-customer <cid>`** (repeatable) — generate HCL for these specific accounts. Set `GOOGLE_ADS_LOGIN_CUSTOMER_ID` only when access is through a manager account; omit it for direct access.
- **`-mcc <mcc_id>`** (repeatable) — walk the MCC's `customer_client` tree and generate for every non-manager, non-hidden, ENABLED child. Each MCC implicitly becomes its own `login_customer_id` for that batch, so you don't need the env var set.

Combine modes in a single invocation:

```sh
go run ./cmd/googleads-tfgen -mcc 1111111111                      # all children of one MCC
go run ./cmd/googleads-tfgen -mcc 1111111111 -mcc 2222222222      # multiple MCCs
go run ./cmd/googleads-tfgen -customer 3333333333                 # one explicit customer
go run ./cmd/googleads-tfgen -customer 3333333333 -mcc 1111111111 # both
```

First-run setup (~5 min, one time):

1. In Google Cloud Console, enable the Google Ads API and configure an API access level for the project.
2. In Google Cloud Console → APIs & Services → Credentials, create an OAuth client of type **Desktop app**. Copy its client ID and secret.
3. Set the OAuth env vars (in `.env` or your shell). Set the login customer only for manager-account access:

```sh
export GOOGLE_ADS_CLIENT_ID=...
export GOOGLE_ADS_CLIENT_SECRET=...
export GOOGLE_ADS_LOGIN_CUSTOMER_ID=...    # optional; your MCC, digits only
```

Google sunset developer tokens on September 9, 2026. `GOOGLE_ADS_DEVELOPER_TOKEN` remains an optional compatibility setting for API versions that still accept the header.

Run:

```sh
go run ./cmd/googleads-tfgen -customer 1234567890 -customer 9876543210
```

On the first run, your browser opens for Google sign-in. The resulting refresh token is cached at `$XDG_CONFIG_HOME/googleads-tfgen/credentials.json` (mode `0600`); later runs reuse it silently.

To switch Google accounts or rotate the token, pass `-relogin`. To skip the browser flow entirely (e.g. in CI), set `GOOGLE_ADS_REFRESH_TOKEN` and the tool will use it directly.

Output:

```
generated/
  1234567890/
    provider.tf
    budgets.tf
    campaigns.tf
    ad_groups.tf
    ad_group_ads.tf
    keywords.tf
    imports.tf
  9876543210/
    ...
```

`generated/` is in `.gitignore`.

## What it captures

| File | Contents |
|---|---|
| `provider.tf` | `terraform { required_providers ... }` + empty `provider "googleads"` block (credentials come from env). |
| `budgets.tf` | `googleads_campaign_budget` per non-removed budget. |
| `text_assets.tf` | `googleads_text_asset` per text asset. Image assets are skipped (their bytes can't round-trip from the API). |
| `conversion_actions.tf` | `googleads_conversion_action` per non-removed conversion action. |
| `labels.tf` | `googleads_label` per non-removed label. |
| `shared_sets.tf` | `googleads_shared_set` per non-removed shared set. |
| `shared_criteria.tf` | `googleads_shared_criterion` per keyword inside a shared set. |
| `customer_negative_criteria.tf` | `googleads_customer_negative_criterion` per account-wide exclusion. |
| `campaigns.tf` | `googleads_campaign` per non-removed campaign, with `campaign_budget_id` resolved to a budget reference and smart-bidding params (target_cpa_micros / target_roas / cpc_bid_*) populated where set. |
| `campaign_shared_sets.tf` | `googleads_campaign_shared_set` per campaign↔shared-set attachment. |
| `campaign_criteria.tf` | `googleads_campaign_criterion` per location / language / device / ad_schedule / ip_block. Other variants skipped. |
| `campaign_labels.tf` | `googleads_campaign_label` per attachment. |
| `ad_groups.tf` | `googleads_ad_group` per non-removed ad group. |
| `ad_group_labels.tf` | `googleads_ad_group_label` per attachment. |
| `ad_group_ads.tf` | `googleads_ad_group_ad` per non-removed **Responsive Search Ad**. Non-RSA ad types are silently skipped. |
| `ad_group_ad_labels.tf` | `googleads_ad_group_ad_label` per attachment. |
| `keywords.tf` | `googleads_ad_group_criterion` per non-removed **keyword** criterion. Non-keyword criteria are skipped here. |
| `ad_group_criterion_labels.tf` | `googleads_ad_group_criterion_label` per attachment. |
| `ad_group_audience_criteria.tf` | `googleads_ad_group_audience_criterion` per user_list / age_range / gender criterion. |
| `asset_groups.tf` | `googleads_asset_group` per non-removed asset group. |
| `asset_group_assets.tf` | `googleads_asset_group_asset` per non-removed link between an asset group and an asset. |
| `imports.tf` | All matching `import { to = ..., id = "customers/.../..." }` blocks. |

## Applying

```sh
cd generated/1234567890
terraform init
terraform plan      # should propose ~0 changes if HCL matches reality
terraform apply     # records imports into state
```

After the first apply, `imports.tf` can be deleted (or moved aside) — its job is done.

## Notes & limits

- Resource labels are derived from numeric IDs (`budget_42`, `campaign_99`), not display names, so re-running the generator after a rename produces stable diffs.
- The generator only covers resource types and variants modeled by this provider. Unsupported criterion and ad variants are skipped; inspect generated comments before importing.
- Image asset bytes cannot be recovered from the Ads API, so image assets are not generated even though the provider can create them from local files.
- If a campaign references a budget that wasn't returned by `ListBudgets` (e.g. it belongs to a different account), the `campaign_budget_id` is written as the raw API resource name string instead of a `googleads_campaign_budget.<label>.id` reference. The import still works.
