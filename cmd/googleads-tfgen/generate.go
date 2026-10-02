package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vittorfp/terraform-provider-googleads/internal/googleads"
)

// generateForCustomer pulls every supported resource from one customer
// account and emits a folder of HCL files into outDir. The file layout
// is per-resource-type with all imports collected in a single
// imports.tf at the end.
func generateForCustomer(ctx context.Context, client *googleads.Client, customerID, outDir string, foreachThreshold int) (summary, error) {
	s := summary{counts: map[string]int{}}

	// Fetch all the things in series — these are small queries and
	// account-wide rate limits make parallelism counterproductive.
	type fetched struct {
		budgets         []googleads.BudgetView
		campaigns       []googleads.CampaignView
		adGroups        []googleads.AdGroupView
		rsas            []googleads.AdGroupAdView
		keywords        []googleads.CriterionView
		audiences       []googleads.AudienceCriterionView
		assets          []googleads.AssetView
		assetGroups     []googleads.AssetGroupView
		assetGroupLinks []googleads.AssetGroupAssetView
		conversions     []googleads.ConversionActionView
		sharedSets      []googleads.SharedSetView
		sharedCriteria  []googleads.SharedCriterionView
		campShared      []googleads.CampaignSharedSetView
		negatives       []googleads.CustomerNegativeCriterionView
		campCriteria    []googleads.CampaignCriterionView
		labels          []googleads.LabelView
		campLabels      []googleads.LabelLinkView
		agLabels        []googleads.LabelLinkView
		agAdLabels      []googleads.LabelLinkView
		critLabels      []googleads.LabelLinkView
	}
	var f fetched
	var err error
	if f.budgets, err = client.ListBudgets(ctx, customerID); err != nil {
		return s, fmt.Errorf("list budgets: %w", err)
	}
	if f.campaigns, err = client.ListCampaigns(ctx, customerID); err != nil {
		return s, fmt.Errorf("list campaigns: %w", err)
	}
	if f.adGroups, err = client.ListAdGroups(ctx, customerID); err != nil {
		return s, fmt.Errorf("list ad groups: %w", err)
	}
	if f.rsas, err = client.ListResponsiveSearchAds(ctx, customerID); err != nil {
		return s, fmt.Errorf("list RSAs: %w", err)
	}
	if f.keywords, err = client.ListKeywords(ctx, customerID); err != nil {
		return s, fmt.Errorf("list keywords: %w", err)
	}
	if f.audiences, err = client.ListAudienceCriteria(ctx, customerID); err != nil {
		return s, fmt.Errorf("list audience criteria: %w", err)
	}
	if f.assets, err = client.ListTextAssets(ctx, customerID); err != nil {
		return s, fmt.Errorf("list text assets: %w", err)
	}
	if f.assetGroups, err = client.ListAssetGroups(ctx, customerID); err != nil {
		return s, fmt.Errorf("list asset groups: %w", err)
	}
	if f.assetGroupLinks, err = client.ListAssetGroupAssets(ctx, customerID); err != nil {
		return s, fmt.Errorf("list asset group assets: %w", err)
	}
	if f.conversions, err = client.ListConversionActions(ctx, customerID); err != nil {
		return s, fmt.Errorf("list conversion actions: %w", err)
	}
	if f.sharedSets, err = client.ListSharedSets(ctx, customerID); err != nil {
		return s, fmt.Errorf("list shared sets: %w", err)
	}
	if f.sharedCriteria, err = client.ListSharedCriteria(ctx, customerID); err != nil {
		return s, fmt.Errorf("list shared criteria: %w", err)
	}
	if f.campShared, err = client.ListCampaignSharedSets(ctx, customerID); err != nil {
		return s, fmt.Errorf("list campaign shared sets: %w", err)
	}
	if f.negatives, err = client.ListCustomerNegativeCriteria(ctx, customerID); err != nil {
		return s, fmt.Errorf("list customer negative criteria: %w", err)
	}
	if f.campCriteria, err = client.ListCampaignCriteria(ctx, customerID); err != nil {
		return s, fmt.Errorf("list campaign criteria: %w", err)
	}
	if f.labels, err = client.ListLabels(ctx, customerID); err != nil {
		return s, fmt.Errorf("list labels: %w", err)
	}
	if f.campLabels, err = client.ListCampaignLabels(ctx, customerID); err != nil {
		return s, fmt.Errorf("list campaign labels: %w", err)
	}
	if f.agLabels, err = client.ListAdGroupLabels(ctx, customerID); err != nil {
		return s, fmt.Errorf("list ad group labels: %w", err)
	}
	if f.agAdLabels, err = client.ListAdGroupAdLabels(ctx, customerID); err != nil {
		return s, fmt.Errorf("list ad group ad labels: %w", err)
	}
	if f.critLabels, err = client.ListAdGroupCriterionLabels(ctx, customerID); err != nil {
		return s, fmt.Errorf("list ad group criterion labels: %w", err)
	}

	// Stable ordering by resource_name keeps re-runs diff-free.
	sortByResourceName(f.budgets, func(b googleads.BudgetView) string { return b.ResourceName })
	sortByResourceName(f.campaigns, func(c googleads.CampaignView) string { return c.ResourceName })
	sortByResourceName(f.adGroups, func(g googleads.AdGroupView) string { return g.ResourceName })
	sortByResourceName(f.rsas, func(a googleads.AdGroupAdView) string { return a.ResourceName })
	sortByResourceName(f.keywords, func(k googleads.CriterionView) string { return k.ResourceName })
	sortByResourceName(f.audiences, func(a googleads.AudienceCriterionView) string { return a.ResourceName })
	sortByResourceName(f.assets, func(a googleads.AssetView) string { return a.ResourceName })
	sortByResourceName(f.assetGroups, func(g googleads.AssetGroupView) string { return g.ResourceName })
	sortByResourceName(f.assetGroupLinks, func(g googleads.AssetGroupAssetView) string { return g.ResourceName })
	sortByResourceName(f.conversions, func(c googleads.ConversionActionView) string { return c.ResourceName })
	sortByResourceName(f.sharedSets, func(x googleads.SharedSetView) string { return x.ResourceName })
	sortByResourceName(f.sharedCriteria, func(x googleads.SharedCriterionView) string { return x.ResourceName })
	sortByResourceName(f.campShared, func(x googleads.CampaignSharedSetView) string { return x.ResourceName })
	sortByResourceName(f.negatives, func(x googleads.CustomerNegativeCriterionView) string { return x.ResourceName })
	sortByResourceName(f.campCriteria, func(x googleads.CampaignCriterionView) string { return x.ResourceName })
	sortByResourceName(f.labels, func(x googleads.LabelView) string { return x.ResourceName })
	sortByResourceName(f.campLabels, func(x googleads.LabelLinkView) string { return x.ResourceName })
	sortByResourceName(f.agLabels, func(x googleads.LabelLinkView) string { return x.ResourceName })
	sortByResourceName(f.agAdLabels, func(x googleads.LabelLinkView) string { return x.ResourceName })
	sortByResourceName(f.critLabels, func(x googleads.LabelLinkView) string { return x.ResourceName })

	// Populate the name table — order matters: producers before consumers.
	names := newNameTable()
	for _, b := range f.budgets {
		names.set(b.ResourceName, "googleads_campaign_budget."+budgetLabel(b.ResourceName))
	}
	for _, c := range f.campaigns {
		names.set(c.ResourceName, "googleads_campaign."+campaignLabel(c.ResourceName))
	}
	for _, g := range f.adGroups {
		names.set(g.ResourceName, "googleads_ad_group."+adGroupLabel(g.ResourceName))
	}
	for _, a := range f.rsas {
		names.set(a.ResourceName, "googleads_ad_group_ad."+adGroupAdLabel(a.ResourceName))
	}
	for _, k := range f.keywords {
		names.set(k.ResourceName, "googleads_ad_group_criterion."+keywordLabel(k.ResourceName))
	}
	for _, a := range f.audiences {
		names.set(a.ResourceName, "googleads_ad_group_audience_criterion."+audienceLabel(a.ResourceName))
	}
	for _, a := range f.assets {
		names.set(a.ResourceName, "googleads_text_asset."+assetLabel(a.ResourceName))
	}
	for _, g := range f.assetGroups {
		names.set(g.ResourceName, "googleads_asset_group."+assetGroupLabel(g.ResourceName))
	}
	for _, x := range f.sharedSets {
		names.set(x.ResourceName, "googleads_shared_set."+sharedSetLabel(x.ResourceName))
	}
	for _, x := range f.labels {
		names.set(x.ResourceName, "googleads_label."+labelLabelFor(x.ResourceName))
	}

	imports := &bytes.Buffer{}
	importsW := &hclWriter{w: imports}
	importsW.comment(fmt.Sprintf("Generated by googleads-tfgen for customer %s.", customerID))
	importsW.comment("Apply with: terraform init && terraform plan && terraform apply")
	importsW.printf("\n")

	write := func(filename string, body []byte) error {
		return os.WriteFile(filepath.Join(outDir, filename), body, 0o644)
	}

	if err := write("provider.tf", renderProvider(customerID)); err != nil {
		return s, err
	}

	// renderAll dispatches per-resource-type rendering, writes the file if
	// non-empty, and bumps the summary counter.
	renderAll := []struct {
		filename string
		kind     string
		count    int
		buf      []byte
	}{
		{"budgets.tf", "budgets", len(f.budgets), renderBudgets(customerID, f.budgets, importsW, names)},
		{"text_assets.tf", "text_assets", len(f.assets), renderTextAssets(customerID, f.assets, importsW)},
		{"conversion_actions.tf", "conversion_actions", len(f.conversions), renderConversionActions(customerID, f.conversions, importsW)},
		{"labels.tf", "labels", len(f.labels), renderLabels(customerID, f.labels, importsW)},
		{"shared_sets.tf", "shared_sets", len(f.sharedSets), renderSharedSets(customerID, f.sharedSets, importsW)},
		{"shared_criteria.tf", "shared_criteria", len(f.sharedCriteria), renderSharedCriteria(customerID, f.sharedCriteria, importsW, names)},
		{"customer_negative_criteria.tf", "customer_negative_criteria", len(f.negatives), renderCustomerNegativeCriteria(customerID, f.negatives, importsW, names)},
		{"campaigns.tf", "campaigns", len(f.campaigns), renderCampaigns(customerID, f.campaigns, importsW, names)},
		{"campaign_shared_sets.tf", "campaign_shared_sets", len(f.campShared), renderCampaignSharedSets(customerID, f.campShared, importsW, names)},
		{"campaign_criteria.tf", "campaign_criteria", len(f.campCriteria), renderCampaignCriteria(customerID, f.campCriteria, importsW, names)},
		{"campaign_labels.tf", "campaign_labels", len(f.campLabels), renderLabelLinks(customerID, f.campLabels, importsW, names, "googleads_campaign_label", "campaignLabels", func(rn string) string { return "clbl_" + tilde(rn, "campaignLabels") })},
		{"ad_groups.tf", "ad_groups", len(f.adGroups), renderAdGroups(customerID, f.adGroups, importsW, names)},
		{"ad_group_labels.tf", "ad_group_labels", len(f.agLabels), renderLabelLinks(customerID, f.agLabels, importsW, names, "googleads_ad_group_label", "adGroupLabels", func(rn string) string { return "aglbl_" + tilde(rn, "adGroupLabels") })},
		{"ad_group_ads.tf", "ad_group_ads", len(f.rsas), renderAds(customerID, f.rsas, importsW, names)},
		{"ad_group_ad_labels.tf", "ad_group_ad_labels", len(f.agAdLabels), renderLabelLinks(customerID, f.agAdLabels, importsW, names, "googleads_ad_group_ad_label", "adGroupAdLabels", func(rn string) string { return "aalbl_" + tilde(rn, "adGroupAdLabels") })},
		{"keywords.tf", "keywords", len(f.keywords), renderKeywords(customerID, f.keywords, importsW, names, foreachThreshold)},
		{"ad_group_criterion_labels.tf", "ad_group_criterion_labels", len(f.critLabels), renderLabelLinks(customerID, f.critLabels, importsW, names, "googleads_ad_group_criterion_label", "adGroupCriterionLabels", func(rn string) string { return "aclbl_" + tilde(rn, "adGroupCriterionLabels") })},
		{"ad_group_audience_criteria.tf", "ad_group_audience_criteria", len(f.audiences), renderAudienceCriteria(customerID, f.audiences, importsW, names)},
		{"asset_groups.tf", "asset_groups", len(f.assetGroups), renderAssetGroups(customerID, f.assetGroups, importsW, names)},
		{"asset_group_assets.tf", "asset_group_assets", len(f.assetGroupLinks), renderAssetGroupAssets(customerID, f.assetGroupLinks, importsW, names)},
	}
	for _, r := range renderAll {
		if len(r.buf) > 0 {
			if err := write(r.filename, r.buf); err != nil {
				return s, err
			}
			s.counts[r.kind] = r.count
		}
	}

	if importsW.err != nil {
		return s, importsW.err
	}
	if err := write("imports.tf", imports.Bytes()); err != nil {
		return s, err
	}
	return s, nil
}

