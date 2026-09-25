package main

import (
	"bytes"
	"fmt"

	"github.com/vittorfp/terraform-provider-googleads/internal/googleads"
)

// One render function per resource type added in PRs #2 – #10. The
// shape mirrors the originals in generate.go: each writes a per-type
// .tf file body and appends matching `import` blocks to the shared
// imports writer.

func renderTextAssets(customerID string, items []googleads.AssetView, imp *hclWriter) []byte {
	if len(items) == 0 {
		return nil
	}
	buf := &bytes.Buffer{}
	w := &hclWriter{w: buf}
	for _, a := range items {
		label := assetLabel(a.ResourceName)
		w.blockOpen("resource \"googleads_text_asset\" \"%s\"", label)
		w.attrString("  ", "customer_id", customerID)
		if a.Name != "" {
			w.attrString("  ", "name", a.Name)
		}
		w.attrString("  ", "text", a.Text)
		w.blockClose()

		imp.blockOpen("import")
		imp.attrRef("  ", "to", "googleads_text_asset."+label)
		imp.attrString("  ", "id", a.ResourceName)
		imp.blockClose()
	}
	return buf.Bytes()
}

func renderConversionActions(customerID string, items []googleads.ConversionActionView, imp *hclWriter) []byte {
	if len(items) == 0 {
		return nil
	}
	buf := &bytes.Buffer{}
	w := &hclWriter{w: buf}
	for _, c := range items {
		label := conversionActionLabel(c.ResourceName)
		w.blockOpen("resource \"googleads_conversion_action\" \"%s\"", label)
		w.attrString("  ", "customer_id", customerID)
		w.attrString("  ", "name", c.Name)
		w.attrString("  ", "type", c.Type)
		if c.Status != "" && c.Status != "UNSPECIFIED" {
			w.attrString("  ", "status", c.Status)
		}
		if c.Category != "" && c.Category != "UNSPECIFIED" {
			w.attrString("  ", "category", c.Category)
		}
		if c.CountingType != "" && c.CountingType != "UNSPECIFIED" {
			w.attrString("  ", "counting_type", c.CountingType)
		}
		if c.ClickThroughLookbackWindowDays > 0 {
			w.attrInt("  ", "click_through_lookback_window_days", c.ClickThroughLookbackWindowDays)
		}
		if c.ViewThroughLookbackWindowDays > 0 {
			w.attrInt("  ", "view_through_lookback_window_days", c.ViewThroughLookbackWindowDays)
		}
		w.attrBool("  ", "primary_for_goal", c.PrimaryForGoal)
		w.attrBool("  ", "include_in_conversions_metric", c.IncludeInConversionsMetric)
		if c.DefaultValue > 0 {
			w.printf("  default_value = %g\n", c.DefaultValue)
		}
		if c.DefaultCurrencyCode != "" {
			w.attrString("  ", "default_currency_code", c.DefaultCurrencyCode)
		}
		w.attrBool("  ", "always_use_default_value", c.AlwaysUseDefaultValue)
		w.blockClose()

		imp.blockOpen("import")
		imp.attrRef("  ", "to", "googleads_conversion_action."+label)
		imp.attrString("  ", "id", c.ResourceName)
		imp.blockClose()
	}
	return buf.Bytes()
}

func renderSharedSets(customerID string, items []googleads.SharedSetView, imp *hclWriter) []byte {
	if len(items) == 0 {
		return nil
	}
	buf := &bytes.Buffer{}
	w := &hclWriter{w: buf}
	for _, s := range items {
		label := sharedSetLabel(s.ResourceName)
		w.blockOpen("resource \"googleads_shared_set\" \"%s\"", label)
		w.attrString("  ", "customer_id", customerID)
		w.attrString("  ", "name", s.Name)
		w.attrString("  ", "type", s.Type)
		w.blockClose()

		imp.blockOpen("import")
		imp.attrRef("  ", "to", "googleads_shared_set."+label)
		imp.attrString("  ", "id", s.ResourceName)
		imp.blockClose()
	}
	return buf.Bytes()
}

func renderSharedCriteria(customerID string, items []googleads.SharedCriterionView, imp *hclWriter, names *nameTable) []byte {
	if len(items) == 0 {
		return nil
	}
	buf := &bytes.Buffer{}
	w := &hclWriter{w: buf}
	for _, c := range items {
		label := sharedCriterionLabel(c.ResourceName)
		w.blockOpen("resource \"googleads_shared_criterion\" \"%s\"", label)
		w.attrString("  ", "customer_id", customerID)
		if ref, ok := names.ref(c.SharedSet); ok {
			w.attrRef("  ", "shared_set_id", ref+".id")
		} else {
			w.attrString("  ", "shared_set_id", c.SharedSet)
		}
		w.attrString("  ", "keyword_text", c.KeywordText)
		w.attrString("  ", "match_type", c.MatchType)
		w.blockClose()

		imp.blockOpen("import")
		imp.attrRef("  ", "to", "googleads_shared_criterion."+label)
		imp.attrString("  ", "id", c.ResourceName)
		imp.blockClose()
	}
	return buf.Bytes()
}

