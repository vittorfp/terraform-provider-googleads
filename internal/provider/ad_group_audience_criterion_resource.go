package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vittorfp/terraform-provider-googleads/internal/googleads"
)

var (
	_ resource.Resource                     = (*adGroupAudienceCriterionResource)(nil)
	_ resource.ResourceWithConfigure        = (*adGroupAudienceCriterionResource)(nil)
	_ resource.ResourceWithImportState      = (*adGroupAudienceCriterionResource)(nil)
	_ resource.ResourceWithConfigValidators = (*adGroupAudienceCriterionResource)(nil)
	_ resource.ResourceWithValidateConfig   = (*adGroupAudienceCriterionResource)(nil)
)

func NewAdGroupAudienceCriterionResource() resource.Resource {
	return &adGroupAudienceCriterionResource{}
}

type adGroupAudienceCriterionResource struct {
	client *googleads.Client
}

type adGroupAudienceCriterionModel struct {
	ID          types.String  `tfsdk:"id"`
	CustomerID  types.String  `tfsdk:"customer_id"`
	AdGroupID   types.String  `tfsdk:"ad_group_id"`
	Status      types.String  `tfsdk:"status"`
	BidModifier types.Float64 `tfsdk:"bid_modifier"`
	Negative    types.Bool    `tfsdk:"negative"`

	UserListID   types.String `tfsdk:"user_list_id"`
	AgeRangeType types.String `tfsdk:"age_range_type"`
	GenderType   types.String `tfsdk:"gender_type"`
}

func (r *adGroupAudienceCriterionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ad_group_audience_criterion"
}

func (r *adGroupAudienceCriterionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	immutable := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Audience or demographic criterion attached to an ad group. Exactly one variant must be set (user_list_id, age_range_type, or gender_type). Status and bid_modifier are mutable; everything else is immutable. Use `googleads_ad_group_criterion` for keyword criteria.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Resource name (customers/{cid}/adGroupCriteria/{ag_id}~{criterion_id}).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"customer_id": schema.StringAttribute{
				Required: true, Description: "Google Ads customer ID.",
				PlanModifiers: immutable,
			},
			"ad_group_id": schema.StringAttribute{
				Required: true, Description: "Parent ad group resource name. Immutable.",
				PlanModifiers: immutable,
			},
			"status": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "ENABLED, PAUSED, or REMOVED. Mutable.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"bid_modifier": schema.Float64Attribute{
				Optional: true, Computed: true,
				Description:   "Bid multiplier in range 0.1–10.0. Mutable.",
				PlanModifiers: []planmodifier.Float64{float64planmodifier.UseStateForUnknown()},
			},
			"negative": schema.BoolAttribute{
				Optional: true, Computed: true,
				Description:   "If true, exclude the audience rather than target it. Immutable.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"user_list_id": schema.StringAttribute{
				Optional: true,
				Description: "Target/exclude a user list (remarketing list, customer match, similar audience, etc.). Resource name of a user_list (customers/{cid}/userLists/{id}). Immutable.",
				PlanModifiers: immutable,
			},
			"age_range_type": schema.StringAttribute{
				Optional: true,
				Description: "AGE_RANGE_18_24 / 25_34 / 35_44 / 45_54 / 55_64 / 65_UP / UNDETERMINED. Immutable.",
				Validators: []validator.String{
					stringvalidator.OneOf(
						"AGE_RANGE_18_24", "AGE_RANGE_25_34", "AGE_RANGE_35_44",
						"AGE_RANGE_45_54", "AGE_RANGE_55_64", "AGE_RANGE_65_UP",
						"AGE_RANGE_UNDETERMINED",
					),
				},
				PlanModifiers: immutable,
			},
			"gender_type": schema.StringAttribute{
				Optional: true,
				Description: "MALE, FEMALE, or UNDETERMINED. Immutable.",
				Validators: []validator.String{
					stringvalidator.OneOf("MALE", "FEMALE", "UNDETERMINED"),
				},
				PlanModifiers: immutable,
			},
		},
	}
}

func (r *adGroupAudienceCriterionResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(
			path.MatchRoot("user_list_id"),
			path.MatchRoot("age_range_type"),
			path.MatchRoot("gender_type"),
		),
	}
}

func (r *adGroupAudienceCriterionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerClient(req.ProviderData, &resp.Diagnostics)
}

func (r *adGroupAudienceCriterionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan adGroupAudienceCriterionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	in := buildAudienceCriterionInput(&plan)
	rn, err := r.client.CreateAudienceCriterion(ctx, in)
	if err != nil {
		addAPIErrorDiagnostics(&resp.Diagnostics, "Failed to create audience criterion", nil, err)
		return
	}
	plan.ID = types.StringValue(rn)
	view, err := r.client.GetAudienceCriterion(ctx, rn)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back audience criterion", err.Error())
		return
	}
	applyAudienceCriterionView(&plan, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *adGroupAudienceCriterionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state adGroupAudienceCriterionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	view, err := r.client.GetAudienceCriterion(ctx, state.ID.ValueString())
	if err != nil {
		if googleads.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read audience criterion", err.Error())
		return
	}
	applyAudienceCriterionView(&state, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *adGroupAudienceCriterionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state adGroupAudienceCriterionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var paths []string
	if !plan.Status.Equal(state.Status) {
		paths = append(paths, "status")
	}
	if !plan.BidModifier.Equal(state.BidModifier) {
		paths = append(paths, "bid_modifier")
	}
	if len(paths) > 0 {
		in := buildAudienceCriterionInput(&plan)
		if err := r.client.UpdateAudienceCriterion(ctx, state.ID.ValueString(), in, paths); err != nil {
			addAPIErrorDiagnostics(&resp.Diagnostics, "Failed to update audience criterion", nil, err)
			return
		}
	}
	view, err := r.client.GetAudienceCriterion(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back audience criterion", err.Error())
		return
	}
	plan.ID = state.ID
	applyAudienceCriterionView(&plan, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *adGroupAudienceCriterionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state adGroupAudienceCriterionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.RemoveAudienceCriterion(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to remove audience criterion", err.Error())
	}
}

func (r *adGroupAudienceCriterionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *adGroupAudienceCriterionResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cfg adGroupAudienceCriterionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	validateSameCustomer(&resp.Diagnostics, cfg.CustomerID, map[string]types.String{
		"ad_group_id":  cfg.AdGroupID,
		"user_list_id": cfg.UserListID,
	})
}

func buildAudienceCriterionInput(m *adGroupAudienceCriterionModel) googleads.AudienceCriterionInput {
	in := googleads.AudienceCriterionInput{
		CustomerID:   m.CustomerID.ValueString(),
		AdGroup:      m.AdGroupID.ValueString(),
		Status:       m.Status.ValueString(),
		UserListID:   m.UserListID.ValueString(),
		AgeRangeType: m.AgeRangeType.ValueString(),
		GenderType:   m.GenderType.ValueString(),
	}
	if !m.BidModifier.IsNull() && !m.BidModifier.IsUnknown() {
		v := m.BidModifier.ValueFloat64()
		in.BidModifier = &v
	}
	if !m.Negative.IsNull() && !m.Negative.IsUnknown() {
		v := m.Negative.ValueBool()
		in.Negative = &v
	}
	return in
}

func applyAudienceCriterionView(m *adGroupAudienceCriterionModel, v *googleads.AudienceCriterionView) {
	if cid, _, err := googleads.ParseResourceName(v.ResourceName, "adGroupCriteria"); err == nil {
		m.CustomerID = types.StringValue(cid)
	}
	m.AdGroupID = types.StringValue(v.AdGroup)
	m.Status = types.StringValue(v.Status)
	m.BidModifier = types.Float64Value(v.BidModifier)
	m.Negative = types.BoolValue(v.Negative)
	m.UserListID = nullableString(v.UserListID)
	m.AgeRangeType = nullableEnum(v.AgeRangeType)
	m.GenderType = nullableEnum(v.GenderType)
}
