package googleads

import (
	"fmt"
	"strings"
)

// ParseResourceName splits a fully qualified Ads API resource name of the
// form "customers/{cid}/<kind>/{rest}" into (customerID, rest).
// The kind segment is checked against expectedKind for safety.
func ParseResourceName(resourceName, expectedKind string) (customerID, rest string, err error) {
	parts := strings.Split(resourceName, "/")
	if len(parts) != 4 || parts[0] != "customers" || parts[2] != expectedKind {
		return "", "", fmt.Errorf("googleads: resource name %q does not match customers/{id}/%s/{rest}", resourceName, expectedKind)
	}
	if parts[1] == "" || parts[3] == "" {
		return "", "", fmt.Errorf("googleads: resource name %q has empty segment", resourceName)
	}
	return parts[1], parts[3], nil
}

// CustomerOfResourceName returns the customer ID segment of a fully
// qualified Ads API resource name without caring about the kind. Useful
// for cross-customer consistency checks at plan time, where the caller
// only needs to know which customer a resource belongs to.
func CustomerOfResourceName(resourceName string) (string, error) {
	parts := strings.Split(resourceName, "/")
	if len(parts) < 2 || parts[0] != "customers" || parts[1] == "" {
		return "", fmt.Errorf("googleads: resource name %q does not start with customers/{id}/", resourceName)
	}
	return parts[1], nil
}

// BuildResourceName joins a customer ID, kind, and tail (which may itself
// contain "~" for composite IDs such as adGroupAds) into a full resource name.
func BuildResourceName(customerID, kind, tail string) string {
	return fmt.Sprintf("customers/%s/%s/%s", customerID, kind, tail)
}

// CompositeID joins parent~child pairs used by the Ads API for ad_group_ads
// and ad_group_criteria.
func CompositeID(parent, child string) string {
	return parent + "~" + child
}

// SplitComposite reverses CompositeID.
func SplitComposite(s string) (parent, child string, err error) {
	parts := strings.SplitN(s, "~", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("googleads: %q is not a valid composite id", s)
	}
	return parts[0], parts[1], nil
}
