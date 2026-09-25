package googleads

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/shenzhencenter/google-ads-pb/common"
	errorspb "github.com/shenzhencenter/google-ads-pb/errors"
)

func TestExtractExemptiblePolicyKeys(t *testing.T) {
	t.Parallel()
	key := &common.PolicyViolationKey{
		PolicyName:    StringPtr("HEALTH_IN_PERSONALIZED_ADS"),
		ViolatingText: StringPtr("psicologa online"),
	}
	err := healthPolicyViolationError(key)
	got := ExtractExemptiblePolicyKeys(err)
	if len(got) != 1 {
		t.Fatalf("len(keys) = %d, want 1", len(got))
	}
	if got[0].GetPolicyName() != "HEALTH_IN_PERSONALIZED_ADS" {
		t.Errorf("policy_name = %q", got[0].GetPolicyName())
	}
	if got[0].GetViolatingText() != "psicologa online" {
		t.Errorf("violating_text = %q", got[0].GetViolatingText())
	}
}

func TestExtractExemptiblePolicyKeys_SkipsNonExemptible(t *testing.T) {
	t.Parallel()
	st := status.New(codes.InvalidArgument, "policy violation")
	gaFailure := &errorspb.GoogleAdsFailure{
		Errors: []*errorspb.GoogleAdsError{{
			Message: "blocked",
			Details: &errorspb.ErrorDetails{
				PolicyViolationDetails: &errorspb.PolicyViolationDetails{
					IsExemptible: false,
					Key: &common.PolicyViolationKey{
						PolicyName: StringPtr("SOME_POLICY"),
					},
				},
			},
		}},
	}
	st, err := st.WithDetails(gaFailure)
	if err != nil {
		t.Fatal(err)
	}
	if keys := ExtractExemptiblePolicyKeys(st.Err()); len(keys) != 0 {
		t.Fatalf("expected no keys, got %d", len(keys))
	}
}

func healthPolicyViolationError(key *common.PolicyViolationKey) error {
	st := status.New(codes.InvalidArgument, "A policy was violated.")
	gaFailure := &errorspb.GoogleAdsFailure{
		Errors: []*errorspb.GoogleAdsError{{
			Message: "A policy was violated.",
			Details: &errorspb.ErrorDetails{
				PolicyViolationDetails: &errorspb.PolicyViolationDetails{
					IsExemptible:       true,
					Key:                proto.Clone(key).(*common.PolicyViolationKey),
					ExternalPolicyName: "HEALTH_IN_PERSONALIZED_ADS",
				},
			},
		}},
	}
	st, err := st.WithDetails(gaFailure)
	if err != nil {
		panic(err)
	}
	return st.Err()
}