type summary struct {
	counts map[string]int
}

func (s summary) String() string {
	if len(s.counts) == 0 {
		return "(empty account)"
	}
	keys := make([]string, 0, len(s.counts))
	for k := range s.counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", k, s.counts[k]))
	}
	return strings.Join(parts, " ")
}

// sortByResourceName is a tiny generic helper to keep the orchestrator
// readable. Sorts in-place by the per-item resource name extracted via
// key.
func sortByResourceName[T any](items []T, key func(T) string) {
	sort.Slice(items, func(i, j int) bool { return key(items[i]) < key(items[j]) })
}

// tilde returns the trailing path component of a resource name with
// `~` segments converted to `_`, suitable for embedding in an HCL
// identifier.
func tilde(resourceName, kind string) string {
	_, id, err := googleads.ParseResourceName(resourceName, kind)
	if err != nil {
		return safeID(kind, resourceName)
	}
	return strings.ReplaceAll(id, "~", "_")
}

func renderProvider(customerID string) []byte {
	buf := &bytes.Buffer{}
	w := &hclWriter{w: buf}
	w.comment(fmt.Sprintf("Generated by googleads-tfgen for customer %s.", customerID))
	w.printf("\n")
	w.blockOpen("terraform")
	w.printf("  required_providers {\n")
	w.printf("    googleads = {\n")
	w.printf("      source  = \"vittorfp/googleads\"\n")
	w.printf("      version = \"~> 0.6\"\n")
	w.printf("    }\n")
	w.printf("  }\n")
	w.blockClose()
	w.blockOpen("provider \"googleads\"")
	w.comment("Credentials are read from GOOGLE_ADS_* env vars.")
	w.blockClose()
	return buf.Bytes()
}

