# Backlog

Loose list of things worth doing next. Roughly ordered within each section by impact, not deadline. Convert to GitHub Issues if any of these get scheduled.

## Resource coverage

- **Performance Max end-to-end** — the asset layer (PR #2), grouping layer (PR #3), and smart bidding params (PR #4) are in place. A PMax campaign can now be declared in HCL with `advertising_channel_type = "PERFORMANCE_MAX"` + a smart bidding strategy + asset groups. PR #19 added a wire-level unit test that drives the full stack and asserts every Mutate request shape. PR #21 added `googleads_customer_conversion_goal` — the customer-level toggle that makes Smart Bidding actually serve. Still missing: video assets; live-account smoke test.
- ~~`googleads_text_asset`, `googleads_image_asset`~~ — landed in PR #2.
- ~~`googleads_asset_group`, `googleads_asset_group_asset`~~ — landed in PR #3.
- ~~Smart bidding params (TARGET_CPA, TARGET_ROAS, MAXIMIZE_CONVERSIONS with target_cpa, MAXIMIZE_CONVERSION_VALUE with target_roas)~~ — landed in PR #4.
- ~~**Conversion actions** (`googleads_conversion_action`)~~ — landed in PR #5. Covers WEBPAGE / UPLOAD_CLICKS / UPLOAD_CALLS types and the common reporting categories. App / Firebase / GA4 sub-objects, attribution model overrides, and phone-call duration thresholds are still follow-ups.
- ~~**Ad extensions** (`googleads_sitelink_asset`, `googleads_callout_asset`, `googleads_structured_snippet_asset`) + `googleads_campaign_asset` link~~ — landed in PR #14. ~~`googleads_ad_group_asset` and `googleads_customer_asset`~~ landed in PR #25 — extensions can now be attached at any scope. Asset schedule fields (start_date/end_date/ad_schedule_targets) on sitelinks/callouts deferred.
- ~~**Shared sets** (`googleads_shared_set`, `googleads_shared_criterion`, `googleads_campaign_shared_set`)~~ — landed in PR #6. Covers shared NEGATIVE_KEYWORDS (most common). NEGATIVE_PLACEMENTS / brands / webpages / etc. are follow-ups when needed.
- ~~**Customer-level exclusions** (`googleads_customer_negative_criterion`)~~ — landed in PR #7. Covers placement_url, youtube_channel_id, mobile_application_id, ip_address, negative_keyword_list_id (the five most-used variants). Content labels, mobile-app categories, YouTube videos, and placement lists are follow-ups.
- ~~**Audience / demographic criteria** under ad groups~~ — landed in PR #9 as `googleads_ad_group_audience_criterion`. Covers user_list, age_range, gender. Parental status / income range / topic / placement are follow-ups.
- ~~**Campaign criteria** (`googleads_campaign_criterion`)~~ — landed in PR #8. Covers location, language, device, ad_schedule, and ip_block. PR #30 adds the keyword variant (typically paired with `negative = true` for campaign-level negative keywords). Remaining ~28 variants (audience, topic, custom audience, etc.) land later.
- ~~**Labels** (`googleads_label` + 4 link types)~~ — landed in PR #10. Covers campaign, ad_group, ad_group_ad, and ad_group_criterion link variants.
- **Non-RSA ad types**: at minimum App ads and Demand Gen ads. Probably never image/video ad bodies — those should be assets.

## Provider quality

- ~~**APIError → diagnostic field paths**~~ — fixture in PR #23, full rollout in PR #24. Every Create + Update call site across all 26 resources (plus the four label-link variants) now uses `addAPIErrorDiagnostics`. Most pass `nil` for the attrMap, which gives one diagnostic per failing field with the API field path in the detail — already a big improvement on the giant-error-string baseline. Per-resource attrMap to attach diagnostics to specific HCL lines is a future polish; `googleads_campaign_budget` has it as a template.
- ~~**Plan-time cross-customer validation**~~ — landed in PR #26. Each parent-referencing resource (`campaign`, `ad_group`, `ad_group_ad`, `ad_group_criterion`, `ad_group_audience_criterion`, `campaign_criterion`, `asset_group`, `asset_group_asset`, `campaign_shared_set`, `shared_criterion`, `campaign_asset`, the four label-links) now implements `ValidateConfig` and surfaces a per-attribute diagnostic at plan time when a referenced resource name's customer segment differs from `customer_id`. Saves a round trip and points at the exact attribute instead of relying on the API's opaque cross-customer error.
- ~~**Retry policy**~~ — landed in PR #34. New provider attributes `max_retries` (env `GOOGLE_ADS_MAX_RETRIES`) and `retry_backoff_ms` (env `GOOGLE_ADS_RETRY_BACKOFF_MS`) install a gRPC unary interceptor that retries `UNAVAILABLE` / `DEADLINE_EXCEEDED` / `RESOURCE_EXHAUSTED` with exponential backoff capped at 30s. `max_retries = 0` (default) preserves fail-fast behavior. Chained ahead of the concurrency interceptor so a retry doesn't hold a slot during backoff.
- ~~**Protected replace on destructive immutable fields**~~ — landed in PR #28. New provider attribute `allow_destructive_replace` (default `false`, env `GOOGLE_ADS_ALLOW_DESTRUCTIVE_REPLACE`) plus a custom `ProtectedReplace()` plan modifier. Applied to `campaign.advertising_channel_type`, `campaign.bidding_strategy_type`, and `ad_group.type` — changing any of those now errors at plan time instead of silently destroying the resource and losing Smart Bidding learning. Override by flipping the flag for a single run.
- ~~**`removal_policy` (soft | protect)**~~ — landed in PR #29. Each of `campaign`, `ad_group`, `ad_group_ad`, `campaign_budget` gains a `removal_policy` attribute. `soft` (default) keeps current behavior (calls the Ads API Remove mutation → marks REMOVED). `protect` refuses the destroy at apply time, forcing the operator to flip the policy back or delete out-of-band. Safeguard against accidental `terraform destroy` of production resources. Other resources can opt in later by importing the shared helper.
- ~~**Concurrency limiting**~~ — landed in PR #27. New provider attribute `parallelism` (env `GOOGLE_ADS_PARALLELISM`) installs a semaphore-backed unary gRPC interceptor that caps concurrent Ads API requests across all resources sharing the provider. 0 (default) preserves historical unlimited behavior. Stream calls (Search/SearchStream) intentionally bypass the limit — they're rarely concurrent in practice and bounding them would force the resource layer to serialize reads.
- ~~**Service account auth path**~~ — landed in PR #17. New `service_account_json_path` + `impersonate_email` provider attributes; falls back to refresh-token, then ADC. Requires Workspace domain-wide delegation set up on the SA with the adwords scope.
- **Multi-account ergonomics**: document or codify whether to use one provider per account (aliases) or one provider with per-resource `customer_id`. We currently support the latter.

## Generator (`cmd/googleads-tfgen`)

- ~~Cover the resources added in PRs #2–#10~~ — landed in PR #13. The generator now emits 21 file types (provider + ~20 resource families) and resolves cross-resource references for budgets, campaigns, ad groups, RSAs, shared sets, asset groups, and labels.
- **Cover non-keyword criteria** in `keywords.tf` (ad_group_criterion has placement/topic/etc. variants we still skip).
- ~~**Account walker**~~ — landed in PR #18 as the `-mcc <id>` flag (repeatable). Enumerates the MCC's `customer_client` tree (non-manager + ENABLED + non-hidden) and runs the generator for each child. Combines with `-customer` in one invocation.
- **`terraform fmt` the output** before writing — the current spacing is fine but not canonical.
- **Optional human-friendly labels**: `-label-style=slug` produces `campaign_brand_search` instead of `campaign_99123456`. Numeric stays the default for diff stability.
- **Per-campaign file layout** as an alternative to per-resource-type. Big accounts read better when one `.tf` file == one campaign + its ad groups + ads + keywords.
- **`--dry-run`** mode that prints what would be written.
- **Filters**: `-status ENABLED`, `-channel SEARCH`, `-since <date>` to skip irrelevant resources.

## Testing & CI

- ~~**GitHub Actions CI** on every PR~~ — landed in PR #11. `.github/workflows/ci.yml` runs `go build`, `go vet`, `go test -race`, `golangci-lint` (non-blocking for now), and `terraform fmt -check` on examples. `testacc` deliberately not wired up — needs credentials.
- ~~**Mock-server unit tests**~~ — fixture in PR #16, first wave in PR #16, second wave in PR #22 covering text/image/sitelink/callout/snippet/campaign asset, asset_group + asset_group_asset, shared_set + shared_criterion + campaign_shared_set, conversion_action, customer_negative_criterion (5 variants), campaign_criterion (5 variants), ad_group_audience_criterion (3 variants), label + 4 link types. 75 client-layer tests total. `customer_conversion_goal` (PR #21) test still pending; will land alongside that merge.
- **Snapshot test for the generator**: run against a fixture customer (mock server) and assert the seven `.tf` files match committed goldens.
- **Nightly `testacc`** workflow against a dedicated test MCC.

## Docs & DX

- ~~**`tfplugindocs generate`** wired into `make docs`~~ — landed in PR #15. CI verifies `docs/` is in sync on every PR; drift fails the run.
- ~~**`CONTRIBUTING.md`** with local setup, test safety, redaction rules, and PR expectations.~~
- **`docs/development.md`**: how to add a new resource end-to-end (client method → resource file → example → test → docs).
- **A small "import your account in 5 minutes" guide** based on the actual session that produced PR #1.

## Release & publishing

- ~~Release scaffolding~~ — landed in PR #20. Fixed the `formats` → `format` quirk that blocked `goreleaser build` locally. Added `.github/workflows/release.yml` that fires on `v*` tags, runs goreleaser, optionally imports a GPG key when secrets are set. README has the tag-and-push runbook.
- ~~**Private releases through `v0.5.0`** — cross-platform archives, checksums, and the Registry manifest are published by GoReleaser.~~
- ~~**Publish a sanitized `v0.6.0` or later to the Terraform Registry.** Public repo, signing secrets, signed release, and Registry listing completed.~~
- Decide on stability commitments before v1.0.0: schema breaking changes, deprecation policy.

## Cleanup

- **Refactor `applyXxxView`** repetition across resources. Generics could help but it's marginal — only do if a fifth+ resource lands.
- **Fold `cmd/googleads-diag` into `googleads-tfgen` as a subcommand** (`googleads-tfgen diag <cid>`) so users only install one binary.
- **Drop the workaround** for the global `pr-template-check.py` hook once it's been tightened to scope by repo owner.