func renderCampaignSharedSets(customerID string, items []googleads.CampaignSharedSetView, imp *hclWriter, names *nameTable) []byte {
	if len(items) == 0 {
		return nil
	}
	buf := &bytes.Buffer{}
	w := &hclWriter{w: buf}
	for _, x := range items {
		label := campaignSharedSetLabel(x.ResourceName)
		w.blockOpen("resource \"googleads_campaign_shared_set\" \"%s\"", label)
		w.attrString("  ", "customer_id", customerID)
		if ref, ok := names.ref(x.Campaign); ok {
			w.attrRef("  ", "campaign_id", ref+".id")
		} else {
			w.attrString("  ", "campaign_id", x.Campaign)
		}
		if ref, ok := names.ref(x.SharedSet); ok {
			w.attrRef("  ", "shared_set_id", ref+".id")
		} else {
			w.attrString("  ", "shared_set_id", x.SharedSet)
		}
		w.blockClose()

		imp.blockOpen("import")
		imp.attrRef("  ", "to", "googleads_campaign_shared_set."+label)
		imp.attrString("  ", "id", x.ResourceName)
		imp.blockClose()
	}
	return buf.Bytes()
}

func renderCustomerNegativeCriteria(customerID string, items []googleads.CustomerNegativeCriterionView, imp *hclWriter, names *nameTable) []byte {
	if len(items) == 0 {
		return nil
	}
	buf := &bytes.Buffer{}
	w := &hclWriter{w: buf}
	for _, x := range items {
		label := customerNegativeCriterionLabel(x.ResourceName)

		// The API oneof has 9 variants; the provider models 6. If the row
		// is one of the 3 we don't model yet (MOBILE_APP_CATEGORY,
		// YOUTUBE_VIDEO, PLACEMENT_LIST), emit a SKIPPED comment and
		// drop the matching import so terraform plan stays clean.
		// Closes the second half of #41.
		if !cncHasSupportedVariant(x) {
			w.comment(fmt.Sprintf("SKIPPED %s: customer_negative_criterion type %q is not yet modeled by the provider (see issue #41).", x.ResourceName, x.Type))
			w.comment("To remove this exclusion, delete it via the Ads UI or API.")
			w.printf("\n")
			continue
		}

		w.blockOpen("resource \"googleads_customer_negative_criterion\" \"%s\"", label)
		w.attrString("  ", "customer_id", customerID)
		switch {
		case x.PlacementURL != "":
			w.attrString("  ", "placement_url", x.PlacementURL)
		case x.YoutubeChannelID != "":
			w.attrString("  ", "youtube_channel_id", x.YoutubeChannelID)
		case x.MobileApplicationID != "":
			w.attrString("  ", "mobile_application_id", x.MobileApplicationID)
		case x.IPAddress != "":
			w.attrString("  ", "ip_address", x.IPAddress)
		case x.NegativeKeywordListID != "":
			if ref, ok := names.ref(x.NegativeKeywordListID); ok {
				w.attrRef("  ", "negative_keyword_list_id", ref+".id")
			} else {
				w.attrString("  ", "negative_keyword_list_id", x.NegativeKeywordListID)
			}
		case x.ContentLabelType != "":
			w.attrString("  ", "content_label_type", x.ContentLabelType)
		}
		w.blockClose()

		imp.blockOpen("import")
		imp.attrRef("  ", "to", "googleads_customer_negative_criterion."+label)
		imp.attrString("  ", "id", x.ResourceName)
		imp.blockClose()
	}
	return buf.Bytes()
}

// cncHasSupportedVariant returns true when at least one provider-modeled
// variant field is populated on the view. The Ads API oneof exposes 9
// types; the provider currently covers 6. Anything else is unsupported.
func cncHasSupportedVariant(x googleads.CustomerNegativeCriterionView) bool {
	return x.PlacementURL != "" ||
		x.YoutubeChannelID != "" ||
		x.MobileApplicationID != "" ||
		x.IPAddress != "" ||
		x.NegativeKeywordListID != "" ||
		x.ContentLabelType != ""
}