func renderBudgets(customerID string, items []googleads.BudgetView, imp *hclWriter, _ *nameTable) []byte {
	if len(items) == 0 {
		return nil
	}
	buf := &bytes.Buffer{}
	w := &hclWriter{w: buf}
	for _, b := range items {
		label := budgetLabel(b.ResourceName)
		w.blockOpen("resource \"googleads_campaign_budget\" \"%s\"", label)
		w.attrString("  ", "customer_id", customerID)
		w.attrString("  ", "name", b.Name)
		w.attrInt("  ", "amount_micros", b.AmountMicros)
		if b.DeliveryMethod != "" && b.DeliveryMethod != "UNSPECIFIED" {
			w.attrString("  ", "delivery_method", b.DeliveryMethod)
		}
		w.attrBool("  ", "explicitly_shared", b.ExplicitlyShared)
		w.blockClose()

		imp.blockOpen("import")
		imp.attrRef("  ", "to", "googleads_campaign_budget."+label)
		imp.attrString("  ", "id", b.ResourceName)
		imp.blockClose()
	}
	return buf.Bytes()
}

func renderCampaigns(customerID string, items []googleads.CampaignView, imp *hclWriter, names *nameTable) []byte {
	if len(items) == 0 {
		return nil
	}
	buf := &bytes.Buffer{}
	w := &hclWriter{w: buf}
	for _, c := range items {
		label := campaignLabel(c.ResourceName)
		w.blockOpen("resource \"googleads_campaign\" \"%s\"", label)
		w.attrString("  ", "customer_id", customerID)
		w.attrString("  ", "name", c.Name)
		w.attrString("  ", "advertising_channel_type", c.AdvertisingChannelType)
		if ref, ok := names.ref(c.CampaignBudget); ok {
			w.attrRef("  ", "campaign_budget_id", ref+".id")
		} else {
			// Budget lives outside our scope (e.g. shared across accounts) —
			// fall back to the raw resource name so the import still works.
			w.attrString("  ", "campaign_budget_id", c.CampaignBudget)
		}
		if c.Status != "" && c.Status != "UNSPECIFIED" {
			w.attrString("  ", "status", c.Status)
		}
		if c.BiddingStrategyType != "" && c.BiddingStrategyType != "UNSPECIFIED" {
			w.attrString("  ", "bidding_strategy_type", c.BiddingStrategyType)
		}
		if c.ContainsEuPoliticalAdvertising != "" && c.ContainsEuPoliticalAdvertising != "UNSPECIFIED" && c.ContainsEuPoliticalAdvertising != "UNKNOWN" {
			w.attrString("  ", "contains_eu_political_advertising", c.ContainsEuPoliticalAdvertising)
		}
		if c.TargetCpaMicros > 0 {
			w.attrInt("  ", "target_cpa_micros", c.TargetCpaMicros)
		}
		if c.TargetRoas > 0 {
			w.printf("  target_roas = %g\n", c.TargetRoas)
		}
		if c.CpcBidCeilingMicros > 0 {
			w.attrInt("  ", "cpc_bid_ceiling_micros", c.CpcBidCeilingMicros)
		}
		if c.CpcBidFloorMicros > 0 {
			w.attrInt("  ", "cpc_bid_floor_micros", c.CpcBidFloorMicros)
		}
		// Always emit network_settings and geo_target_type_setting so the
		// generated HCL pins the live values in Terraform. Without this, a
		// future API default change could quietly drift the campaign away
		// from what it was when imported.
		ns := c.NetworkSettings
		w.printf("  network_settings = {\n")
		w.attrBool("    ", "target_google_search", ns.TargetGoogleSearch)
		w.attrBool("    ", "target_search_network", ns.TargetSearchNetwork)
		w.attrBool("    ", "target_content_network", ns.TargetContentNetwork)
		w.attrBool("    ", "target_partner_search_network", ns.TargetPartnerSearchNetwork)
		w.printf("  }\n")
		g := c.GeoTargetTypeSetting
		if (g.PositiveGeoTargetType != "" && g.PositiveGeoTargetType != "UNSPECIFIED") ||
			(g.NegativeGeoTargetType != "" && g.NegativeGeoTargetType != "UNSPECIFIED") {
			w.printf("  geo_target_type_setting = {\n")
			if g.PositiveGeoTargetType != "" && g.PositiveGeoTargetType != "UNSPECIFIED" {
				w.attrString("    ", "positive_geo_target_type", g.PositiveGeoTargetType)
			}
			if g.NegativeGeoTargetType != "" && g.NegativeGeoTargetType != "UNSPECIFIED" {
				w.attrString("    ", "negative_geo_target_type", g.NegativeGeoTargetType)
			}
			w.printf("  }\n")
		}
		w.blockClose()

		imp.blockOpen("import")
		imp.attrRef("  ", "to", "googleads_campaign."+label)
		imp.attrString("  ", "id", c.ResourceName)
		imp.blockClose()
	}
	return buf.Bytes()
}

