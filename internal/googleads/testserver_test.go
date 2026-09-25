package googleads

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"

	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	pb "github.com/shenzhencenter/google-ads-pb/services"
)

const bufSize = 1024 * 1024

// testServer is an in-process gRPC server that stubs the Ads API
// services this package talks to. Each registered fake records the
// requests it receives so tests can assert on the wire-level payload
// (field masks, oneof variants, immutable fields surviving updates,
// etc.). Search responses are programmable via setRows().
type testServer struct {
	t  *testing.T
	mu sync.Mutex

	listener *bufconn.Listener
	server   *grpc.Server

	// Mutate captures, in arrival order.
	budgetOps                   []*pb.CampaignBudgetOperation
	campaignOps                 []*pb.CampaignOperation
	adGroupOps                  []*pb.AdGroupOperation
	adGroupAdOps                []*pb.AdGroupAdOperation
	criterionOps                []*pb.AdGroupCriterionOperation
	conversionOps               []*pb.ConversionActionOperation
	assetOps                    []*pb.AssetOperation
	assetGroupOps               []*pb.AssetGroupOperation
	assetGroupAssetOps          []*pb.AssetGroupAssetOperation
	sharedSetOps                []*pb.SharedSetOperation
	sharedCriterionOps          []*pb.SharedCriterionOperation
	campaignSharedSetOps        []*pb.CampaignSharedSetOperation
	customerNegativeCriterionOps []*pb.CustomerNegativeCriterionOperation
	campaignCriterionOps        []*pb.CampaignCriterionOperation
	labelOps                    []*pb.LabelOperation
	campaignLabelOps            []*pb.CampaignLabelOperation
	adGroupLabelOps             []*pb.AdGroupLabelOperation
	adGroupAdLabelOps           []*pb.AdGroupAdLabelOperation
	adGroupCriterionLabelOps    []*pb.AdGroupCriterionLabelOperation
	campaignAssetOps            []*pb.CampaignAssetOperation
	adGroupAssetOps             []*pb.AdGroupAssetOperation
	customerAssetOps            []*pb.CustomerAssetOperation

	// Programmable Search response. Tests set the next row to return.
	searchRow *pb.GoogleAdsRow

	// Optional hook for AdGroupCriterionService.MutateAdGroupCriteria. Return
	// (nil, err) to fail; (non-nil, nil) to short-circuit success; (nil, nil)
	// to fall through to the default stub behaviour.
	adGroupCriterionMutate func(context.Context, *pb.MutateAdGroupCriteriaRequest) (*pb.MutateAdGroupCriteriaResponse, error)
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()
	ts := &testServer{
		t:        t,
		listener: bufconn.Listen(bufSize),
		server:   grpc.NewServer(),
	}
	pb.RegisterCampaignBudgetServiceServer(ts.server, &fakeBudgetSvc{ts: ts})
	pb.RegisterCampaignServiceServer(ts.server, &fakeCampaignSvc{ts: ts})
	pb.RegisterAdGroupServiceServer(ts.server, &fakeAdGroupSvc{ts: ts})
	pb.RegisterAdGroupAdServiceServer(ts.server, &fakeAdGroupAdSvc{ts: ts})
	pb.RegisterAdGroupCriterionServiceServer(ts.server, &fakeAdGroupCriterionSvc{ts: ts})
	pb.RegisterConversionActionServiceServer(ts.server, &fakeConversionSvc{ts: ts})
	pb.RegisterAssetServiceServer(ts.server, &fakeAssetSvc{ts: ts})
	pb.RegisterAssetGroupServiceServer(ts.server, &fakeAssetGroupSvc{ts: ts})
	pb.RegisterAssetGroupAssetServiceServer(ts.server, &fakeAssetGroupAssetSvc{ts: ts})
	pb.RegisterSharedSetServiceServer(ts.server, &fakeSharedSetSvc{ts: ts})
	pb.RegisterSharedCriterionServiceServer(ts.server, &fakeSharedCriterionSvc{ts: ts})
	pb.RegisterCampaignSharedSetServiceServer(ts.server, &fakeCampaignSharedSetSvc{ts: ts})
	pb.RegisterCustomerNegativeCriterionServiceServer(ts.server, &fakeCustomerNegativeCriterionSvc{ts: ts})
	pb.RegisterCampaignCriterionServiceServer(ts.server, &fakeCampaignCriterionSvc{ts: ts})
	pb.RegisterLabelServiceServer(ts.server, &fakeLabelSvc{ts: ts})
	pb.RegisterCampaignLabelServiceServer(ts.server, &fakeCampaignLabelSvc{ts: ts})
	pb.RegisterAdGroupLabelServiceServer(ts.server, &fakeAdGroupLabelSvc{ts: ts})
	pb.RegisterAdGroupAdLabelServiceServer(ts.server, &fakeAdGroupAdLabelSvc{ts: ts})
	pb.RegisterAdGroupCriterionLabelServiceServer(ts.server, &fakeAdGroupCriterionLabelSvc{ts: ts})
	pb.RegisterCampaignAssetServiceServer(ts.server, &fakeCampaignAssetSvc{ts: ts})
	pb.RegisterAdGroupAssetServiceServer(ts.server, &fakeAdGroupAssetSvc{ts: ts})
	pb.RegisterCustomerAssetServiceServer(ts.server, &fakeCustomerAssetSvc{ts: ts})
	pb.RegisterGoogleAdsServiceServer(ts.server, &fakeGoogleAdsSvc{ts: ts})
	go func() {
		if err := ts.server.Serve(ts.listener); err != nil {
			// Server.Stop() makes Serve return an error; ignore once
			// the test has finished.
			t.Logf("test gRPC server: %v", err)
		}
	}()
	t.Cleanup(func() {
		ts.server.Stop()
		_ = ts.listener.Close()
	})
	return ts
}