func renderCampaignCriteria(customerID string, items []googleads.CampaignCriterionView, imp *hclWriter, names *nameTable) []byte {
	if len(items) == 0 {
		return nil
	}
	buf := &bytes.Buffer{}
	w := &hclWriter{w: buf}
	for _, x := range items {
		label := campaignCriterionLabel(x.ResourceName)
		w.blockOpen("resource \"googleads_campaign_criterion\" \"%s\"", label)
		w.attrString("  ", "customer_id", customerID)
		if ref, ok := names.ref(x.Campaign); ok {
			w.attrRef("  ", "campaign_id", ref+".id")
		} else {
			w.attrString("  ", "campaign_id", x.Campaign)
		}
		if x.Status != "" && x.Status != "UNSPECIFIED" {
			w.attrString("  ", "status", x.Status)
		}
		if x.BidModifier != 0 {
			w.printf("  bid_modifier = %g\n", float64(x.BidModifier))
		}
		w.attrBool("  ", "negative", x.Negative)
		switch {
		case x.LocationID != "":
			w.attrString("  ", "location_id", x.LocationID)
		case x.LanguageID != "":
			w.attrString("  ", "language_id", x.LanguageID)
		case x.DeviceType != "" && x.DeviceType != "UNSPECIFIED" && x.DeviceType != "UNKNOWN":
			w.attrString("  ", "device_type", x.DeviceType)
		case x.IPAddress != "":
			w.attrString("  ", "ip_address", x.IPAddress)
		case x.AdScheduleDayOfWeek != "" && x.AdScheduleDayOfWeek != "UNSPECIFIED":
			w.attrString("  ", "ad_schedule_day_of_week", x.AdScheduleDayOfWeek)
			w.attrInt("  ", "ad_schedule_start_hour", int64(x.AdScheduleStartHour))
			w.attrInt("  ", "ad_schedule_end_hour", int64(x.AdScheduleEndHour))
			w.attrString("  ", "ad_schedule_start_minute", x.AdScheduleStartMinute)
			w.attrString("  ", "ad_schedule_end_minute", x.AdScheduleEndMinute)
		case x.HasProximity:
			w.printf("  latitude  = %g\n", x.ProximityLatitude)
			w.printf("  longitude = %g\n", x.ProximityLongitude)
			w.printf("  radius    = %g\n", x.ProximityRadius)
			if x.ProximityRadiusUnits != "" && x.ProximityRadiusUnits != "UNSPECIFIED" && x.ProximityRadiusUnits != "UNKNOWN" {
				w.attrString("  ", "radius_units", x.ProximityRadiusUnits)
			}
			if x.ProximityAddressCountry != "" {
				w.attrString("  ", "address_country_code", x.ProximityAddressCountry)
			}
			if x.ProximityAddressCityName != "" {
				w.attrString("  ", "address_city_name", x.ProximityAddressCityName)
			}
		}
		w.blockClose()

		imp.blockOpen("import")
		imp.attrRef("  ", "to", "googleads_campaign_criterion."+label)
		imp.attrString("  ", "id", x.ResourceName)
		imp.blockClose()
	}
	return buf.Bytes()
}

func renderAudienceCriteria(customerID string, items []googleads.AudienceCriterionView, imp *hclWriter, names *nameTable) []byte {
	if len(items) == 0 {
		return nil
	}
	buf := &bytes.Buffer{}
	w := &hclWriter{w: buf}
	for _, x := range items {
		label := audienceLabel(x.ResourceName)
		w.blockOpen("resource \"googleads_ad_group_audience_criterion\" \"%s\"", label)
		w.attrString("  ", "customer_id", customerID)
		if ref, ok := names.ref(x.AdGroup); ok {
			w.attrRef("  ", "ad_group_id", ref+".id")
		} else {
			w.attrString("  ", "ad_group_id", x.AdGroup)
		}
		if x.Status != "" && x.Status != "UNSPECIFIED" {
			w.attrString("  ", "status", x.Status)
		}
		if x.BidModifier != 0 {
			w.printf("  bid_modifier = %g\n", x.BidModifier)
		}
		w.attrBool("  ", "negative", x.Negative)
		switch {
		case x.UserListID != "":
			w.attrString("  ", "user_list_id", x.UserListID)
		case x.AgeRangeType != "" && x.AgeRangeType != "UNSPECIFIED" && x.AgeRangeType != "UNKNOWN":
			w.attrString("  ", "age_range_type", x.AgeRangeType)
		case x.GenderType != "" && x.GenderType != "UNSPECIFIED" && x.GenderType != "UNKNOWN":
			w.attrString("  ", "gender_type", x.GenderType)
		}
		w.blockClose()

		imp.blockOpen("import")
		imp.attrRef("  ", "to", "googleads_ad_group_audience_criterion."+label)
		imp.attrString("  ", "id", x.ResourceName)
		imp.blockClose()
	}
	return buf.Bytes()
}