func renderAdGroups(customerID string, items []googleads.AdGroupView, imp *hclWriter, names *nameTable) []byte {
	if len(items) == 0 {
		return nil
	}
	buf := &bytes.Buffer{}
	w := &hclWriter{w: buf}
	for _, g := range items {
		label := adGroupLabel(g.ResourceName)
		w.blockOpen("resource \"googleads_ad_group\" \"%s\"", label)
		w.attrString("  ", "customer_id", customerID)
		if ref, ok := names.ref(g.Campaign); ok {
			w.attrRef("  ", "campaign_id", ref+".id")
		} else {
			w.attrString("  ", "campaign_id", g.Campaign)
		}
		w.attrString("  ", "name", g.Name)
		if g.Status != "" && g.Status != "UNSPECIFIED" {
			w.attrString("  ", "status", g.Status)
		}
		if g.Type != "" && g.Type != "UNSPECIFIED" {
			w.attrString("  ", "type", g.Type)
		}
		if g.CpcBidMicros > 0 {
			w.attrInt("  ", "cpc_bid_micros", g.CpcBidMicros)
		}
		w.blockClose()

		imp.blockOpen("import")
		imp.attrRef("  ", "to", "googleads_ad_group."+label)
		imp.attrString("  ", "id", g.ResourceName)
		imp.blockClose()
	}
	return buf.Bytes()
}

