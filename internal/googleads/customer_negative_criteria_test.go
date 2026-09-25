package googleads

import (
	"context"
	"testing"

	"github.com/shenzhencenter/google-ads-pb/resources"
)

func TestCreateCustomerNegativeCriterion_DispatchesVariant(t *testing.T) {
	cases := []struct {
		name string
		in   CustomerNegativeCriterionInput
		want any // proto variant type
	}{
		{"placement", CustomerNegativeCriterionInput{CustomerID: "123", PlacementURL: "http://badsite"}, (*resources.CustomerNegativeCriterion_Placement)(nil)},
		{"ip", CustomerNegativeCriterionInput{CustomerID: "123", IPAddress: "1.2.3.4"}, (*resources.CustomerNegativeCriterion_IpBlock)(nil)},
		{"youtube", CustomerNegativeCriterionInput{CustomerID: "123", YoutubeChannelID: "UC123"}, (*resources.CustomerNegativeCriterion_YoutubeChannel)(nil)},
		{"app", CustomerNegativeCriterionInput{CustomerID: "123", MobileApplicationID: "1-476943146"}, (*resources.CustomerNegativeCriterion_MobileApplication)(nil)},
		{"shared-set", CustomerNegativeCriterionInput{CustomerID: "123", NegativeKeywordListID: "customers/123/sharedSets/1"}, (*resources.CustomerNegativeCriterion_NegativeKeywordList)(nil)},
		{"content-label", CustomerNegativeCriterionInput{CustomerID: "123", ContentLabelType: "SEXUALLY_SUGGESTIVE"}, (*resources.CustomerNegativeCriterion_ContentLabel)(nil)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ts := newTestServer(t)
			c := ts.newClient(t)
			if _, err := c.CreateCustomerNegativeCriterion(context.Background(), tc.in); err != nil {
				t.Fatalf("Create: %v", err)
			}
			op := ts.customerNegativeCriterionOps[0].GetCreate()
			gotKind := op.GetCriterion()
			switch tc.want.(type) {
			case *resources.CustomerNegativeCriterion_Placement:
				if _, ok := gotKind.(*resources.CustomerNegativeCriterion_Placement); !ok {
					t.Errorf("got %T", gotKind)
				}
			case *resources.CustomerNegativeCriterion_IpBlock:
				if _, ok := gotKind.(*resources.CustomerNegativeCriterion_IpBlock); !ok {
					t.Errorf("got %T", gotKind)
				}
			case *resources.CustomerNegativeCriterion_YoutubeChannel:
				if _, ok := gotKind.(*resources.CustomerNegativeCriterion_YoutubeChannel); !ok {
					t.Errorf("got %T", gotKind)
				}
			case *resources.CustomerNegativeCriterion_MobileApplication:
				if _, ok := gotKind.(*resources.CustomerNegativeCriterion_MobileApplication); !ok {
					t.Errorf("got %T", gotKind)
				}
			case *resources.CustomerNegativeCriterion_NegativeKeywordList:
				if _, ok := gotKind.(*resources.CustomerNegativeCriterion_NegativeKeywordList); !ok {
					t.Errorf("got %T", gotKind)
				}
			case *resources.CustomerNegativeCriterion_ContentLabel:
				cl, ok := gotKind.(*resources.CustomerNegativeCriterion_ContentLabel)
				if !ok {
					t.Errorf("got %T", gotKind)
					return
				}
				// Wire-shape: enum reaches the proto as the named value,
				// not the default UNSPECIFIED.
				if got := cl.ContentLabel.GetType().String(); got != "SEXUALLY_SUGGESTIVE" {
					t.Errorf("content label type = %q, want SEXUALLY_SUGGESTIVE", got)
				}
			}
		})
	}
}
