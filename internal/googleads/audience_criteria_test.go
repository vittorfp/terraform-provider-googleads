package googleads

import (
	"context"
	"testing"

	"github.com/shenzhencenter/google-ads-pb/resources"
)

func TestCreateAudienceCriterion_DispatchesVariant(t *testing.T) {
	cases := []struct {
		name string
		in   AudienceCriterionInput
		ok   func(any) bool
	}{
		{
			"user_list",
			AudienceCriterionInput{CustomerID: "123", AdGroup: "customers/123/adGroups/9", UserListID: "customers/123/userLists/1"},
			func(v any) bool { _, ok := v.(*resources.AdGroupCriterion_UserList); return ok },
		},
		{
			"age_range",
			AudienceCriterionInput{CustomerID: "123", AdGroup: "customers/123/adGroups/9", AgeRangeType: "AGE_RANGE_25_34"},
			func(v any) bool { _, ok := v.(*resources.AdGroupCriterion_AgeRange); return ok },
		},
		{
			"gender",
			AudienceCriterionInput{CustomerID: "123", AdGroup: "customers/123/adGroups/9", GenderType: "MALE"},
			func(v any) bool { _, ok := v.(*resources.AdGroupCriterion_Gender); return ok },
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ts := newTestServer(t)
			c := ts.newClient(t)
			if _, err := c.CreateAudienceCriterion(context.Background(), tc.in); err != nil {
				t.Fatalf("Create: %v", err)
			}
			variant := ts.criterionOps[0].GetCreate().GetCriterion()
			if !tc.ok(variant) {
				t.Errorf("variant = %T", variant)
			}
		})
	}
}