func renderAds(customerID string, items []googleads.AdGroupAdView, imp *hclWriter, names *nameTable) []byte {
	if len(items) == 0 {
		return nil
	}
	buf := &bytes.Buffer{}
	w := &hclWriter{w: buf}
	for _, a := range items {
		label := adGroupAdLabel(a.ResourceName)
		w.blockOpen("resource \"googleads_ad_group_ad\" \"%s\"", label)
		w.attrString("  ", "customer_id", customerID)
		if ref, ok := names.ref(a.AdGroup); ok {
			w.attrRef("  ", "ad_group_id", ref+".id")
		} else {
			w.attrString("  ", "ad_group_id", a.AdGroup)
		}
		if a.Status != "" && a.Status != "UNSPECIFIED" {
			w.attrString("  ", "status", a.Status)
		}
		w.attrStringList("  ", "final_urls", a.FinalURLs)
		w.attrStringList("  ", "headlines", a.Headlines)
		w.attrStringList("  ", "descriptions", a.Descriptions)
		if a.Path1 != "" {
			w.attrString("  ", "path1", a.Path1)
		}
		if a.Path2 != "" {
			w.attrString("  ", "path2", a.Path2)
		}
		w.blockClose()

		imp.blockOpen("import")
		imp.attrRef("  ", "to", "googleads_ad_group_ad."+label)
		imp.attrString("  ", "id", a.ResourceName)
		imp.blockClose()
	}
	return buf.Bytes()
}

