package googleads

import (
	"context"

	pb "github.com/shenzhencenter/google-ads-pb/services"
)

// Fake services for the rest of the Ads API surface this package
// uses. Same pattern as testserver_test.go: each fake records its
// incoming ops and returns synthetic resource names via mintResults.

type fakeSharedSetSvc struct {
	pb.UnimplementedSharedSetServiceServer
	ts *testServer
}

func (f *fakeSharedSetSvc) MutateSharedSets(_ context.Context, req *pb.MutateSharedSetsRequest) (*pb.MutateSharedSetsResponse, error) {
	f.ts.mu.Lock()
	f.ts.sharedSetOps = append(f.ts.sharedSetOps, req.GetOperations()...)
	f.ts.mu.Unlock()
	return &pb.MutateSharedSetsResponse{
		Results: mintResults(req.GetCustomerId(), "sharedSets", len(req.GetOperations()), func(rn string) *pb.MutateSharedSetResult {
			return &pb.MutateSharedSetResult{ResourceName: rn}
		}),
	}, nil
}

type fakeSharedCriterionSvc struct {
	pb.UnimplementedSharedCriterionServiceServer
	ts *testServer
}

func (f *fakeSharedCriterionSvc) MutateSharedCriteria(_ context.Context, req *pb.MutateSharedCriteriaRequest) (*pb.MutateSharedCriteriaResponse, error) {
	f.ts.mu.Lock()
	f.ts.sharedCriterionOps = append(f.ts.sharedCriterionOps, req.GetOperations()...)
	f.ts.mu.Unlock()
	return &pb.MutateSharedCriteriaResponse{
		Results: mintResults(req.GetCustomerId(), "sharedCriteria", len(req.GetOperations()), func(rn string) *pb.MutateSharedCriterionResult {
			return &pb.MutateSharedCriterionResult{ResourceName: rn}
		}),
	}, nil
}

type fakeCampaignSharedSetSvc struct {
	pb.UnimplementedCampaignSharedSetServiceServer
	ts *testServer
}

func (f *fakeCampaignSharedSetSvc) MutateCampaignSharedSets(_ context.Context, req *pb.MutateCampaignSharedSetsRequest) (*pb.MutateCampaignSharedSetsResponse, error) {
	f.ts.mu.Lock()
	f.ts.campaignSharedSetOps = append(f.ts.campaignSharedSetOps, req.GetOperations()...)
	f.ts.mu.Unlock()
	return &pb.MutateCampaignSharedSetsResponse{
		Results: mintResults(req.GetCustomerId(), "campaignSharedSets", len(req.GetOperations()), func(rn string) *pb.MutateCampaignSharedSetResult {
			return &pb.MutateCampaignSharedSetResult{ResourceName: rn}
		}),
	}, nil
}

type fakeCustomerNegativeCriterionSvc struct {
	pb.UnimplementedCustomerNegativeCriterionServiceServer
	ts *testServer
}

func (f *fakeCustomerNegativeCriterionSvc) MutateCustomerNegativeCriteria(_ context.Context, req *pb.MutateCustomerNegativeCriteriaRequest) (*pb.MutateCustomerNegativeCriteriaResponse, error) {
	f.ts.mu.Lock()
	f.ts.customerNegativeCriterionOps = append(f.ts.customerNegativeCriterionOps, req.GetOperations()...)
	f.ts.mu.Unlock()
	return &pb.MutateCustomerNegativeCriteriaResponse{
		Results: mintResults(req.GetCustomerId(), "customerNegativeCriteria", len(req.GetOperations()), func(rn string) *pb.MutateCustomerNegativeCriteriaResult {
			return &pb.MutateCustomerNegativeCriteriaResult{ResourceName: rn}
		}),
	}, nil
}

type fakeCampaignCriterionSvc struct {
	pb.UnimplementedCampaignCriterionServiceServer
	ts *testServer
}

func (f *fakeCampaignCriterionSvc) MutateCampaignCriteria(_ context.Context, req *pb.MutateCampaignCriteriaRequest) (*pb.MutateCampaignCriteriaResponse, error) {
	f.ts.mu.Lock()
	f.ts.campaignCriterionOps = append(f.ts.campaignCriterionOps, req.GetOperations()...)
	f.ts.mu.Unlock()
	return &pb.MutateCampaignCriteriaResponse{
		Results: mintResults(req.GetCustomerId(), "campaignCriteria", len(req.GetOperations()), func(rn string) *pb.MutateCampaignCriterionResult {
			return &pb.MutateCampaignCriterionResult{ResourceName: rn}
		}),
	}, nil
}

type fakeLabelSvc struct {
	pb.UnimplementedLabelServiceServer
	ts *testServer
}

func (f *fakeLabelSvc) MutateLabels(_ context.Context, req *pb.MutateLabelsRequest) (*pb.MutateLabelsResponse, error) {
	f.ts.mu.Lock()
	f.ts.labelOps = append(f.ts.labelOps, req.GetOperations()...)
	f.ts.mu.Unlock()
	return &pb.MutateLabelsResponse{
		Results: mintResults(req.GetCustomerId(), "labels", len(req.GetOperations()), func(rn string) *pb.MutateLabelResult {
			return &pb.MutateLabelResult{ResourceName: rn}
		}),
	}, nil
}

type fakeCampaignLabelSvc struct {
	pb.UnimplementedCampaignLabelServiceServer
	ts *testServer
}

