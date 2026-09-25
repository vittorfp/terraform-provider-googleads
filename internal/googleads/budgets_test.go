package googleads

import (
	"context"
	"testing"

	pb "github.com/shenzhencenter/google-ads-pb/services"
)

func TestCreateBudget_ShapesTheOperationCorrectly(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)

	rn, err := c.CreateBudget(context.Background(), BudgetInput{
		CustomerID:     "123",
		Name:           "tf-test",
		AmountMicros:   5_000_000,
		DeliveryMethod: "STANDARD",
	})
	if err != nil {
		t.Fatalf("CreateBudget: %v", err)
	}
	if rn != "customers/123/campaignBudgets/1000" {
		t.Fatalf("resource name = %q", rn)
	}

	if got := len(ts.budgetOps); got != 1 {
		t.Fatalf("expected 1 budget op, got %d", got)
	}
	op := ts.budgetOps[0]
	create := op.GetCreate()
	if create == nil {
		t.Fatal("op was not a Create")
	}
	if create.GetName() != "tf-test" {
		t.Errorf("name = %q", create.GetName())
	}
	if create.GetAmountMicros() != 5_000_000 {
		t.Errorf("amount_micros = %d", create.GetAmountMicros())
	}
	if got := create.GetDeliveryMethod().String(); got != "STANDARD" {
		t.Errorf("delivery_method = %q", got)
	}
	if op.GetUpdateMask() != nil {
		t.Error("Create should not carry a FieldMask")
	}
}

func TestUpdateBudget_SendsOnlyChangedFieldsInMask(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)

	rn := "customers/123/campaignBudgets/42"
	err := c.UpdateBudget(context.Background(), rn, BudgetInput{
		Name:         "renamed",
		AmountMicros: 99,
	}, []string{"name"})
	if err != nil {
		t.Fatalf("UpdateBudget: %v", err)
	}

	if got := len(ts.budgetOps); got != 1 {
		t.Fatalf("expected 1 op, got %d", got)
	}
	op := ts.budgetOps[0]
	update := op.GetUpdate()
	if update == nil {
		t.Fatal("op was not an Update")
	}
	if update.GetResourceName() != rn {
		t.Errorf("resource_name = %q", update.GetResourceName())
	}
	if update.GetName() != "renamed" {
		t.Errorf("name = %q", update.GetName())
	}
	// amount_micros wasn't in `paths`, so even though the input carries 99,
	// it must not be written into the proto (otherwise the API would
	// reject or, worse, silently apply it).
	if got := update.GetAmountMicros(); got != 0 {
		t.Errorf("amount_micros should be zero (not in mask), got %d", got)
	}
	if mask := op.GetUpdateMask(); mask == nil || len(mask.GetPaths()) != 1 || mask.GetPaths()[0] != "name" {
		t.Errorf("update mask = %+v", mask)
	}
}

func TestRemoveBudget_SendsTheResourceName(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)

	rn := "customers/123/campaignBudgets/42"
	if err := c.RemoveBudget(context.Background(), rn); err != nil {
		t.Fatalf("RemoveBudget: %v", err)
	}
	if got := len(ts.budgetOps); got != 1 {
		t.Fatalf("expected 1 op, got %d", got)
	}
	op := ts.budgetOps[0]
	if got := op.GetRemove(); got != rn {
		t.Errorf("Remove = %q, want %q", got, rn)
	}
}

func TestGetBudget_MapsRowFieldsToView(t *testing.T) {
	ts := newTestServer(t)
	c := ts.newClient(t)

	rn := "customers/123/campaignBudgets/42"
	row := &pb.GoogleAdsRow{}
	// Fill the row via reflection-free helpers — the fake just returns
	// whatever we set.
	row.CampaignBudget = mustBuildBudgetRow(rn, "renamed", 7_500_000)
	ts.setSearchRow(row)

	view, err := c.GetBudget(context.Background(), rn)
	if err != nil {
		t.Fatalf("GetBudget: %v", err)
	}
	if view.Name != "renamed" {
		t.Errorf("Name = %q", view.Name)
	}
	if view.AmountMicros != 7_500_000 {
		t.Errorf("AmountMicros = %d", view.AmountMicros)
	}
	if view.ResourceName != rn {
		t.Errorf("ResourceName = %q", view.ResourceName)
	}
}
