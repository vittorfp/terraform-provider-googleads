package googleads

import (
	"context"

	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/shenzhencenter/google-ads-pb/common"
	errorspb "github.com/shenzhencenter/google-ads-pb/errors"
	"github.com/shenzhencenter/google-ads-pb/services"
)

// ExtractExemptiblePolicyKeys scans a raw gRPC error from the Google Ads API
// and returns the PolicyViolationKey entries for violations marked
// is_exemptible. Mirrors scripts/ads/apply.py::_extract_exemptible_policy_keys.
func ExtractExemptiblePolicyKeys(err error) []*common.PolicyViolationKey {
	if err == nil {
		return nil
	}
	st, ok := status.FromError(err)
	if !ok {
		return nil
	}
	unmarshal := proto.UnmarshalOptions{DiscardUnknown: true}
	var keys []*common.PolicyViolationKey
	for _, detail := range st.Proto().GetDetails() {
		var failure errorspb.GoogleAdsFailure
		if err := unmarshal.Unmarshal(detail.GetValue(), &failure); err != nil {
			continue
		}
		for _, e := range failure.GetErrors() {
			pvd := e.GetDetails().GetPolicyViolationDetails()
			if pvd == nil || !pvd.GetIsExemptible() || pvd.GetKey() == nil {
				continue
			}
			keys = append(keys, proto.Clone(pvd.GetKey()).(*common.PolicyViolationKey))
		}
	}
	return keys
}

// mutateAdGroupCriteriaCreate sends a create operation and, on an exemptible
// policy violation, retries once with exempt_policy_violation_keys populated.
func (c *Client) mutateAdGroupCriteriaCreate(
	ctx context.Context,
	customerID string,
	op *services.AdGroupCriterionOperation,
) (string, error) {
	cl, err := c.adGroupCriterionClient(ctx)
	if err != nil {
		return "", err
	}
	resp, err := cl.MutateAdGroupCriteria(ctx, &services.MutateAdGroupCriteriaRequest{
		CustomerId: customerID,
		Operations: []*services.AdGroupCriterionOperation{op},
	})
	if err == nil {
		return resp.GetResults()[0].GetResourceName(), nil
	}
	keys := ExtractExemptiblePolicyKeys(err)
	if len(keys) == 0 {
		return "", WrapAPIError(err)
	}
	retryOp := proto.Clone(op).(*services.AdGroupCriterionOperation)
	retryOp.ExemptPolicyViolationKeys = append(retryOp.ExemptPolicyViolationKeys, keys...)
	resp, err = cl.MutateAdGroupCriteria(ctx, &services.MutateAdGroupCriteriaRequest{
		CustomerId: customerID,
		Operations: []*services.AdGroupCriterionOperation{retryOp},
	})
	if err != nil {
		return "", WrapAPIError(err)
	}
	return resp.GetResults()[0].GetResourceName(), nil
}