func (f *fakeCampaignLabelSvc) MutateCampaignLabels(_ context.Context, req *pb.MutateCampaignLabelsRequest) (*pb.MutateCampaignLabelsResponse, error) {
	f.ts.mu.Lock()
	f.ts.campaignLabelOps = append(f.ts.campaignLabelOps, req.GetOperations()...)
	f.ts.mu.Unlock()
	return &pb.MutateCampaignLabelsResponse{
		Results: mintResults(req.GetCustomerId(), "campaignLabels", len(req.GetOperations()), func(rn string) *pb.MutateCampaignLabelResult {
			return &pb.MutateCampaignLabelResult{ResourceName: rn}
		}),
	}, nil
}

type fakeAdGroupLabelSvc struct {
	pb.UnimplementedAdGroupLabelServiceServer
	ts *testServer
}

func (f *fakeAdGroupLabelSvc) MutateAdGroupLabels(_ context.Context, req *pb.MutateAdGroupLabelsRequest) (*pb.MutateAdGroupLabelsResponse, error) {
	f.ts.mu.Lock()
	f.ts.adGroupLabelOps = append(f.ts.adGroupLabelOps, req.GetOperations()...)
	f.ts.mu.Unlock()
	return &pb.MutateAdGroupLabelsResponse{
		Results: mintResults(req.GetCustomerId(), "adGroupLabels", len(req.GetOperations()), func(rn string) *pb.MutateAdGroupLabelResult {
			return &pb.MutateAdGroupLabelResult{ResourceName: rn}
		}),
	}, nil
}

type fakeAdGroupAdLabelSvc struct {
	pb.UnimplementedAdGroupAdLabelServiceServer
	ts *testServer
}

func (f *fakeAdGroupAdLabelSvc) MutateAdGroupAdLabels(_ context.Context, req *pb.MutateAdGroupAdLabelsRequest) (*pb.MutateAdGroupAdLabelsResponse, error) {
	f.ts.mu.Lock()
	f.ts.adGroupAdLabelOps = append(f.ts.adGroupAdLabelOps, req.GetOperations()...)
	f.ts.mu.Unlock()
	return &pb.MutateAdGroupAdLabelsResponse{
		Results: mintResults(req.GetCustomerId(), "adGroupAdLabels", len(req.GetOperations()), func(rn string) *pb.MutateAdGroupAdLabelResult {
			return &pb.MutateAdGroupAdLabelResult{ResourceName: rn}
		}),
	}, nil
}

type fakeAdGroupCriterionLabelSvc struct {
	pb.UnimplementedAdGroupCriterionLabelServiceServer
	ts *testServer
}

func (f *fakeAdGroupCriterionLabelSvc) MutateAdGroupCriterionLabels(_ context.Context, req *pb.MutateAdGroupCriterionLabelsRequest) (*pb.MutateAdGroupCriterionLabelsResponse, error) {
	f.ts.mu.Lock()
	f.ts.adGroupCriterionLabelOps = append(f.ts.adGroupCriterionLabelOps, req.GetOperations()...)
	f.ts.mu.Unlock()
	return &pb.MutateAdGroupCriterionLabelsResponse{
		Results: mintResults(req.GetCustomerId(), "adGroupCriterionLabels", len(req.GetOperations()), func(rn string) *pb.MutateAdGroupCriterionLabelResult {
			return &pb.MutateAdGroupCriterionLabelResult{ResourceName: rn}
		}),
	}, nil
}

type fakeCampaignAssetSvc struct {
	pb.UnimplementedCampaignAssetServiceServer
	ts *testServer
}

func (f *fakeCampaignAssetSvc) MutateCampaignAssets(_ context.Context, req *pb.MutateCampaignAssetsRequest) (*pb.MutateCampaignAssetsResponse, error) {
	f.ts.mu.Lock()
	f.ts.campaignAssetOps = append(f.ts.campaignAssetOps, req.GetOperations()...)
	f.ts.mu.Unlock()
	return &pb.MutateCampaignAssetsResponse{
		Results: mintResults(req.GetCustomerId(), "campaignAssets", len(req.GetOperations()), func(rn string) *pb.MutateCampaignAssetResult {
			return &pb.MutateCampaignAssetResult{ResourceName: rn}
		}),
	}, nil
}

type fakeAdGroupAssetSvc struct {
	pb.UnimplementedAdGroupAssetServiceServer
	ts *testServer
}

func (f *fakeAdGroupAssetSvc) MutateAdGroupAssets(_ context.Context, req *pb.MutateAdGroupAssetsRequest) (*pb.MutateAdGroupAssetsResponse, error) {
	f.ts.mu.Lock()
	f.ts.adGroupAssetOps = append(f.ts.adGroupAssetOps, req.GetOperations()...)
	f.ts.mu.Unlock()
	return &pb.MutateAdGroupAssetsResponse{
		Results: mintResults(req.GetCustomerId(), "adGroupAssets", len(req.GetOperations()), func(rn string) *pb.MutateAdGroupAssetResult {
			return &pb.MutateAdGroupAssetResult{ResourceName: rn}
		}),
	}, nil
}

type fakeCustomerAssetSvc struct {
	pb.UnimplementedCustomerAssetServiceServer
	ts *testServer
}

func (f *fakeCustomerAssetSvc) MutateCustomerAssets(_ context.Context, req *pb.MutateCustomerAssetsRequest) (*pb.MutateCustomerAssetsResponse, error) {
	f.ts.mu.Lock()
	f.ts.customerAssetOps = append(f.ts.customerAssetOps, req.GetOperations()...)
	f.ts.mu.Unlock()
	return &pb.MutateCustomerAssetsResponse{
		Results: mintResults(req.GetCustomerId(), "customerAssets", len(req.GetOperations()), func(rn string) *pb.MutateCustomerAssetResult {
			return &pb.MutateCustomerAssetResult{ResourceName: rn}
		}),
	}, nil
}
