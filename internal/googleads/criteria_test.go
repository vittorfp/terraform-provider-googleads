package googleads

import (
	"context"
	"testing"

	"github.com/shenzhencenter/google-ads-pb/common"
	"github.com/shenzhencenter/google-ads-pb/resources"
	pb "github.com/shenzhencenter/google-ads-pb/services"
)

func TestCreateCriterion_BuildsKeywordOneof(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)
	neg := true
	_, err := c.CreateCriterion(context.Background(), CriterionInput{
		CustomerID:  "123",
		AdGroup:     "customers/123/adGroups/9",
		KeywordText: "running shoes",
		MatchType:   "EXACT",
		Status:      "ENABLED",
		Negative:    &neg,
	})
	if err != nil {
		t.Fatalf("CreateCriterion: %v", err)
	}
	create := ts.criterionOps[0].GetCreate()
	if got := create.GetAdGroup(); got != "customers/123/adGroups/9" {
		t.Errorf("ad_group = %q", got)
	}
	if !create.GetNegative() {
		t.Error("negative should be true")
	}
	kw, ok := create.GetCriterion().(*resources.AdGroupCriterion_Keyword)
	if !ok {
		t.Fatalf("Criterion should be Keyword, got %T", create.GetCriterion())
	}
	if got := kw.Keyword.GetText(); got != "running shoes" {
		t.Errorf("keyword text = %q", got)
	}
	if got := kw.Keyword.GetMatchType().String(); got != "EXACT" {
		t.Errorf("match_type = %q", got)
	}
}

func TestCreateCriterion_RetriesExemptiblePolicyViolation(t *testing.T) {
	ts := newTestServer(t)
	policyKey := &common.PolicyViolationKey{
		PolicyName:    StringPtr("HEALTH_IN_PERSONALIZED_ADS"),
		ViolatingText: StringPtr("psicologa online"),
	}
	var attempts int
	ts.adGroupCriterionMutate = func(_ context.Context, req *pb.MutateAdGroupCriteriaRequest) (*pb.MutateAdGroupCriteriaResponse, error) {
		attempts++
		op := req.GetOperations()[0]
		if len(op.GetExemptPolicyViolationKeys()) == 0 {
			return nil, healthPolicyViolationError(policyKey)
		}
		return nil, nil
	}
	c := ts.newClient(t)
	rn, err := c.CreateCriterion(context.Background(), CriterionInput{
		CustomerID:  "123",
		AdGroup:     "customers/123/adGroups/9",
		KeywordText: "psicologa online",
		MatchType:   "PHRASE",
		Status:      "ENABLED",
	})
	if err != nil {
		t.Fatalf("CreateCriterion: %v", err)
	}
	if rn == "" {
		t.Fatal("expected resource name")
	}
	if attempts != 2 {
		t.Fatalf("expected 2 mutate calls, got %d", attempts)
	}
	if len(ts.criterionOps) != 1 {
		t.Fatalf("expected 1 recorded op, got %d", len(ts.criterionOps))
	}
	retry := ts.criterionOps[0]
	if len(retry.GetExemptPolicyViolationKeys()) != 1 {
		t.Fatalf("exempt keys = %d, want 1", len(retry.GetExemptPolicyViolationKeys()))
	}
	if got := retry.GetExemptPolicyViolationKeys()[0].GetPolicyName(); got != "HEALTH_IN_PERSONALIZED_ADS" {
		t.Errorf("policy_name = %q", got)
	}
}

func TestUpdateCriterion_MaskScopesUpdates(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)
	rn := "customers/123/adGroupCriteria/9~10"
	bid := int64(800_000)
	in := CriterionInput{
		Status:       "PAUSED",
		CpcBidMicros: &bid,
	}
	if err := c.UpdateCriterion(context.Background(), rn, in, []string{"status"}); err != nil {
		t.Fatalf("UpdateCriterion: %v", err)
	}
	op := ts.criterionOps[0]
	update := op.GetUpdate()
	if update.GetResourceName() != rn {
		t.Errorf("resource_name = %q", update.GetResourceName())
	}
	if got := update.GetStatus().String(); got != "PAUSED" {
		t.Errorf("status = %q", got)
	}
	if got := update.GetCpcBidMicros(); got != 0 {
		t.Errorf("cpc_bid_micros leaked: %d", got)
	}
	if mask := op.GetUpdateMask(); mask == nil || len(mask.GetPaths()) != 1 || mask.GetPaths()[0] != "status" {
		t.Errorf("update mask = %+v", mask)
	}
}
