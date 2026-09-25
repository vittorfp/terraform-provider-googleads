package main

import (
	"fmt"
	"strings"

	"github.com/vittorfp/terraform-provider-googleads/internal/googleads"
)

// nameTable resolves a Google Ads resource name (the API's canonical ID,
// `customers/{cid}/<kind>/{rest}`) to its Terraform reference string
// (`<resource_type>.<label>`). We populate it as we emit each resource so
// downstream resources can write proper HCL references.
type nameTable struct {
	refs map[string]string // api resource name → "<resource_type>.<tf_label>"
}

func newNameTable() *nameTable { return &nameTable{refs: map[string]string{}} }

func (t *nameTable) set(resourceName, ref string) { t.refs[resourceName] = ref }

func (t *nameTable) ref(resourceName string) (string, bool) {
	v, ok := t.refs[resourceName]
	return v, ok
}

// Stable, predictable TF identifiers based on the trailing numeric ID. Using
// names from the account would break re-runs whenever a user renames things.

func budgetLabel(resourceName string) string {
	_, id, err := googleads.ParseResourceName(resourceName, "campaignBudgets")
	if err != nil {
		return safeID("budget", resourceName)
	}
	return fmt.Sprintf("budget_%s", id)
}

func campaignLabel(resourceName string) string {
	_, id, err := googleads.ParseResourceName(resourceName, "campaigns")
	if err != nil {
		return safeID("campaign", resourceName)
	}
	return fmt.Sprintf("campaign_%s", id)
}

func adGroupLabel(resourceName string) string {
	_, id, err := googleads.ParseResourceName(resourceName, "adGroups")
	if err != nil {
		return safeID("ad_group", resourceName)
	}
	return fmt.Sprintf("ad_group_%s", id)
}

func adGroupAdLabel(resourceName string) string {
	_, id, err := googleads.ParseResourceName(resourceName, "adGroupAds")
	if err != nil {
		return safeID("ad", resourceName)
	}
	// id is "<ad_group_id>~<ad_id>"
	id = strings.ReplaceAll(id, "~", "_")
	return fmt.Sprintf("ad_%s", id)
}

func keywordLabel(resourceName string) string {
	_, id, err := googleads.ParseResourceName(resourceName, "adGroupCriteria")
	if err != nil {
		return safeID("kw", resourceName)
	}
	id = strings.ReplaceAll(id, "~", "_")
	return fmt.Sprintf("kw_%s", id)
}

// Labels for the resource types added in PRs #2 – #10.

func assetLabel(resourceName string) string {
	_, id, err := googleads.ParseResourceName(resourceName, "assets")
	if err != nil {
		return safeID("asset", resourceName)
	}
	return "asset_" + id
}

func assetGroupLabel(resourceName string) string {
	_, id, err := googleads.ParseResourceName(resourceName, "assetGroups")
	if err != nil {
		return safeID("ag", resourceName)
	}
	return "ag_" + id
}

func assetGroupAssetLabel(resourceName string) string {
	_, id, err := googleads.ParseResourceName(resourceName, "assetGroupAssets")
	if err != nil {
		return safeID("aga", resourceName)
	}
	return "aga_" + strings.ReplaceAll(id, "~", "_")
}

func conversionActionLabel(resourceName string) string {
	_, id, err := googleads.ParseResourceName(resourceName, "conversionActions")
	if err != nil {
		return safeID("ca", resourceName)
	}
	return "ca_" + id
}

func sharedSetLabel(resourceName string) string {
	_, id, err := googleads.ParseResourceName(resourceName, "sharedSets")
	if err != nil {
		return safeID("ss", resourceName)
	}
	return "ss_" + id
}

func sharedCriterionLabel(resourceName string) string {
	_, id, err := googleads.ParseResourceName(resourceName, "sharedCriteria")
	if err != nil {
		return safeID("sc", resourceName)
	}
	return "sc_" + strings.ReplaceAll(id, "~", "_")
}

func campaignSharedSetLabel(resourceName string) string {
	_, id, err := googleads.ParseResourceName(resourceName, "campaignSharedSets")
	if err != nil {
		return safeID("css", resourceName)
	}
	return "css_" + strings.ReplaceAll(id, "~", "_")
}

func customerNegativeCriterionLabel(resourceName string) string {
	_, id, err := googleads.ParseResourceName(resourceName, "customerNegativeCriteria")
	if err != nil {
		return safeID("cnc", resourceName)
	}
	return "cnc_" + id
}

func campaignCriterionLabel(resourceName string) string {
	_, id, err := googleads.ParseResourceName(resourceName, "campaignCriteria")
	if err != nil {
		return safeID("cc", resourceName)
	}
	return "cc_" + strings.ReplaceAll(id, "~", "_")
}

func audienceLabel(resourceName string) string {
	_, id, err := googleads.ParseResourceName(resourceName, "adGroupCriteria")
	if err != nil {
		return safeID("aud", resourceName)
	}
	return "aud_" + strings.ReplaceAll(id, "~", "_")
}

func labelLabelFor(resourceName string) string {
	_, id, err := googleads.ParseResourceName(resourceName, "labels")
	if err != nil {
		return safeID("lbl", resourceName)
	}
	return "lbl_" + id
}

func labelLinkLabel(prefix, kind, resourceName string) string {
	_, id, err := googleads.ParseResourceName(resourceName, kind)
	if err != nil {
		return safeID(prefix, resourceName)
	}
	return prefix + "_" + strings.ReplaceAll(id, "~", "_")
}

// safeID is a last-resort sanitiser for unexpected resource-name shapes.
// It keeps only [A-Za-z0-9_] and prefixes a `prefix_` so the result is a
// valid HCL identifier.
func safeID(prefix, raw string) string {
	var b strings.Builder
	b.WriteString(prefix)
	b.WriteByte('_')
	for _, r := range raw {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}
