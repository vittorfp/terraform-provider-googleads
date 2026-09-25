package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/vittorfp/terraform-provider-googleads/internal/googleads"
)

// keywordSet builds N keyword views all hung off the same ad group,
// with unique (text, match_type) tuples so the for_each key collision
// test passes too.
func keywordSet(adGroup string, n int) []googleads.CriterionView {
	out := make([]googleads.CriterionView, n)
	for i := 0; i < n; i++ {
		out[i] = googleads.CriterionView{
			ResourceName: "customers/123/adGroupCriteria/9~" + itoa(1000+i),
			AdGroup:      adGroup,
			KeywordText:  "kw_" + itoa(i),
			MatchType:    "EXACT",
			Status:       "ENABLED",
		}
	}
	return out
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var d []byte
	for i > 0 {
		d = append([]byte{byte('0' + i%10)}, d...)
		i /= 10
	}
	return string(d)
}

func renderResult(t *testing.T, items []googleads.CriterionView, threshold int) (resourcesTF, importsTF string) {
	t.Helper()
	importsBuf := &bytes.Buffer{}
	importsW := &hclWriter{w: importsBuf}
	resources := renderKeywords("123", items, importsW, &nameTable{}, threshold)
	return string(resources), importsBuf.String()
}

func TestRenderKeywords_BelowThresholdUsesBlocks(t *testing.T) {
	res, imp := renderResult(t, keywordSet("customers/123/adGroups/9", 3), 10)
	if strings.Contains(res, "for_each") {
		t.Errorf("3 keywords should NOT use for_each, got:\n%s", res)
	}
	// One resource block per keyword.
	if got := strings.Count(res, "resource \"googleads_ad_group_criterion\""); got != 3 {
		t.Errorf("resource blocks = %d, want 3", got)
	}
	// One import block per keyword.
	if got := strings.Count(imp, "import {"); got != 3 {
		t.Errorf("import blocks = %d, want 3", got)
	}
}

func TestRenderKeywords_AtThresholdUsesForEach(t *testing.T) {
	res, imp := renderResult(t, keywordSet("customers/123/adGroups/9", 12), 10)
	if !strings.Contains(res, "for_each = local.kws_") {
		t.Errorf("12 keywords at threshold 10 should use for_each, got:\n%s", res)
	}
	// One resource block total (the for_each one), not 12.
	if got := strings.Count(res, "resource \"googleads_ad_group_criterion\""); got != 1 {
		t.Errorf("resource blocks = %d, want 1 (the for_each)", got)
	}
	// One locals block with all 12 entries (each entry starts with
	// a quoted key followed by " = {").
	if got := strings.Count(res, "__EXACT\" = {"); got != 12 {
		t.Errorf("entries in locals = %d, want 12", got)
	}
	// Imports still per-keyword, but indexed.
	if got := strings.Count(imp, "googleads_ad_group_criterion.kws_"); got != 12 {
		t.Errorf("indexed imports = %d, want 12", got)
	}
	if !strings.Contains(imp, `["kw_0__EXACT"]`) {
		t.Errorf("import key not in expected shape:\n%s", imp)
	}
}

func TestRenderKeywords_ThresholdZeroDisablesForEach(t *testing.T) {
	res, _ := renderResult(t, keywordSet("customers/123/adGroups/9", 50), 0)
	if strings.Contains(res, "for_each") {
		t.Errorf("threshold 0 should disable for_each entirely, got for_each in output")
	}
}

func TestRenderKeywords_MixedAdGroupsDispatchIndependently(t *testing.T) {
	var all []googleads.CriterionView
	all = append(all, keywordSet("customers/123/adGroups/1", 3)...)  // below threshold
	all = append(all, keywordSet("customers/123/adGroups/2", 12)...) // at threshold
	res, _ := renderResult(t, all, 10)
	// AdGroup /1 → 3 blocks, AdGroup /2 → 1 for_each block
	if got := strings.Count(res, "for_each = local."); got != 1 {
		t.Errorf("for_each blocks = %d, want 1 (only the big group)", got)
	}
}

// renderCNCResult runs renderCustomerNegativeCriteria and returns the
// resource HCL + the matching imports buffer.
func renderCNCResult(t *testing.T, items []googleads.CustomerNegativeCriterionView) (resourcesTF, importsTF string) {
	t.Helper()
	importsBuf := &bytes.Buffer{}
	importsW := &hclWriter{w: importsBuf}
	resources := renderCustomerNegativeCriteria("123", items, importsW, &nameTable{})
	return string(resources), importsBuf.String()
}

func TestRenderCustomerNegativeCriteria_ContentLabelEmitsField(t *testing.T) {
	items := []googleads.CustomerNegativeCriterionView{{
		ResourceName:     "customers/123/customerNegativeCriteria/300090179",
		Type:             "CONTENT_LABEL",
		ContentLabelType: "SEXUALLY_SUGGESTIVE",
	}}
	res, imp := renderCNCResult(t, items)
	if !strings.Contains(res, `content_label_type = "SEXUALLY_SUGGESTIVE"`) {
		t.Errorf("expected content_label_type attr, got:\n%s", res)
	}
	if !strings.Contains(imp, "googleads_customer_negative_criterion.cnc_300090179") {
		t.Errorf("expected import block, got:\n%s", imp)
	}
}

// Reproduction of issue #41 second half: when a CNC type is not modeled
// by the provider (here: MOBILE_APP_CATEGORY → all variant fields empty
// on the View), tfgen MUST NOT emit an empty resource block (it would
// fail ExactlyOneOf at plan time) and MUST NOT emit the import block.
func TestRenderCustomerNegativeCriteria_UnsupportedVariantIsSkipped(t *testing.T) {
	items := []googleads.CustomerNegativeCriterionView{
		{
			// supported — should emit a normal resource.
			ResourceName: "customers/123/customerNegativeCriteria/1",
			Type:         "IP_BLOCK",
			IPAddress:    "1.2.3.4",
		},
		{
			// unsupported — only customer_id would be written. Must skip.
			ResourceName: "customers/123/customerNegativeCriteria/2",
			Type:         "MOBILE_APP_CATEGORY",
		},
	}
	res, imp := renderCNCResult(t, items)
	if got := strings.Count(res, "resource \"googleads_customer_negative_criterion\""); got != 1 {
		t.Errorf("resource blocks = %d, want 1 (the supported one), got:\n%s", got, res)
	}
	if !strings.Contains(res, "SKIPPED") {
		t.Errorf("expected SKIPPED comment for unsupported type, got:\n%s", res)
	}
	if strings.Contains(imp, "/customerNegativeCriteria/2") {
		t.Errorf("import for unsupported type leaked, got:\n%s", imp)
	}
	if !strings.Contains(imp, "/customerNegativeCriteria/1") {
		t.Errorf("import for supported type missing, got:\n%s", imp)
	}
}
