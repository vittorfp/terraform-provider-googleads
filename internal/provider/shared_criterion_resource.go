package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vittorfp/terraform-provider-googleads/internal/googleads"
)

var (
	_ resource.Resource                   = (*sharedCriterionResource)(nil)
	_ resource.ResourceWithConfigure      = (*sharedCriterionResource)(nil)
	_ resource.ResourceWithImportState    = (*sharedCriterionResource)(nil)
	_ resource.ResourceWithValidateConfig = (*sharedCriterionResource)(nil)
)

func NewSharedCriterionResource() resource.Resource { return &sharedCriterionResource{} }

type sharedCriterionResource struct {
	client *googleads.Client
}

type sharedCriterionModel struct {
	ID          types.String `tfsdk:"id"`
	CustomerID  types.String `tfsdk:"customer_id"`
	SharedSetID types.String `tfsdk:"shared_set_id"`
	KeywordText types.String `tfsdk:"keyword_text"`
	MatchType   types.String `tfsdk:"match_type"`
}

func (r *sharedCriterionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_shared_criterion"
}

func (r *sharedCriterionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A keyword inside a `googleads_shared_set` of type NEGATIVE_KEYWORDS. The Ads API has no update operation for shared criteria — every field is immutable, so any change forces resource replacement. v1 covers keyword criteria only; placements/brands/etc. land in a future PR.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Resource name (customers/{cid}/sharedCriteria/{shared_set_id}~{criterion_id}).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"customer_id": schema.StringAttribute{
				Required: true, Description: "Google Ads customer ID.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"shared_set_id": schema.StringAttribute{
				Required: true, Description: "Parent shared set resource name. Immutable.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"keyword_text": schema.StringAttribute{
				Required:    true,
				Description: "Keyword text. Immutable.",
				Validators:  []validator.String{runesLengthAtMost(80)},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"match_type": schema.StringAttribute{
				Required:    true,
				Description: "EXACT, PHRASE, or BROAD. Immutable.",
				Validators: []validator.String{
					stringvalidator.OneOf("EXACT", "PHRASE", "BROAD"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *sharedCriterionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerClient(req.ProviderData, &resp.Diagnostics)
}

func (r *sharedCriterionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan sharedCriterionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rn, err := r.client.CreateSharedCriterion(ctx, googleads.SharedCriterionInput{
		CustomerID:  plan.CustomerID.ValueString(),
		SharedSet:   plan.SharedSetID.ValueString(),
		KeywordText: plan.KeywordText.ValueString(),
		MatchType:   plan.MatchType.ValueString(),
	})
	if err != nil {
		addAPIErrorDiagnostics(&resp.Diagnostics, "Failed to create shared criterion", nil, err)
		return
	}
	plan.ID = types.StringValue(rn)
	view, err := r.client.GetSharedCriterion(ctx, rn)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back shared criterion", err.Error())
		return
	}
	applySharedCriterionView(&plan, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *sharedCriterionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state sharedCriterionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	view, err := r.client.GetSharedCriterion(ctx, state.ID.ValueString())
	if err != nil {
		if googleads.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read shared criterion", err.Error())
		return
	}
	applySharedCriterionView(&state, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is unreachable in practice — every attribute is RequiresReplace
// so Terraform always recreates the resource instead of updating. The
// method must still exist to satisfy the resource.Resource interface.
func (r *sharedCriterionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan sharedCriterionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *sharedCriterionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state sharedCriterionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.RemoveSharedCriterion(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to remove shared criterion", err.Error())
	}
}

func (r *sharedCriterionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *sharedCriterionResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cfg sharedCriterionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	validateSameCustomer(&resp.Diagnostics, cfg.CustomerID, map[string]types.String{
		"shared_set_id": cfg.SharedSetID,
	})
}

func applySharedCriterionView(m *sharedCriterionModel, v *googleads.SharedCriterionView) {
	if cid, _, err := googleads.ParseResourceName(v.ResourceName, "sharedCriteria"); err == nil {
		m.CustomerID = types.StringValue(cid)
	}
	m.SharedSetID = types.StringValue(v.SharedSet)
	m.KeywordText = types.StringValue(v.KeywordText)
	m.MatchType = types.StringValue(v.MatchType)
}
