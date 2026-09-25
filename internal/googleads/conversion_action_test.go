package googleads

import (
	"context"
	"testing"
)

func TestCreateConversionAction_AllFields(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)
	primary, includeInMetric, alwaysDefault := true, true, false
	clickWindow, viewWindow := int64(30), int64(1)
	defaultValue := 50.0
	if _, err := c.CreateConversionAction(context.Background(), ConversionActionInput{
		CustomerID:                     "123",
		Name:                           "Website Purchase",
		Type:                           "WEBPAGE",
		Category:                       "PURCHASE",
		CountingType:                   "ONE_PER_CLICK",
		ClickThroughLookbackWindowDays: &clickWindow,
		ViewThroughLookbackWindowDays:  &viewWindow,
		PrimaryForGoal:                 &primary,
		IncludeInConversionsMetric:     &includeInMetric,
		DefaultValue:                   &defaultValue,
		DefaultCurrencyCode:            "USD",
		AlwaysUseDefaultValue:          &alwaysDefault,
	}); err != nil {
		t.Fatalf("CreateConversionAction: %v", err)
	}
	op := ts.conversionOps[0].GetCreate()
	if op.GetType().String() != "WEBPAGE" {
		t.Errorf("type = %q", op.GetType().String())
	}
	if op.GetCategory().String() != "PURCHASE" {
		t.Errorf("category = %q", op.GetCategory().String())
	}
	if op.GetCountingType().String() != "ONE_PER_CLICK" {
		t.Errorf("counting_type = %q", op.GetCountingType().String())
	}
	if !op.GetPrimaryForGoal() {
		t.Error("primary_for_goal should be true")
	}
	if op.GetClickThroughLookbackWindowDays() != 30 {
		t.Errorf("click_through_lookback = %d", op.GetClickThroughLookbackWindowDays())
	}
	vs := op.GetValueSettings()
	if vs.GetDefaultValue() != 50.0 {
		t.Errorf("default_value = %g", vs.GetDefaultValue())
	}
	if vs.GetDefaultCurrencyCode() != "USD" {
		t.Errorf("default_currency_code = %q", vs.GetDefaultCurrencyCode())
	}
	if vs.GetAlwaysUseDefaultValue() {
		t.Error("always_use_default_value should be false")
	}
}

func TestUpdateConversionAction_ValueSettingsMask(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)
	rn := "customers/123/conversionActions/42"
	dv := 100.0
	in := ConversionActionInput{
		Name:         "renamed",
		DefaultValue: &dv,
	}
	// Update both `name` and a nested value_settings field. The mask
	// must carry the dotted child path verbatim.
	paths := []string{"name", "value_settings.default_value"}
	if err := c.UpdateConversionAction(context.Background(), rn, in, paths); err != nil {
		t.Fatalf("UpdateConversionAction: %v", err)
	}
	op := ts.conversionOps[0]
	gotPaths := op.GetUpdateMask().GetPaths()
	if len(gotPaths) != 2 || gotPaths[1] != "value_settings.default_value" {
		t.Errorf("mask = %v", gotPaths)
	}
	if op.GetUpdate().GetValueSettings().GetDefaultValue() != 100.0 {
		t.Errorf("default_value = %g", op.GetUpdate().GetValueSettings().GetDefaultValue())
	}
}