func renderKeywords(customerID string, items []googleads.CriterionView, imp *hclWriter, names *nameTable, foreachThreshold int) []byte {
	if len(items) == 0 {
		return nil
	}
	buf := &bytes.Buffer{}
	w := &hclWriter{w: buf}

	// Group by ad_group_id so we can decide per-group whether to emit
	// as a for_each block or per-keyword blocks. Items within a group
	// preserve their original (already-sorted) order; group iteration
	// order is sorted by ad_group resource name for stable diffs.
	groups := map[string][]googleads.CriterionView{}
	var groupOrder []string
	for _, k := range items {
		if _, seen := groups[k.AdGroup]; !seen {
			groupOrder = append(groupOrder, k.AdGroup)
		}
		groups[k.AdGroup] = append(groups[k.AdGroup], k)
	}
	sort.Strings(groupOrder)

	for _, ag := range groupOrder {
		ks := groups[ag]
		if foreachThreshold > 0 && len(ks) >= foreachThreshold {
			renderKeywordForEach(w, imp, customerID, ag, ks, names)
		} else {
			renderKeywordBlocks(w, imp, customerID, ks, names)
		}
	}
	return buf.Bytes()
}

// renderKeywordBlocks emits one resource block + one import block per
// keyword — the original (pre-for_each) shape. Used for ad groups
// with few keywords.
func renderKeywordBlocks(w, imp *hclWriter, customerID string, ks []googleads.CriterionView, names *nameTable) {
	for _, k := range ks {
		label := keywordLabel(k.ResourceName)
		w.blockOpen("resource \"googleads_ad_group_criterion\" \"%s\"", label)
		w.attrString("  ", "customer_id", customerID)
		if ref, ok := names.ref(k.AdGroup); ok {
			w.attrRef("  ", "ad_group_id", ref+".id")
		} else {
			w.attrString("  ", "ad_group_id", k.AdGroup)
		}
		w.attrString("  ", "keyword_text", k.KeywordText)
		w.attrString("  ", "match_type", k.MatchType)
		if k.Status != "" && k.Status != "UNSPECIFIED" {
			w.attrString("  ", "status", k.Status)
		}
		w.attrBool("  ", "negative", k.Negative)
		if k.CpcBidMicros > 0 {
			w.attrInt("  ", "cpc_bid_micros", k.CpcBidMicros)
		}
		w.blockClose()

		imp.blockOpen("import")
		imp.attrRef("  ", "to", "googleads_ad_group_criterion."+label)
		imp.attrString("  ", "id", k.ResourceName)
		imp.blockClose()
	}
}