// newClient returns a *Client wired to talk to ts via bufconn. No auth
// interceptors are installed — Ads-API-side header validation isn't
// what these tests cover.
func (ts *testServer) newClient(t *testing.T) *Client {
	t.Helper()
	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return ts.listener.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial bufconn: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return newClientFromOpts([]option.ClientOption{option.WithGRPCConn(conn)})
}

// setSearchRow programs the next (and only) row returned by Search /
// SearchOne. Tests call this before invoking GetXxx.
func (ts *testServer) setSearchRow(row *pb.GoogleAdsRow) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.searchRow = row
}

// --- fake services -----------------------------------------------------------

type fakeBudgetSvc struct {
	pb.UnimplementedCampaignBudgetServiceServer
	ts *testServer
}

func (f *fakeBudgetSvc) MutateCampaignBudgets(_ context.Context, req *pb.MutateCampaignBudgetsRequest) (*pb.MutateCampaignBudgetsResponse, error) {
	f.ts.mu.Lock()
	f.ts.budgetOps = append(f.ts.budgetOps, req.GetOperations()...)
	f.ts.mu.Unlock()
	return &pb.MutateCampaignBudgetsResponse{
		Results: mintResults(req.GetCustomerId(), "campaignBudgets", len(req.GetOperations()), func(rn string) *pb.MutateCampaignBudgetResult {
			return &pb.MutateCampaignBudgetResult{ResourceName: rn}
		}),
	}, nil
}

type fakeCampaignSvc struct {
	pb.UnimplementedCampaignServiceServer
	ts *testServer
}

func (f *fakeCampaignSvc) MutateCampaigns(_ context.Context, req *pb.MutateCampaignsRequest) (*pb.MutateCampaignsResponse, error) {
	f.ts.mu.Lock()
	f.ts.campaignOps = append(f.ts.campaignOps, req.GetOperations()...)
	f.ts.mu.Unlock()
	return &pb.MutateCampaignsResponse{
		Results: mintResults(req.GetCustomerId(), "campaigns", len(req.GetOperations()), func(rn string) *pb.MutateCampaignResult {
			return &pb.MutateCampaignResult{ResourceName: rn}
		}),
	}, nil
}

type fakeAdGroupSvc struct {
	pb.UnimplementedAdGroupServiceServer
	ts *testServer
}

func (f *fakeAdGroupSvc) MutateAdGroups(_ context.Context, req *pb.MutateAdGroupsRequest) (*pb.MutateAdGroupsResponse, error) {
	f.ts.mu.Lock()
	f.ts.adGroupOps = append(f.ts.adGroupOps, req.GetOperations()...)
	f.ts.mu.Unlock()
	return &pb.MutateAdGroupsResponse{
		Results: mintResults(req.GetCustomerId(), "adGroups", len(req.GetOperations()), func(rn string) *pb.MutateAdGroupResult {
			return &pb.MutateAdGroupResult{ResourceName: rn}
		}),
	}, nil
}

type fakeAdGroupAdSvc struct {
	pb.UnimplementedAdGroupAdServiceServer
	ts *testServer
}

func (f *fakeAdGroupAdSvc) MutateAdGroupAds(_ context.Context, req *pb.MutateAdGroupAdsRequest) (*pb.MutateAdGroupAdsResponse, error) {
	f.ts.mu.Lock()
	f.ts.adGroupAdOps = append(f.ts.adGroupAdOps, req.GetOperations()...)
	f.ts.mu.Unlock()
	return &pb.MutateAdGroupAdsResponse{
		Results: mintResults(req.GetCustomerId(), "adGroupAds", len(req.GetOperations()), func(rn string) *pb.MutateAdGroupAdResult {
			return &pb.MutateAdGroupAdResult{ResourceName: rn}
		}),
	}, nil
}

type fakeAdGroupCriterionSvc struct {
	pb.UnimplementedAdGroupCriterionServiceServer
	ts *testServer
}