func renderAssetGroups(customerID string, items []googleads.AssetGroupView, imp *hclWriter, names *nameTable) []byte {
	if len(items) == 0 {
		return nil
	}
	buf := &bytes.Buffer{}
	w := &hclWriter{w: buf}
	for _, g := range items {
		label := assetGroupLabel(g.ResourceName)
		w.blockOpen("resource \"googleads_asset_group\" \"%s\"", label)
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
		w.attrStringList("  ", "final_urls", g.FinalURLs)
		if len(g.FinalMobileURLs) > 0 {
			w.attrStringList("  ", "final_mobile_urls", g.FinalMobileURLs)
		}
		if g.Path1 != "" {
			w.attrString("  ", "path1", g.Path1)
		}
		if g.Path2 != "" {
			w.attrString("  ", "path2", g.Path2)
		}
		w.blockClose()

		imp.blockOpen("import")
		imp.attrRef("  ", "to", "googleads_asset_group."+label)
		imp.attrString("  ", "id", g.ResourceName)
		imp.blockClose()
	}
	return buf.Bytes()
}

func renderAssetGroupAssets(customerID string, items []googleads.AssetGroupAssetView, imp *hclWriter, names *nameTable) []byte {
	if len(items) == 0 {
		return nil
	}
	buf := &bytes.Buffer{}
	w := &hclWriter{w: buf}
	for _, x := range items {
		label := assetGroupAssetLabel(x.ResourceName)
		w.blockOpen("resource \"googleads_asset_group_asset\" \"%s\"", label)
		w.attrString("  ", "customer_id", customerID)
		if ref, ok := names.ref(x.AssetGroup); ok {
			w.attrRef("  ", "asset_group_id", ref+".id")
		} else {
			w.attrString("  ", "asset_group_id", x.AssetGroup)
		}
		if ref, ok := names.ref(x.Asset); ok {
			w.attrRef("  ", "asset_id", ref+".id")
		} else {
			w.attrString("  ", "asset_id", x.Asset)
		}
		w.attrString("  ", "field_type", x.FieldType)
		if x.Status != "" && x.Status != "UNSPECIFIED" {
			w.attrString("  ", "status", x.Status)
		}
		w.blockClose()

		imp.blockOpen("import")
		imp.attrRef("  ", "to", "googleads_asset_group_asset."+label)
		imp.attrString("  ", "id", x.ResourceName)
		imp.blockClose()
	}
	return buf.Bytes()
}

func renderLabels(customerID string, items []googleads.LabelView, imp *hclWriter) []byte {
	if len(items) == 0 {
		return nil
	}
	buf := &bytes.Buffer{}
	w := &hclWriter{w: buf}
	for _, x := range items {
		label := labelLabelFor(x.ResourceName)
		w.blockOpen("resource \"googleads_label\" \"%s\"", label)
		w.attrString("  ", "customer_id", customerID)
		w.attrString("  ", "name", x.Name)
		if x.BackgroundColor != "" {
			w.attrString("  ", "background_color", x.BackgroundColor)
		}
		if x.Description != "" {
			w.attrString("  ", "description", x.Description)
		}
		w.blockClose()

		imp.blockOpen("import")
		imp.attrRef("  ", "to", "googleads_label."+label)
		imp.attrString("  ", "id", x.ResourceName)
		imp.blockClose()
	}
	return buf.Bytes()
}

// renderLabelLinks emits one of the four *_label join resource types.
// Parameterised by the Terraform resource type name, the API kind
// segment, and a labelFn that produces the per-row HCL identifier.
func renderLabelLinks(customerID string, items []googleads.LabelLinkView, imp *hclWriter, names *nameTable, resourceType, kind string, labelFn func(string) string) []byte {
	if len(items) == 0 {
		return nil
	}
	_ = kind // currently unused — kept in signature for future filtering
	buf := &bytes.Buffer{}
	w := &hclWriter{w: buf}
	for _, x := range items {
		label := labelFn(x.ResourceName)
		w.blockOpen("resource \"%s\" \"%s\"", resourceType, label)
		w.attrString("  ", "customer_id", customerID)
		if ref, ok := names.ref(x.Parent); ok {
			w.attrRef("  ", "parent", ref+".id")
		} else {
			w.attrString("  ", "parent", x.Parent)
		}
		if ref, ok := names.ref(x.Label); ok {
			w.attrRef("  ", "label_id", ref+".id")
		} else {
			w.attrString("  ", "label_id", x.Label)
		}
		w.blockClose()

		imp.blockOpen("import")
		imp.attrRef("  ", "to", resourceType+"."+label)
		imp.attrString("  ", "id", x.ResourceName)
		imp.blockClose()
	}
	return buf.Bytes()
}