// renderKeywordForEach emits a single resource block with for_each
// over a local map, plus one import block per existing keyword that
// references the indexed instance. Used when an ad group has many
// keywords, so the .tf file stays readable.
func renderKeywordForEach(w, imp *hclWriter, customerID, adGroup string, ks []googleads.CriterionView, names *nameTable) {
	// Resource label derives from the ad group label so consumers can
	// reason about which group's keywords this block covers.
	groupLabel := adGroupLabel(adGroup)
	resLabel := "kws_" + groupLabel

	// locals { kws_<group> = { "<id>" = { keyword_text=..., match_type=..., ... } } }
	w.blockOpen("locals")
	w.printf("  %s = {\n", resLabel)
	for _, k := range ks {
		key := keywordForEachKey(k)
		w.printf("    %q = {\n", key)
		w.printf("      keyword_text = %q\n", k.KeywordText)
		w.printf("      match_type   = %q\n", k.MatchType)
		w.printf("      negative     = %t\n", k.Negative)
		if k.Status != "" && k.Status != "UNSPECIFIED" {
			w.printf("      status       = %q\n", k.Status)
		}
		if k.CpcBidMicros > 0 {
			w.printf("      cpc_bid_micros = %d\n", k.CpcBidMicros)
		}
		w.printf("    }\n")
	}
	w.printf("  }\n")
	w.blockClose()

	// The resource block — one for the whole group.
	w.blockOpen("resource \"googleads_ad_group_criterion\" \"%s\"", resLabel)
	w.printf("  for_each = local.%s\n", resLabel)
	w.attrString("  ", "customer_id", customerID)
	if ref, ok := names.ref(adGroup); ok {
		w.attrRef("  ", "ad_group_id", ref+".id")
	} else {
		w.attrString("  ", "ad_group_id", adGroup)
	}
	w.printf("  keyword_text = each.value.keyword_text\n")
	w.printf("  match_type   = each.value.match_type\n")
	w.printf("  negative     = each.value.negative\n")
	w.printf("  status       = try(each.value.status, null)\n")
	w.printf("  cpc_bid_micros = try(each.value.cpc_bid_micros, null)\n")
	w.blockClose()

	// One import per existing keyword, addressing the indexed instance.
	for _, k := range ks {
		key := keywordForEachKey(k)
		imp.blockOpen("import")
		imp.printf("  to = googleads_ad_group_criterion.%s[%q]\n", resLabel, key)
		imp.attrString("  ", "id", k.ResourceName)
		imp.blockClose()
	}
}

// keywordForEachKey builds the for_each map key for a keyword. The key
// must be unique within the ad group and stable across regenerations.
// "<text>__<match_type>" satisfies both — the API also enforces that
// (text, match_type) is unique per ad group.
func keywordForEachKey(k googleads.CriterionView) string {
	return k.KeywordText + "__" + k.MatchType
}