func (f *fakeAdGroupCriterionSvc) MutateAdGroupCriteria(ctx context.Context, req *pb.MutateAdGroupCriteriaRequest) (*pb.MutateAdGroupCriteriaResponse, error) {
	f.ts.mu.Lock()
	hook := f.ts.adGroupCriterionMutate
	f.ts.mu.Unlock()
	if hook != nil {
		resp, err := hook(ctx, req)
		if err != nil {
			return nil, err
		}
		if resp != nil {
			return resp, nil
		}
	}
	f.ts.mu.Lock()
	f.ts.criterionOps = append(f.ts.criterionOps, req.GetOperations()...)
	f.ts.mu.Unlock()
	return &pb.MutateAdGroupCriteriaResponse{
		Results: mintResults(req.GetCustomerId(), "adGroupCriteria", len(req.GetOperations()), func(rn string) *pb.MutateAdGroupCriterionResult {
			return &pb.MutateAdGroupCriterionResult{ResourceName: rn}
		}),
	}, nil
}

type fakeConversionSvc struct {
	pb.UnimplementedConversionActionServiceServer
	ts *testServer
}

func (f *fakeConversionSvc) MutateConversionActions(_ context.Context, req *pb.MutateConversionActionsRequest) (*pb.MutateConversionActionsResponse, error) {
	f.ts.mu.Lock()
	f.ts.conversionOps = append(f.ts.conversionOps, req.GetOperations()...)
	f.ts.mu.Unlock()
	return &pb.MutateConversionActionsResponse{
		Results: mintResults(req.GetCustomerId(), "conversionActions", len(req.GetOperations()), func(rn string) *pb.MutateConversionActionResult {
			return &pb.MutateConversionActionResult{ResourceName: rn}
		}),
	}, nil
}

type fakeAssetSvc struct {
	pb.UnimplementedAssetServiceServer
	ts *testServer
}

func (f *fakeAssetSvc) MutateAssets(_ context.Context, req *pb.MutateAssetsRequest) (*pb.MutateAssetsResponse, error) {
	f.ts.mu.Lock()
	f.ts.assetOps = append(f.ts.assetOps, req.GetOperations()...)
	f.ts.mu.Unlock()
	return &pb.MutateAssetsResponse{
		Results: mintResults(req.GetCustomerId(), "assets", len(req.GetOperations()), func(rn string) *pb.MutateAssetResult {
			return &pb.MutateAssetResult{ResourceName: rn}
		}),
	}, nil
}

type fakeAssetGroupSvc struct {
	pb.UnimplementedAssetGroupServiceServer
	ts *testServer
}

func (f *fakeAssetGroupSvc) MutateAssetGroups(_ context.Context, req *pb.MutateAssetGroupsRequest) (*pb.MutateAssetGroupsResponse, error) {
	f.ts.mu.Lock()
	f.ts.assetGroupOps = append(f.ts.assetGroupOps, req.GetOperations()...)
	f.ts.mu.Unlock()
	return &pb.MutateAssetGroupsResponse{
		Results: mintResults(req.GetCustomerId(), "assetGroups", len(req.GetOperations()), func(rn string) *pb.MutateAssetGroupResult {
			return &pb.MutateAssetGroupResult{ResourceName: rn}
		}),
	}, nil
}

type fakeAssetGroupAssetSvc struct {
	pb.UnimplementedAssetGroupAssetServiceServer
	ts *testServer
}

func (f *fakeAssetGroupAssetSvc) MutateAssetGroupAssets(_ context.Context, req *pb.MutateAssetGroupAssetsRequest) (*pb.MutateAssetGroupAssetsResponse, error) {
	f.ts.mu.Lock()
	f.ts.assetGroupAssetOps = append(f.ts.assetGroupAssetOps, req.GetOperations()...)
	f.ts.mu.Unlock()
	return &pb.MutateAssetGroupAssetsResponse{
		Results: mintResults(req.GetCustomerId(), "assetGroupAssets", len(req.GetOperations()), func(rn string) *pb.MutateAssetGroupAssetResult {
			return &pb.MutateAssetGroupAssetResult{ResourceName: rn}
		}),
	}, nil
}

type fakeGoogleAdsSvc struct {
	pb.UnimplementedGoogleAdsServiceServer
	ts *testServer
}

func (f *fakeGoogleAdsSvc) Search(_ context.Context, _ *pb.SearchGoogleAdsRequest) (*pb.SearchGoogleAdsResponse, error) {
	f.ts.mu.Lock()
	defer f.ts.mu.Unlock()
	if f.ts.searchRow == nil {
		return &pb.SearchGoogleAdsResponse{}, nil
	}
	return &pb.SearchGoogleAdsResponse{Results: []*pb.GoogleAdsRow{f.ts.searchRow}}, nil
}

// mintResults builds the canonical N synthetic resource names that a
// Mutate response would return for N operations.
func mintResults[T any](customerID, kind string, n int, makeResult func(string) T) []T {
	results := make([]T, n)
	for i := 0; i < n; i++ {
		results[i] = makeResult(fmt.Sprintf("customers/%s/%s/%d", customerID, kind, 1000+i))
	}
	return results
}
